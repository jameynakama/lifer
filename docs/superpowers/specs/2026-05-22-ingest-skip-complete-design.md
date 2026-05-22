# Design: `--skip-complete` flag for `ingest`

**Date:** 2026-05-22  
**Status:** Approved

## Problem

When `ingest` is interrupted mid-run (network failure, process kill, etc.), restarting it from scratch re-hits xeno-canto and Macaulay for every species -- including the ~100+ already ingested. There is no way to pick up where it left off.

## Solution

Add a `--skip-complete` boolean flag. When set, the ingester queries the DB upfront for all species that already have ≥1 recording AND ≥1 image, then filters them out of the work queue before any workers spin up.

## Skip Condition

A species is considered complete if:
- It has at least one row in `recordings` for that species
- It has at least one row in `species_images` for that species

This mirrors the existing post-run cleanup logic (`ListIncompleteSpecies`): anything that survived cleanup is considered done. If the user wants more recordings/images than were previously downloaded, they run without `--skip-complete` (full reingest).

## Changes

### 1. New sqlc query -- `ListCompleteSpeciesEbirdCodes`

File: `backend/internal/store/queries/species.sql` (or appropriate queries file)

```sql
-- name: ListCompleteSpeciesEbirdCodes :many
SELECT s.ebird_code
FROM species s
WHERE EXISTS (SELECT 1 FROM recordings r WHERE r.species_id = s.id)
  AND EXISTS (SELECT 1 FROM species_images si WHERE si.species_id = s.id);
```

Run `just generate` after adding.

### 2. Flag in `main.go`

```go
skipComplete := flag.Bool("skip-complete", false, "skip species that already have recordings and images in the DB")
```

### 3. Filter logic in `main.go`

After building the deduplicated `codes` slice, when `--skip-complete` is set:

1. Call `q.ListCompleteSpeciesEbirdCodes(ctx)`
2. Build `complete map[string]struct{}` from result
3. Filter `codes` to only those not in `complete`
4. Log: `--skip-complete: skipping N already-complete species, processing M remaining`

The worker pool and post-run cleanup run exactly as before on the filtered list.

## Non-changes

- `ingestSpecies` is unchanged
- Worker pool concurrency is unchanged
- Post-run cleanup (`ListIncompleteSpecies` → delete) is unchanged -- still runs on the filtered batch to handle any newly incomplete species from the resumed run

## Usage

```
ingest --skip-complete US-OR
ingest --skip-complete --workers 10 US-OR US-WA US-ID
```

## Testing

- Unit test the filter logic: given a list of codes and a complete-set, assert the filtered output
- Integration or manual test: ingest a region partially, kill the process, resume with `--skip-complete`, assert previously-ingested species are skipped (check log output)
