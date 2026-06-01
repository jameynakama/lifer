# Ingest TUI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `log.Printf` progress output in `cmd/ingest` with a Docker-style Bubble Tea TUI -- per-worker rows showing upload ticks, completed species scrolling up with pass/fail indicators, error reports printed as plain text after TUI exits.

**Architecture:** Three files in `backend/cmd/ingest/` share `package main`. `ingest.go` holds pure logic; goroutines call a `send func(any)` callback instead of `log.Printf`. `tui.go` holds the Bubble Tea model, message types, and rendering. `main.go` wires everything: typed semaphore for worker slot IDs, worker goroutine that sends `allDoneMsg{}` on completion, `p.Run()` on the main goroutine.

**Tech Stack:** `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/bubbles/progress`, `github.com/charmbracelet/lipgloss`

---

## File Map

| File | Action | Responsibility |
|------|--------|---------------|
| `backend/cmd/ingest/main.go` | Modify | CLI flags, env, pre-TUI setup, TUI wiring |
| `backend/cmd/ingest/ingest.go` | Create | All ingest logic -- `ingestSpecies`, `fetchAndUpload`, helpers |
| `backend/cmd/ingest/tui.go` | Create | Message types, model, `Init`/`Update`/`View`, Lipgloss styles |
| `backend/cmd/ingest/main_test.go` | Modify | Update `fetchAndUpload` call sites; add message-send test |
| `backend/cmd/ingest/tui_test.go` | Create | Unit tests for `Update` -- pure function, no terminal needed |
| `backend/go.mod` / `go.sum` | Modify | Add three Charm deps |

---

## Task 1: Add Charm dependencies

**Files:**
- Modify: `backend/go.mod`, `backend/go.sum`

- [ ] **Step 1: Add the three packages**

```bash
cd backend && go get github.com/charmbracelet/bubbletea github.com/charmbracelet/bubbles github.com/charmbracelet/lipgloss
```

- [ ] **Step 2: Verify the build still compiles**

```bash
just test args='-run=^$ ./cmd/ingest/...'
```

Expected: PASS (no tests match the empty pattern, but build must succeed)

- [ ] **Step 3: Commit**

```bash
jj commit -m "chore(ingest): add bubbletea, bubbles, lipgloss deps"
```

---

## Task 2: Define TUI types in `tui.go`

**Files:**
- Create: `backend/cmd/ingest/tui.go`

- [ ] **Step 1: Create `tui.go` with all types (no Update/View yet)**

```go
package main

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// Messages sent from worker goroutines to the TUI via p.Send.

type speciesStartedMsg struct {
	workerID int
	code     string
	name     string
}

type fetchStartedMsg struct {
	workerID int
	key      string
}

type uploadStartedMsg struct {
	workerID int
	key      string
}

type uploadDoneMsg struct {
	workerID int
	key      string
	err      error
}

type speciesDoneMsg struct {
	workerID   int
	code       string
	name       string
	recordings int
	images     int
	failures   []string
}

type allDoneMsg struct{}

type tickMsg time.Time

// Model state

type uploadStatus int

const (
	statusWaiting uploadStatus = iota
	statusFetching
	statusUploading
	statusDone
	statusFailed
)

type uploadItem struct {
	key    string
	status uploadStatus
	err    error
}

type workerState struct {
	name    string
	uploads []uploadItem
}

type completedItem struct {
	name       string
	recordings int
	images     int
	failed     bool
}

type model struct {
	workers   []workerState
	completed []completedItem
	done      int
	total     int
	start     time.Time
	width     int
	height    int
	progress  progress.Model
}

func newModel(total, numWorkers int) model {
	return model{
		workers:  make([]workerState, numWorkers),
		total:    total,
		start:    time.Now(),
		progress: progress.New(progress.WithDefaultGradient()),
	}
}

// appendOrUpdateUpload finds the upload item with key and sets its status,
// or appends a new item if not found.
func appendOrUpdateUpload(items []uploadItem, key string, status uploadStatus, err error) []uploadItem {
	for i := range items {
		if items[i].key == key {
			items[i].status = status
			items[i].err = err
			return items
		}
	}
	return append(items, uploadItem{key: key, status: status, err: err})
}

// Stub Init/Update/View so the package compiles; replaced in Task 3 and Task 7.

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m model) View() string { return "" }
```

- [ ] **Step 2: Verify package compiles**

```bash
just test args='-run=^$ ./cmd/ingest/...'
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
jj commit -m "feat(ingest): define TUI message types and model shape"
```

---

## Task 3: TDD for `tui.go` -- `Init` and `Update`

**Files:**
- Create: `backend/cmd/ingest/tui_test.go`
- Modify: `backend/cmd/ingest/tui.go`

- [ ] **Step 1: Write failing tests in `tui_test.go`**

