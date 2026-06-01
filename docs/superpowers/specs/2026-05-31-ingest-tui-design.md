# Ingest TUI -- Design Spec

**Date:** 2026-05-31
**Scope:** `backend/cmd/ingest/` only. No changes to functionality, flags, or output reports.

## Goal

Replace `log.Printf` progress output in the ingest command with a Docker-style TUI:
active workers shown as fixed rows with per-file upload ticks; completed species scroll
up above them with a pass/fail indicator; full error reports printed as plain text after
the TUI exits.

## File Structure

Split `main.go` into three files, all in `backend/cmd/ingest/`:

| File | Responsibility |
|------|---------------|
| `main.go` | CLI flags, env loading, pre-TUI setup (taxonomy, species list, filters), launches TUI |
| `ingest.go` | `ingestSpecies`, `fetchAndUpload`, helpers -- pure logic, no TUI imports |
| `tui.go` | Bubble Tea model, messages, `Init`/`Update`/`View`, Lipgloss styles |

Pre-TUI steps (taxonomy fetch, species list, filter summary) print simple lines to
stderr. The TUI starts when the worker pool kicks off and exits automatically when all
workers finish.

## Dependencies

Add to `backend/go.mod`:

- `github.com/charmbracelet/bubbletea`
- `github.com/charmbracelet/bubbles`
- `github.com/charmbracelet/lipgloss`

## Worker Slot IDs

Change the semaphore from `chan struct{}` to `chan int` (values `0..workers-1`). Each
goroutine receives its slot ID from the channel and passes it through to all `send`
calls, so the TUI knows which row to update.

## Message Types

All defined in `tui.go`. Goroutines call `send func(tea.Msg)` (passed as a parameter)
instead of `log.Printf`.

```go
type speciesStartedMsg struct{ workerID int; code, name string }
type fetchStartedMsg   struct{ workerID int; key string }   // HTTP GET started
type uploadStartedMsg  struct{ workerID int; key string }   // R2 PUT started
type uploadDoneMsg     struct{ workerID int; key string; err error }
type speciesDoneMsg    struct {
    workerID             int
    code, name          string
    recordings, images  int
    failures            []string
}
type allDoneMsg struct{}  // sent after wg.Wait() → triggers tea.Quit
type tickMsg    time.Time // sent every 100ms for elapsed timer
```

`fetchAndUpload` is split into observable steps: after the HTTP response is received
(before the R2 upload), it calls `send(uploadStartedMsg{...})`. This gives the
two-phase fetch→upload tick per file.

## Model

```go
type model struct {
    workers   []workerState  // len = --workers flag value; indexed by workerID
    completed []completedItem
    done, total int
    start     time.Time
}

type workerState struct {
    name    string
    uploads []uploadItem
}

type uploadItem struct {
    key    string
    status uploadStatus  // waiting | fetching | uploading | done | failed
    err    error
}

type completedItem struct {
    name              string
    recordings, images int
    failed            bool
}
```

## Rendering

Three sections, top to bottom:

```
  312/623  ██████████████░░░░░░░░░░░░░░░░  0:04:23

  ✓ American Robin         4 rec  3 img
  ✗ Bushtit                0 rec  3 img  partial failure
  ✓ California Scrub-Jay   2 rec  2 img
  · · ·

  ─────────────────────────────────────────────────
  [1] Steller's Jay
      · recordings/stejay/XC123.mp3   ↓ fetching
      · recordings/stejay/XC456.mp3   ↑ uploading
      · images/stejay/ML789.jpg       ✓
      · images/stejay/ML790.jpg       · waiting
  [2] Spotted Towhee
      · recordings/spotow/XC111.mp3   ✓
      · images/spotow/ML222.jpg       ↑ uploading
```

**Header:** `done/total` + Bubbles `progress.Model` bar + elapsed time (updated by
`tickMsg` every 100ms).

**Completed list:** all finished species, trimmed to the last N lines that fit in the
terminal above the divider. ✓ in green, ✗ in red (dimmed name).

**Active workers:** fixed N rows (one per `--workers` slot), each with indented upload
items. Status symbols: `·` waiting, `↓` fetching, `↑` uploading, `✓` done, `✗` failed.

Upload item keys are shortened to just `recordings/<code>/XC123.mp3` (already the
format in the code).

## Post-TUI Reports

After `p.Run()` returns, `main.go` runs the existing summary blocks unchanged:

- `=== PARTIAL UPLOAD FAILURES ===` -- species where uploads errored, with a re-run
  hint: `just ingest --species <codes> <region>`
- `=== MISSING MEDIA ===` -- species with 0 recordings or 0 images (likely XC taxonomy
  mismatches), with a re-run hint: `just ingest --xc-override "..." --skip-complete <region>`

These are populated from `speciesDoneMsg.failures` and the post-cleanup DB query,
exactly as today.

## Signature Changes

`fetchAndUpload` gains two parameters to support observable phases:

```go
func fetchAndUpload(
    ctx context.Context, r2c *r2.Client,
    sourceURL, key, contentType string,
    workerID int, send func(tea.Msg),
) (string, error)
```

All call sites in `ingestSpecies` pass through the `workerID` and `send` they already receive.

All three files are `package main` in `cmd/ingest/`, so message types defined in
`tui.go` are directly accessible from `ingest.go` -- no imports needed between them.

## Testing

`ingest.go` is pure: `send func(tea.Msg)` is a no-op `func(tea.Msg) {}` in tests,
and `workerID` can be `0`. Existing `fetchAndUpload` tests update to pass these two
extra args; logic under test is unchanged.

`tui.go` gets unit tests for `Update`: feed it a sequence of messages, assert model
state after each. No terminal or rendering required -- `Update` is a pure function
`(model, msg) → (model, cmd)`.

## Non-Goals

- No TTY fallback (ingest is always run interactively)
- No byte-level upload progress (R2 is a single PutObject; phases are fetch vs. upload)
- No changes to flags, env vars, DB logic, R2 logic, or post-run reports
