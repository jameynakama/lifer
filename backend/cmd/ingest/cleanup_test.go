package main

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

// stubCleanupStore records delete calls; per-method error maps inject failures.
type stubCleanupStore struct {
	recDeletes, imgDeletes, spDeletes []string
	recErr, imgErr, spErr             map[string]error
}

func (s *stubCleanupStore) DeleteRecordingsBySpeciesCode(_ context.Context, code string) error {
	s.recDeletes = append(s.recDeletes, code)
	return s.recErr[code]
}

func (s *stubCleanupStore) DeleteSpeciesImagesBySpeciesCode(_ context.Context, code string) error {
	s.imgDeletes = append(s.imgDeletes, code)
	return s.imgErr[code]
}

func (s *stubCleanupStore) DeleteSpeciesByCode(_ context.Context, code string) error {
	s.spDeletes = append(s.spDeletes, code)
	return s.spErr[code]
}

type stubPrefixDeleter struct {
	prefixes []string
	err      error
}

func (s *stubPrefixDeleter) DeletePrefix(_ context.Context, prefix string) error {
	s.prefixes = append(s.prefixes, prefix)
	return s.err
}

func TestCleanupSpecies_DeletesR2PrefixesAndDBRows(t *testing.T) {
	q := &stubCleanupStore{}
	r2d := &stubPrefixDeleter{}
	cleanupSpecies(context.Background(), &bytes.Buffer{}, q, r2d, []string{"sonspa", "foxspa"})

	wantPrefixes := []string{"recordings/sonspa/", "images/sonspa/", "recordings/foxspa/", "images/foxspa/"}
	if !slices.Equal(r2d.prefixes, wantPrefixes) {
		t.Errorf("Should delete R2 prefixes %v, got %v", wantPrefixes, r2d.prefixes)
	}
	want := []string{"sonspa", "foxspa"}
	if !slices.Equal(q.recDeletes, want) || !slices.Equal(q.imgDeletes, want) || !slices.Equal(q.spDeletes, want) {
		t.Errorf("Should delete recordings/images/species rows for %v, got %v/%v/%v",
			want, q.recDeletes, q.imgDeletes, q.spDeletes)
	}
}

func TestCleanupSpecies_NilR2_SkipsPrefixesStillDeletesRows(t *testing.T) {
	q := &stubCleanupStore{}
	cleanupSpecies(context.Background(), &bytes.Buffer{}, q, nil, []string{"sonspa"})

	if !slices.Equal(q.spDeletes, []string{"sonspa"}) {
		t.Errorf("Should still delete DB rows without an R2 client, got %v", q.spDeletes)
	}
}

func TestCleanupSpecies_WarnsAndContinuesPastErrors(t *testing.T) {
	q := &stubCleanupStore{
		recErr: map[string]error{"sonspa": errors.New("rec boom")},
		spErr:  map[string]error{"sonspa": errors.New("sp boom")},
	}
	r2d := &stubPrefixDeleter{err: errors.New("r2 boom")}
	var buf bytes.Buffer
	cleanupSpecies(context.Background(), &buf, q, r2d, []string{"sonspa", "foxspa"})

	// Every step still attempted for every species despite errors.
	want := []string{"sonspa", "foxspa"}
	if !slices.Equal(q.spDeletes, want) {
		t.Errorf("Should keep deleting after errors, got %v", q.spDeletes)
	}
	out := buf.String()
	for _, frag := range []string{"rec boom", "sp boom", "r2 boom"} {
		if !strings.Contains(out, frag) {
			t.Errorf("Should warn about %q, output:\n%s", frag, out)
		}
	}
}