```go
package main

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func baseModel(t *testing.T) model {
	t.Helper()
	return newModel(10, 3)
}

func TestUpdate_SpeciesStarted(t *testing.T) {
	m := baseModel(t)
	next, _ := m.Update(speciesStartedMsg{workerID: 1, code: "amro", name: "American Robin"})
	got := next.(model)
	if got.workers[1].name != "American Robin" {
		t.Errorf("Should set worker name: got %q", got.workers[1].name)
	}
	if len(got.workers[1].uploads) != 0 {
		t.Errorf("Should clear uploads on new species")
	}
}

func TestUpdate_FetchStarted(t *testing.T) {
	m := baseModel(t)
	m.workers[0].name = "Bushtit"
	next, _ := m.Update(fetchStartedMsg{workerID: 0, key: "recordings/busti/XC1.mp3"})
	got := next.(model)
	if len(got.workers[0].uploads) != 1 {
		t.Fatalf("Should add one upload item")
	}
	if got.workers[0].uploads[0].status != statusFetching {
		t.Errorf("Should set status to fetching")
	}
}

func TestUpdate_UploadStarted(t *testing.T) {
	m := baseModel(t)
	m.workers[0].uploads = []uploadItem{{key: "recordings/busti/XC1.mp3", status: statusFetching}}
	next, _ := m.Update(uploadStartedMsg{workerID: 0, key: "recordings/busti/XC1.mp3"})
	got := next.(model)
	if got.workers[0].uploads[0].status != statusUploading {
		t.Errorf("Should update status to uploading")
	}
}

func TestUpdate_UploadDone_Success(t *testing.T) {
	m := baseModel(t)
	m.workers[2].uploads = []uploadItem{{key: "images/busti/ML1.jpg", status: statusUploading}}
	next, _ := m.Update(uploadDoneMsg{workerID: 2, key: "images/busti/ML1.jpg", err: nil})
	got := next.(model)
	if got.workers[2].uploads[0].status != statusDone {
		t.Errorf("Should set status to done")
	}
	if got.workers[2].uploads[0].err != nil {
		t.Errorf("Should have nil err")
	}
}

func TestUpdate_UploadDone_Error(t *testing.T) {
	m := baseModel(t)
	uploadErr := errors.New("timeout")
	m.workers[1].uploads = []uploadItem{{key: "recordings/busti/XC1.mp3", status: statusUploading}}
	next, _ := m.Update(uploadDoneMsg{workerID: 1, key: "recordings/busti/XC1.mp3", err: uploadErr})
	got := next.(model)
	if got.workers[1].uploads[0].status != statusFailed {
		t.Errorf("Should set status to failed")
	}
	if !errors.Is(got.workers[1].uploads[0].err, uploadErr) {
		t.Errorf("Should store error")
	}
}

func TestUpdate_SpeciesDone_Success(t *testing.T) {
	m := baseModel(t)
	m.workers[0].name = "American Robin"
	m.workers[0].uploads = []uploadItem{{key: "recordings/amro/XC1.mp3", status: statusDone}}
	m.done = 2

	next, _ := m.Update(speciesDoneMsg{
		workerID: 0, code: "amro", name: "American Robin",
		recordings: 4, images: 3, failures: nil,
	})
	got := next.(model)

	if got.done != 3 {
		t.Errorf("Should increment done counter: got %d", got.done)
	}
	if got.workers[0].name != "" {
		t.Errorf("Should clear worker name")
	}
	if len(got.workers[0].uploads) != 0 {
		t.Errorf("Should clear worker uploads")
	}
	if len(got.completed) != 1 {
		t.Fatalf("Should add to completed list")
	}
	c := got.completed[0]
	if c.name != "American Robin" || c.recordings != 4 || c.images != 3 || c.failed {
		t.Errorf("Should store completed item correctly: %+v", c)
	}
}

func TestUpdate_SpeciesDone_WithFailures(t *testing.T) {
	m := baseModel(t)
	next, _ := m.Update(speciesDoneMsg{
		workerID: 0, code: "busti", name: "Bushtit",
		recordings: 0, images: 3, failures: []string{"upload error"},
	})
	got := next.(model)
	if !got.completed[0].failed {
		t.Errorf("Should mark as failed when there are upload failures")
	}
}

func TestUpdate_SpeciesDone_MissingMedia(t *testing.T) {
	m := baseModel(t)
	next, _ := m.Update(speciesDoneMsg{
		workerID: 0, code: "busti", name: "Bushtit",
		recordings: 0, images: 3, failures: nil,
	})
	got := next.(model)
	if !got.completed[0].failed {
		t.Errorf("Should mark as failed when recordings == 0")
	}
}

func TestUpdate_AllDone_ReturnsQuit(t *testing.T) {
	m := baseModel(t)
	_, cmd := m.Update(allDoneMsg{})
	if cmd == nil {
		t.Fatal("Should return a command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("Should return tea.Quit command, got %T", msg)
	}
}

func TestUpdate_Tick_ReturnsTick(t *testing.T) {
	m := baseModel(t)
	_, cmd := m.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Fatal("Should return a tick command for elapsed timer")
	}
}

func TestUpdate_WindowSize_SetsWidthHeight(t *testing.T) {
	m := baseModel(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	got := next.(model)
	if got.width != 120 || got.height != 40 {
		t.Errorf("Should store terminal dimensions: got %dx%d", got.width, got.height)
	}
}
```

- [ ] **Step 2: Run tests -- confirm they fail**

```bash
just test args='-run=TestUpdate ./cmd/ingest/...'
```

Expected: FAIL (Update is a stub returning unchanged model)

- [ ] **Step 3: Implement `Init` and `Update` in `tui.go` (replace stubs)**

Replace the three stub functions at the bottom of `tui.go` with:

```go
func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) }),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case speciesStartedMsg:
		m.workers[msg.workerID].name = msg.name
		m.workers[msg.workerID].uploads = nil
		return m, nil

	case fetchStartedMsg:
		m.workers[msg.workerID].uploads = appendOrUpdateUpload(
			m.workers[msg.workerID].uploads, msg.key, statusFetching, nil)
		return m, nil

	case uploadStartedMsg:
		m.workers[msg.workerID].uploads = appendOrUpdateUpload(
			m.workers[msg.workerID].uploads, msg.key, statusUploading, nil)
		return m, nil

	case uploadDoneMsg:
		status := statusDone
		if msg.err != nil {
			status = statusFailed
		}
		m.workers[msg.workerID].uploads = appendOrUpdateUpload(
			m.workers[msg.workerID].uploads, msg.key, status, msg.err)
		return m, nil

	case speciesDoneMsg:
		m.done++
		m.workers[msg.workerID] = workerState{}
		failed := len(msg.failures) > 0 || msg.recordings == 0 || msg.images == 0
		m.completed = append(m.completed, completedItem{
			name:       msg.name,
			recordings: msg.recordings,
			images:     msg.images,
			failed:     failed,
		})
		cmd := m.progress.SetPercent(float64(m.done) / float64(m.total))
		return m, cmd

	case allDoneMsg:
		return m, tea.Quit

	case tickMsg:
		return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })

	case progress.FrameMsg:
		pm, cmd := m.progress.Update(msg)
		if newPM, ok := pm.(progress.Model); ok {
			m.progress = newPM
		}
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 20
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}
	return m, nil
}

func (m model) View() string { return "" }
```

- [ ] **Step 4: Run tests -- confirm they pass**

```bash
just test args='-run=TestUpdate ./cmd/ingest/...'
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
jj commit -m "feat(ingest): TUI model Update with full message handling"
```

---

## Task 4: Create `ingest.go` -- extract logic, add send callbacks

**Files:**
- Create: `backend/cmd/ingest/ingest.go`
- Modify: `backend/cmd/ingest/main.go` (remove extracted functions)

- [ ] **Step 1: Create `ingest.go` with all extracted + updated functions**

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jameynakama/lifer/internal/ebird"
	"github.com/jameynakama/lifer/internal/macaulay"
	"github.com/jameynakama/lifer/internal/r2"
	"github.com/jameynakama/lifer/internal/store"
	"github.com/jameynakama/lifer/internal/xenocanto"
	"os"
	"log"
)

type ingestStats struct {
	failures   []string
	recordings int
	images     int
}

// retryDelays controls the wait between attempts on a 429 response.
// Overridable in tests.
var retryDelays = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}

// ingestSpecies fetches and uploads media for one species.
// xcOverrides maps ebird codes to [genus, species] pairs for xeno-canto taxonomy overrides.
func ingestSpecies(
	ctx context.Context,
	q *store.Queries,
	xc *xenocanto.Client,
	mac *macaulay.Client,
	entry ebird.TaxonomyEntry,
	maxRec, maxImg int,
	r2c *r2.Client,
	xcOverrides map[string][2]string,
	workerID int,
	send func(any),
) (stats ingestStats, err error) {
	defer func() {
		send(speciesDoneMsg{
			workerID:   workerID,
			code:       entry.SpeciesCode,
			name:       entry.CommonName,
			recordings: stats.recordings,
			images:     stats.images,
			failures:   stats.failures,
		})
	}()

	send(speciesStartedMsg{workerID: workerID, code: entry.SpeciesCode, name: entry.CommonName})

	sp, err := q.UpsertSpecies(ctx, store.UpsertSpeciesParams{
		EbirdCode:      entry.SpeciesCode,
		CommonName:     entry.CommonName,
		ScientificName: entry.SciName,
	})
	if err != nil {
		err = fmt.Errorf("upsert species: %w", err)
		return
	}

	xcGenus, xcSpecies := xcGenSp(entry.SpeciesCode, entry.SciName, xcOverrides)
	perType := maxRec / 2

	type searchResult struct {
		recType string
		recs    []xenocanto.Recording
		err     error
	}
	searchCh := make(chan searchResult, 2)
	for _, recType := range []string{"song", "call"} {
		go func(rt string) {
			recs, err := xc.Search(ctx, xcGenus, xcSpecies, rt)
			searchCh <- searchResult{rt, recs, err}
		}(recType)
	}

	var (
		recWg   sync.WaitGroup
		statsMu sync.Mutex
	)

	recordFailure := func(reason string) {
		statsMu.Lock()
		stats.failures = append(stats.failures, reason)
		statsMu.Unlock()
	}

	for range 2 {
		result := <-searchCh
		if result.err != nil {
			continue
		}
		recs := result.recs
		if len(recs) > perType {
			recs = recs[:perType]
		}
		for _, rec := range recs {
			recWg.Add(1)
			go func(rec xenocanto.Recording) {
				defer recWg.Done()
				key := "recordings/" + sp.EbirdCode + "/" + rec.ID + ".mp3"
				filePath, err := fetchAndUpload(ctx, r2c, rec.FileURL, key, "audio/mpeg", workerID, send)
				if err != nil {
					recordFailure(fmt.Sprintf("recording %s: %v", rec.ID, err))
					return
				}
				if _, err := q.UpsertRecording(ctx, store.UpsertRecordingParams{
					XenoCantoID: rec.ID,
					SpeciesCode: sp.EbirdCode,
					FilePath:    filePath,
					Quality:     rec.Quality,
					Type:        rec.Type,
				}); err != nil {
					return
				}
				statsMu.Lock()
				stats.recordings++
				statsMu.Unlock()
			}(rec)
		}
	}
	recWg.Wait()

	photos, err := mac.Photos(ctx, entry.SpeciesCode, maxImg)
	if err != nil {
		err = nil // non-fatal: species will show 0 images in missingMedia report
		return
	}

	var imgWg sync.WaitGroup
	for _, photo := range photos {
		imgWg.Add(1)
		go func(photo macaulay.Photo) {
			defer imgWg.Done()
			key := "images/" + sp.EbirdCode + "/" + photo.AssetID + ".jpg"
			filePath, err := fetchAndUpload(ctx, r2c, mac.PhotoURL(photo.AssetID), key, "image/jpeg", workerID, send)
			if err != nil {
				recordFailure(fmt.Sprintf("image %s: %v", photo.AssetID, err))
				return
			}
			if _, err := q.UpsertSpeciesImage(ctx, store.UpsertSpeciesImageParams{
				MacaulayID:  photo.AssetID,
				SpeciesCode: sp.EbirdCode,
				FilePath:    filePath,
				Credit:      photo.UserDisplayName,
			}); err != nil {
				return
			}
			statsMu.Lock()
			stats.images++
			statsMu.Unlock()
		}(photo)
	}
	imgWg.Wait()
	return
}

// xcGenSp returns the genus and species to use for a xeno-canto query.
func xcGenSp(ebirdCode, sciName string, xcOverrides map[string][2]string) (genus, species string) {
	if override, ok := xcOverrides[ebirdCode]; ok {
		return override[0], override[1]
	}
	parts := strings.Fields(sciName)
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return sciName, ""
}

// parseXCOverrides parses "--xc-override comrav=Corvus:corax,calsja=Aphelocoma:californica"
func parseXCOverrides(s string) (map[string][2]string, error) {
	out := make(map[string][2]string)
	if s == "" {
		return out, nil
	}
	for _, entry := range strings.Split(s, ",") {
		kv := strings.SplitN(entry, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("expected code=Genus:species, got %q", entry)
		}
		genSp := strings.SplitN(kv[1], ":", 2)
		if len(genSp) != 2 {
			return nil, fmt.Errorf("expected Genus:species after =, got %q", kv[1])
		}
		out[kv[0]] = [2]string{genSp[0], genSp[1]}
	}
	return out, nil
}

// fetchAndUpload GETs from sourceURL and uploads the body to R2 at key.
// Sends fetchStartedMsg, uploadStartedMsg, and uploadDoneMsg via send.
// Retries on 429 from the source. Returns the full public R2 URL.
func fetchAndUpload(ctx context.Context, r2c *r2.Client, sourceURL, key, contentType string, workerID int, send func(any)) (string, error) {
	send(fetchStartedMsg{workerID: workerID, key: key})
	var lastErr error
	for attempt := range len(retryDelays) + 1 {
		if attempt > 0 {
			select {
			case <-time.After(retryDelays[attempt-1]):
			case <-ctx.Done():
				err := ctx.Err()
				send(uploadDoneMsg{workerID: workerID, key: key, err: err})
				return "", err
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
		if err != nil {
			send(uploadDoneMsg{workerID: workerID, key: key, err: err})
			return "", err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			send(uploadDoneMsg{workerID: workerID, key: key, err: err})
			return "", err
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = fmt.Errorf("fetch %s: status 429", sourceURL)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			err := fmt.Errorf("fetch %s: status %d", sourceURL, resp.StatusCode)
			send(uploadDoneMsg{workerID: workerID, key: key, err: err})
			return "", err
		}
		send(uploadStartedMsg{workerID: workerID, key: key})
		url, err := r2c.Upload(ctx, key, contentType, resp.Body)
		resp.Body.Close()
		if err != nil {
			send(uploadDoneMsg{workerID: workerID, key: key, err: err})
			return "", err
		}
		send(uploadDoneMsg{workerID: workerID, key: key, err: nil})
		return url, nil
	}
	send(uploadDoneMsg{workerID: workerID, key: key, err: lastErr})
	return "", lastErr
}

func filterBySpecies(codes []string, taxMap map[string]ebird.TaxonomyEntry, want []string) []string {
	if len(want) == 0 {
		return []string{}
	}
	match := make(map[string]struct{}, len(want))
	for _, w := range want {
		lower := strings.ToLower(w)
		if _, ok := taxMap[lower]; ok {
			match[lower] = struct{}{}
			continue
		}
		for code, entry := range taxMap {
			if strings.ToLower(entry.CommonName) == lower {
				match[code] = struct{}{}
				break
			}
		}
	}
	out := make([]string, 0, len(match))
	for _, c := range codes {
		if _, ok := match[c]; ok {
			out = append(out, c)
		}
	}
	return out
}

func filterComplete(codes []string, complete map[string]struct{}) []string {
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		if _, ok := complete[c]; !ok {
			out = append(out, c)
		}
	}
	return out
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
```

- [ ] **Step 2: Slim `main.go` -- remove all functions now in `ingest.go`**

Replace the entire contents of `backend/cmd/ingest/main.go` with just the `main()` function and its imports (everything after line 22 in the original file, minus the extracted functions). The file should contain only:

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/lifer/internal/ebird"
	"github.com/jameynakama/lifer/internal/macaulay"
	"github.com/jameynakama/lifer/internal/r2"
	"github.com/jameynakama/lifer/internal/store"
	"github.com/jameynakama/lifer/internal/xenocanto"
)

func main() {
	maxRecordings := flag.Int("max-recordings", 4, "max recordings per species (split evenly between song and call)")
	maxImages := flag.Int("max-images", 3, "max images per species")
	workers := flag.Int("workers", 5, "concurrent worker count")
	skipComplete := flag.Bool("skip-complete", false, "skip species that already have ≥1 recording and ≥1 image in the DB")
	speciesFilter := flag.String("species", "", "comma-separated ebird codes or common names to process")
	xcOverrideFlag := flag.String("xc-override", "", "comma-separated xeno-canto taxonomy overrides, e.g. \"comrav=Corvus:corax\"")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ingest [flags] <region-code> [region-code...]\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  ingest US-OR\n")
		fmt.Fprintf(os.Stderr, "  ingest US-OR US-WA US-ID\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	regions := flag.Args()
	if len(regions) == 0 {
		fmt.Fprintln(os.Stderr, "usage: ingest [flags] <region-code> [region-code...]")
		os.Exit(1)
	}

	xcOverrides, err := parseXCOverrides(*xcOverrideFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "--xc-override: %v\n", err)
		os.Exit(1)
	}

	ebirdKey := mustEnv("EBIRD_API_KEY")
	xcKey := mustEnv("XENO_CANTO_API_KEY")
	dbURL := mustEnv("DATABASE_URL")
	r2AccountID := mustEnv("R2_ACCOUNT_ID")
	r2AccessKey := mustEnv("R2_ACCESS_KEY_ID")
	r2SecretKey := mustEnv("R2_SECRET_ACCESS_KEY")
	r2Bucket := mustEnv("R2_BUCKET_NAME")
	r2PubURL := mustEnv("R2_PUBLIC_URL")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	r2c, err := r2.New(r2AccountID, r2AccessKey, r2SecretKey, r2Bucket, r2PubURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "r2 client: %v\n", err)
		os.Exit(1)
	}

	q := store.New(pool)
	ebirdClient := ebird.New(ebirdKey)
	xcClient := xenocanto.New(xcKey)
	macaulayClient := macaulay.New(ebirdKey)

	fmt.Fprintln(os.Stderr, "fetching eBird taxonomy...")
	taxonomy, err := ebirdClient.Taxonomy(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch taxonomy: %v\n", err)
		os.Exit(1)
	}
	taxMap := make(map[string]ebird.TaxonomyEntry, len(taxonomy))
	for _, t := range taxonomy {
		taxMap[t.SpeciesCode] = t
	}
	fmt.Fprintf(os.Stderr, "taxonomy loaded: %d entries\n", len(taxMap))

	seen := make(map[string]struct{})
	var codes []string
	for _, region := range regions {
		list, err := ebirdClient.SpeciesList(ctx, region)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: region %s: %v\n", region, err)
			continue
		}
		for _, code := range list {
			if _, ok := seen[code]; !ok {
				seen[code] = struct{}{}
				codes = append(codes, code)
			}
		}
		fmt.Fprintf(os.Stderr, "region %s: %d species\n", region, len(list))
	}

	if *speciesFilter != "" {
		want := strings.Split(*speciesFilter, ",")
		before := len(codes)
		codes = filterBySpecies(codes, taxMap, want)
		fmt.Fprintf(os.Stderr, "--species: filtered to %d/%d species\n", len(codes), before)
	}

	if *skipComplete {
		completeCodes, err := q.ListCompleteSpeciesEbirdCodes(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip-complete: query complete species: %v\n", err)
			os.Exit(1)
		}
		complete := make(map[string]struct{}, len(completeCodes))
		for _, c := range completeCodes {
			complete[c] = struct{}{}
		}
		before := len(codes)
		codes = filterComplete(codes, complete)
		fmt.Fprintf(os.Stderr, "--skip-complete: skipping %d already-complete species, processing %d remaining\n", before-len(codes), len(codes))
	}

	// Pre-filter to codes with taxonomy entries; warn for missing ones.
	type codeEntry struct {
		code  string
		entry ebird.TaxonomyEntry
	}
	var processable []codeEntry
	for _, code := range codes {
		if entry, ok := taxMap[code]; ok {
			processable = append(processable, codeEntry{code, entry})
		} else {
			fmt.Fprintf(os.Stderr, "warn: no taxonomy entry for %s, skipping\n", code)
		}
	}
	total := len(processable)
	fmt.Fprintf(os.Stderr, "total unique species to process: %d\n", total)

	// --- TUI placeholder: replaced in Task 6 ---
	// For now, keep plain output so the build compiles.
	failedSpecies := map[string][]string{}
	missingMedia := map[string]ingestStats{}

	slots := make(chan int, *workers)
	for i := range *workers {
		slots <- i
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	noop := func(any) {}

	for _, ce := range processable {
		workerID := <-slots
		wg.Add(1)
		go func(ce codeEntry, workerID int) {
			defer wg.Done()
			defer func() { slots <- workerID }()
			stats, err := ingestSpecies(ctx, q, xcClient, macaulayClient, ce.entry, *maxRecordings, *maxImages, r2c, xcOverrides, workerID, noop)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error %s (%s): %v\n", ce.entry.CommonName, ce.code, err)
			}
			mu.Lock()
			if len(stats.failures) > 0 {
				failedSpecies[ce.code] = stats.failures
			} else if err == nil && (stats.recordings == 0 || stats.images == 0) {
				missingMedia[ce.code] = stats
			}
			mu.Unlock()
		}(ce, workerID)
	}
	wg.Wait()

	fmt.Fprintf(os.Stderr, "ingestion complete: %d/%d species\n", total, total)

	// --- post-run cleanup and reports (unchanged) ---
	fmt.Fprintln(os.Stderr, "cleaning up species missing recordings or images...")
	incomplete, err := q.ListIncompleteSpecies(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cleanup: %v\n", err)
		os.Exit(1)
	}
	for _, code := range incomplete {
		for _, prefix := range []string{"recordings/" + code + "/", "images/" + code + "/"} {
			if err := r2c.DeletePrefix(ctx, prefix); err != nil {
				fmt.Fprintf(os.Stderr, "  warn: cleanup R2 %s: %v\n", prefix, err)
			}
		}
		if err := q.DeleteRecordingsBySpeciesCode(ctx, code); err != nil {
			fmt.Fprintf(os.Stderr, "  warn: cleanup recordings %s: %v\n", code, err)
		}
		if err := q.DeleteSpeciesImagesBySpeciesCode(ctx, code); err != nil {
			fmt.Fprintf(os.Stderr, "  warn: cleanup images %s: %v\n", code, err)
		}
		if err := q.DeleteSpeciesByCode(ctx, code); err != nil {
			fmt.Fprintf(os.Stderr, "  warn: cleanup species %s: %v\n", code, err)
		}
	}
	fmt.Fprintf(os.Stderr, "cleanup: removed %d incomplete species\n", len(incomplete))

	if len(failedSpecies) > 0 {
		failedCodes := make([]string, 0, len(failedSpecies))
		for code := range failedSpecies {
			failedCodes = append(failedCodes, code)
		}
		sort.Strings(failedCodes)
		fmt.Printf("\n=== PARTIAL UPLOAD FAILURES (%d species) ===\n", len(failedSpecies))
		for _, code := range failedCodes {
			name := taxMap[code].CommonName
			fmt.Printf("  %s (%s):\n", name, code)
			for _, reason := range failedSpecies[code] {
				fmt.Printf("    - %s\n", reason)
			}
		}
		fmt.Println("cleaning up partial R2 uploads and DB entries for failed species...")
		for _, code := range failedCodes {
			for _, prefix := range []string{"recordings/" + code + "/", "images/" + code + "/"} {
				if err := r2c.DeletePrefix(ctx, prefix); err != nil {
					fmt.Fprintf(os.Stderr, "  warn: R2 delete %s: %v\n", prefix, err)
				}
			}
			if err := q.DeleteRecordingsBySpeciesCode(ctx, code); err != nil {
				fmt.Fprintf(os.Stderr, "  warn: DB delete recordings %s: %v\n", code, err)
			}
			if err := q.DeleteSpeciesImagesBySpeciesCode(ctx, code); err != nil {
				fmt.Fprintf(os.Stderr, "  warn: DB delete images %s: %v\n", code, err)
			}
			if err := q.DeleteSpeciesByCode(ctx, code); err != nil {
				fmt.Fprintf(os.Stderr, "  warn: DB delete species %s: %v\n", code, err)
			}
		}
		fmt.Printf("re-run failed species with:\n")
		fmt.Printf("  just ingest --species %s <region>\n", strings.Join(failedCodes, ","))
	}

	if len(missingMedia) > 0 {
		missingCodes := make([]string, 0, len(missingMedia))
		for code := range missingMedia {
			missingCodes = append(missingCodes, code)
		}
		sort.Strings(missingCodes)
		fmt.Printf("\n=== MISSING MEDIA (%d species) ===\n", len(missingMedia))
		var xcMisses []string
		for _, code := range missingCodes {
			stats := missingMedia[code]
			name := taxMap[code].CommonName
			switch {
			case stats.recordings == 0 && stats.images == 0:
				fmt.Printf("  %s (%s): no recordings, no images\n", name, code)
				xcMisses = append(xcMisses, code)
			case stats.recordings == 0:
				fmt.Printf("  %s (%s): no recordings (xeno-canto miss -- check taxonomy)\n", name, code)
				xcMisses = append(xcMisses, code)
			case stats.images == 0:
				fmt.Printf("  %s (%s): no images (macaulay miss)\n", name, code)
			}
		}
		if len(xcMisses) > 0 {
			fmt.Println("for xeno-canto misses, research the species on xeno-canto.org then re-run:")
			fmt.Printf("  just ingest --xc-override \"<code>=Genus:species,...\" --skip-complete <region>\n")
			fmt.Printf("  missing codes: %s\n", strings.Join(xcMisses, ","))
		}
	}
}
```

- [ ] **Step 3: Verify package compiles**

```bash
just test args='-run=^$ ./cmd/ingest/...'
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
jj commit -m "refactor(ingest): extract ingest.go, add send callbacks to fetchAndUpload/ingestSpecies"
```

---

## Task 5: Update `main_test.go` for new `fetchAndUpload` signature

**Files:**
- Modify: `backend/cmd/ingest/main_test.go`

- [ ] **Step 1: Run existing tests -- confirm they fail due to wrong arity**

```bash
just test args='./cmd/ingest/...'
```

Expected: FAIL -- `fetchAndUpload` called with 5 args, now needs 7

- [ ] **Step 2: Update all `fetchAndUpload` call sites and add a message-sequence test**

Replace the entire `main_test.go` with:

```go
package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jameynakama/lifer/internal/ebird"
	"github.com/jameynakama/lifer/internal/r2"
)

func nopSend(any) {}

func TestFilterBySpecies(t *testing.T) {
	taxMap := map[string]ebird.TaxonomyEntry{
		"busti": {SpeciesCode: "busti", CommonName: "Bushtit"},
		"rukin": {SpeciesCode: "rukin", CommonName: "Ruby-crowned Kinglet"},
		"amro":  {SpeciesCode: "amro", CommonName: "American Robin"},
	}
	codes := []string{"busti", "rukin", "amro"}

	tests := []struct {
		name    string
		want    []string
		wantOut []string
	}{
		{"ebird code", []string{"busti"}, []string{"busti"}},
		{"common name case-insensitive", []string{"bushtit"}, []string{"busti"}},
		{"mixed code and name", []string{"busti", "ruby-crowned kinglet"}, []string{"busti", "rukin"}},
		{"multiple codes", []string{"rukin", "amro"}, []string{"rukin", "amro"}},
		{"no match is excluded", []string{"busti", "doesnotexist"}, []string{"busti"}},
		{"empty want", []string{}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterBySpecies(codes, taxMap, tt.want)
			if len(got) != len(tt.wantOut) {
				t.Fatalf("filterBySpecies() len = %d, want %d (got %v, want %v)", len(got), len(tt.wantOut), got, tt.wantOut)
			}
			for i := range tt.wantOut {
				if got[i] != tt.wantOut[i] {
					t.Errorf("filterBySpecies()[%d] = %q, want %q", i, got[i], tt.wantOut[i])
				}
			}
		})
	}
}

func TestFilterComplete(t *testing.T) {
	tests := []struct {
		name     string
		codes    []string
		complete map[string]struct{}
		want     []string
	}{
		{
			name:     "empty complete set passes all through",
			codes:    []string{"AMRO", "BCCH", "NOCA"},
			complete: map[string]struct{}{},
			want:     []string{"AMRO", "BCCH", "NOCA"},
		},
		{
			name:     "complete species are removed",
			codes:    []string{"AMRO", "BCCH", "NOCA"},
			complete: map[string]struct{}{"BCCH": {}},
			want:     []string{"AMRO", "NOCA"},
		},
		{
			name:     "all complete returns empty slice",
			codes:    []string{"AMRO", "BCCH"},
			complete: map[string]struct{}{"AMRO": {}, "BCCH": {}},
			want:     []string{},
		},
		{
			name:     "nil codes returns empty slice",
			codes:    nil,
			complete: map[string]struct{}{"AMRO": {}},
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterComplete(tt.codes, tt.complete)
			if len(got) != len(tt.want) {
				t.Fatalf("filterComplete() len = %d, want %d (got %v, want %v)", len(got), len(tt.want), got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("filterComplete()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFetchAndUpload_Success(t *testing.T) {
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("audio bytes"))
	}))
	defer src.Close()

	var uploaded string
	r2s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			b, _ := io.ReadAll(r.Body)
			uploaded = string(b)
			w.Header().Set("ETag", `"x"`)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer r2s.Close()

	r2c, err := r2.NewWithEndpoint(r2s.URL, "k", "s", "bucket", "https://pub.example.com")
	if err != nil {
		t.Fatalf("Should create r2 client: %v", err)
	}

	url, err := fetchAndUpload(context.Background(), r2c, src.URL, "recordings/busti/123.mp3", "audio/mpeg", 0, nopSend)
	if err != nil {
		t.Fatalf("Should fetch and upload without error: %v", err)
	}
	if url != "https://pub.example.com/recordings/busti/123.mp3" {
		t.Errorf("Should return public URL, got %q", url)
	}
	if uploaded != "audio bytes" {
		t.Errorf("Should upload source body, got %q", uploaded)
	}
}

func TestFetchAndUpload_SourceNonOK_ReturnsError(t *testing.T) {
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer src.Close()

	r2c, _ := r2.NewWithEndpoint("http://localhost:1", "k", "s", "bucket", "https://pub.example.com")
	_, err := fetchAndUpload(context.Background(), r2c, src.URL, "key", "audio/mpeg", 0, nopSend)
	if err == nil {
		t.Error("Should return error for non-200 source response")
	}
}

func TestFetchAndUpload_Retries429(t *testing.T) {
	origDelays := retryDelays
	retryDelays = []time.Duration{0, 0, 0}
	t.Cleanup(func() { retryDelays = origDelays })

	attempts := 0
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer src.Close()

	r2s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"x"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer r2s.Close()

	r2c, _ := r2.NewWithEndpoint(r2s.URL, "k", "s", "bucket", "https://pub.example.com")
	_, err := fetchAndUpload(context.Background(), r2c, src.URL, "key", "audio/mpeg", 0, nopSend)
	if err != nil {
		t.Fatalf("Should succeed after retries: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Should take 3 attempts, got %d", attempts)
	}
}

func TestFetchAndUpload_SendsMessageSequence(t *testing.T) {
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data"))
	}))
	defer src.Close()

	r2s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"x"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer r2s.Close()

	r2c, _ := r2.NewWithEndpoint(r2s.URL, "k", "s", "bucket", "https://pub.example.com")

	var msgs []any
	send := func(msg any) { msgs = append(msgs, msg) }

	_, err := fetchAndUpload(context.Background(), r2c, src.URL, "recordings/amro/XC1.mp3", "audio/mpeg", 2, send)
	if err != nil {
		t.Fatalf("Should succeed: %v", err)
	}

	if len(msgs) != 3 {
		t.Fatalf("Should send 3 messages (fetch, upload, done), got %d: %v", len(msgs), msgs)
	}
	if _, ok := msgs[0].(fetchStartedMsg); !ok {
		t.Errorf("msg[0] should be fetchStartedMsg, got %T", msgs[0])
	}
	if _, ok := msgs[1].(uploadStartedMsg); !ok {
		t.Errorf("msg[1] should be uploadStartedMsg, got %T", msgs[1])
	}
	done, ok := msgs[2].(uploadDoneMsg)
	if !ok {
		t.Errorf("msg[2] should be uploadDoneMsg, got %T", msgs[2])
	}
	if done.err != nil {
		t.Errorf("uploadDoneMsg should have nil err, got %v", done.err)
	}
	if done.workerID != 2 {
		t.Errorf("Should carry workerID 2, got %d", done.workerID)
	}
}
```

- [ ] **Step 3: Run all ingest tests -- confirm they pass**

```bash
just test args='./cmd/ingest/...'
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
jj commit -m "test(ingest): update fetchAndUpload call sites, add message-sequence test"
```

---

## Task 6: Wire `main.go` to use the TUI

**Files:**
- Modify: `backend/cmd/ingest/main.go`

- [ ] **Step 1: Replace the `--- TUI placeholder ---` section in `main.go`**

Find this block in `main.go` (between the pre-filter step and the post-run cleanup):

```go
	// --- TUI placeholder: replaced in Task 6 ---
	// For now, keep plain output so the build compiles.
	failedSpecies := map[string][]string{}
	missingMedia := map[string]ingestStats{}

	slots := make(chan int, *workers)
	for i := range *workers {
		slots <- i
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	noop := func(any) {}

	for _, ce := range processable {
		workerID := <-slots
		wg.Add(1)
		go func(ce codeEntry, workerID int) {
			defer wg.Done()
			defer func() { slots <- workerID }()
			stats, err := ingestSpecies(ctx, q, xcClient, macaulayClient, ce.entry, *maxRecordings, *maxImages, r2c, xcOverrides, workerID, noop)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error %s (%s): %v\n", ce.entry.CommonName, ce.code, err)
			}
			mu.Lock()
			if len(stats.failures) > 0 {
				failedSpecies[ce.code] = stats.failures
			} else if err == nil && (stats.recordings == 0 || stats.images == 0) {
				missingMedia[ce.code] = stats
			}
			mu.Unlock()
		}(ce, workerID)
	}
	wg.Wait()

	fmt.Fprintf(os.Stderr, "ingestion complete: %d/%d species\n", total, total)
```

Replace it with:

```go
	failedSpecies := map[string][]string{}
	missingMedia := map[string]ingestStats{}

	slots := make(chan int, *workers)
	for i := range *workers {
		slots <- i
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	m := newModel(total, *workers)
	p := tea.NewProgram(m)
	send := func(msg any) { p.Send(msg) }

	go func() {
		for _, ce := range processable {
			workerID := <-slots
			wg.Add(1)
			go func(ce codeEntry, workerID int) {
				defer wg.Done()
				defer func() { slots <- workerID }()
				stats, err := ingestSpecies(ctx, q, xcClient, macaulayClient, ce.entry, *maxRecordings, *maxImages, r2c, xcOverrides, workerID, send)
				if err != nil {
					// error already reflected in stats.failures via deferred speciesDoneMsg
					_ = err
				}
				mu.Lock()
				if len(stats.failures) > 0 {
					failedSpecies[ce.code] = stats.failures
				} else if stats.recordings == 0 || stats.images == 0 {
					missingMedia[ce.code] = stats
				}
				mu.Unlock()
			}(ce, workerID)
		}
		wg.Wait()
		p.Send(allDoneMsg{})
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		os.Exit(1)
	}
```

Also add `tea "github.com/charmbracelet/bubbletea"` to the import block in `main.go`.

- [ ] **Step 2: Verify all tests pass**

```bash
just test args='./cmd/ingest/...'
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
jj commit -m "feat(ingest): wire Bubble Tea TUI into main worker loop"
```

---

## Task 7: Add `View` and Lipgloss styles to `tui.go`

**Files:**
- Modify: `backend/cmd/ingest/tui.go`

- [ ] **Step 1: Add imports and styles, replace the `View` stub**

Add to the imports in `tui.go`:

```go
import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)
```

Add these style vars after the type definitions (before `newModel`):

```go
var (
	styleSuccess  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleFailed   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Faint(true)
	styleSubItem  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleYellow   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleBold     = lipgloss.NewStyle().Bold(true)
	styleDivider  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleElapsed  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleCounter  = lipgloss.NewStyle().Bold(true)
)
```

Replace the `View() string { return "" }` stub with:

```go
func (m model) View() string {
	if m.total == 0 {
		return ""
	}
	var b strings.Builder

	// Header
	elapsed := time.Since(m.start).Round(time.Second)
	h := int(elapsed.Hours())
	min := int(elapsed.Minutes()) % 60
	sec := int(elapsed.Seconds()) % 60
	elapsedStr := fmt.Sprintf("%d:%02d:%02d", h, min, sec)
	counter := styleCounter.Render(fmt.Sprintf("%d/%d", m.done, m.total))
	bar := m.progress.View()
	b.WriteString(fmt.Sprintf("  %s  %s  %s\n\n", counter, bar, styleElapsed.Render(elapsedStr)))

	// Completed species -- show last N that fit above the divider + workers
	workerH := 0
	for _, w := range m.workers {
		if w.name != "" {
			workerH += 1 + len(w.uploads)
		}
	}
	headerLines := 2
	dividerLine := 1
	bottomPad := 1
	availableForCompleted := m.height - headerLines - dividerLine - workerH - bottomPad
	if availableForCompleted < 0 {
		availableForCompleted = 0
	}
	start := 0
	if len(m.completed) > availableForCompleted {
		start = len(m.completed) - availableForCompleted
	}
	for _, c := range m.completed[start:] {
		icon := styleSuccess.Render("✓")
		nameStr := c.name
		detail := fmt.Sprintf("%d rec  %d img", c.recordings, c.images)
		if c.failed {
			icon = styleFailed.Render("✗")
			nameStr = styleFailed.Render(c.name)
			detail = styleFailed.Render(detail)
		}
		b.WriteString(fmt.Sprintf("  %s %-30s  %s\n", icon, nameStr, detail))
	}

	// Divider
	divider := styleDivider.Render(strings.Repeat("─", max(m.width-2, 40)))
	b.WriteString(fmt.Sprintf("  %s\n", divider))

	// Active workers
	for i, w := range m.workers {
		if w.name == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, styleBold.Render(w.name)))
		for _, u := range w.uploads {
			symbol, style := uploadSymbol(u.status)
			shortKey := shortKey(u.key)
			b.WriteString(fmt.Sprintf("      %s %-45s  %s\n",
				styleSubItem.Render("·"),
				styleSubItem.Render(shortKey),
				style.Render(symbol),
			))
		}
	}

	return b.String()
}

func uploadSymbol(s uploadStatus) (string, lipgloss.Style) {
	switch s {
	case statusFetching:
		return "↓ fetching", styleSubItem
	case statusUploading:
		return "↑ uploading", styleYellow
	case statusDone:
		return "✓", styleSuccess
	case statusFailed:
		return "✗", styleFailed
	default:
		return "· waiting", styleSubItem
	}
}

// shortKey trims "recordings/code/file.mp3" to just "file.mp3" for display.
func shortKey(key string) string {
	parts := strings.Split(key, "/")
	if len(parts) == 3 {
		return parts[0][:3] + "/" + parts[1] + "/" + parts[2]
	}
	return key
}

```

Note: `max` is a builtin in Go 1.21+ (`go.mod` is 1.26.1) -- do not define a custom `max` function.

- [ ] **Step 2: Run all tests**

```bash
just test args='./cmd/ingest/...'
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
jj commit -m "feat(ingest): add TUI View with Lipgloss styles"
```

---

## Task 8: Manual smoke test

- [ ] **Step 1: Run a minimal ingest to verify the TUI renders**

```bash
just ingest --species busti US-OR
```

Expected: TUI renders in the terminal -- header with progress bar, worker row for slot [1] with upload ticks for Bushtit's recordings and images, then Bushtit graduates to the completed list with ✓ or ✗. After TUI exits, any error reports print as plain text.

- [ ] **Step 2: Verify ctrl+c exits cleanly**

Start the same command and press ctrl+c. Expected: terminal restored cleanly, no stray output.

- [ ] **Step 3: Final commit if any fixups were needed**

```bash
jj commit -m "fix(ingest): TUI smoke test fixups"
```
