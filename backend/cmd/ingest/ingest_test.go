package main

import (
	"context"
	"errors"
	"testing"

	"github.com/jameynakama/flockdeck/internal/macaulay"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/jameynakama/flockdeck/internal/xenocanto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubUpserter implements ingestStore and metadataStore; nil funcs fail the test if called.
type stubUpserter struct {
	t                  *testing.T
	upsertSpecies      func(arg store.UpsertSpeciesParams) (store.Species, error)
	upsertRecording    func(arg store.UpsertRecordingParams) (store.SpeciesRecording, error)
	upsertSpeciesImage func(arg store.UpsertSpeciesImageParams) (store.SpeciesImage, error)
	listSpeciesCodes   func(ctx context.Context) ([]string, error)
}

func (s *stubUpserter) UpsertSpecies(_ context.Context, arg store.UpsertSpeciesParams) (store.Species, error) {
	if s.upsertSpecies == nil {
		s.t.Fatal("unexpected UpsertSpecies call")
	}
	return s.upsertSpecies(arg)
}

func (s *stubUpserter) UpsertRecording(_ context.Context, arg store.UpsertRecordingParams) (store.SpeciesRecording, error) {
	if s.upsertRecording == nil {
		s.t.Fatal("unexpected UpsertRecording call")
	}
	return s.upsertRecording(arg)
}

func (s *stubUpserter) UpsertSpeciesImage(_ context.Context, arg store.UpsertSpeciesImageParams) (store.SpeciesImage, error) {
	if s.upsertSpeciesImage == nil {
		s.t.Fatal("unexpected UpsertSpeciesImage call")
	}
	return s.upsertSpeciesImage(arg)
}

func (s *stubUpserter) ListSpeciesCodes(ctx context.Context) ([]string, error) {
	if s.listSpeciesCodes == nil {
		s.t.Fatal("unexpected ListSpeciesCodes call")
	}
	return s.listSpeciesCodes(ctx)
}

func discard(any) {}

func TestUploadRecording_Success_UpsertsWithRecordingFields(t *testing.T) {
	var got store.UpsertRecordingParams
	q := &stubUpserter{t: t, upsertRecording: func(arg store.UpsertRecordingParams) (store.SpeciesRecording, error) {
		got = arg
		return store.SpeciesRecording{}, nil
	}}
	rec := xenocanto.Recording{ID: "123", Quality: "A", Type: "song", Rec: "Recordist", FileURL: "https://xc.example/123.mp3"}

	// r2c == nil takes the placeholder path: no network involved.
	err := uploadRecording(context.Background(), q, nil, rec, "spotto", 0, discard)

	require.NoError(t, err)
	assert.Equal(t, "123", got.XenoCantoID)
	assert.Equal(t, "spotto", got.SpeciesCode)
	assert.Equal(t, "A", got.Quality)
	assert.Equal(t, "song", got.Type)
	assert.Equal(t, "Recordist", got.Credit)
	assert.Equal(t, "placeholder://recordings/spotto/123.mp3", got.FilePath)
}

func TestUploadRecording_UpsertError_IsReturnedNotSwallowed(t *testing.T) {
	q := &stubUpserter{t: t, upsertRecording: func(store.UpsertRecordingParams) (store.SpeciesRecording, error) {
		return store.SpeciesRecording{}, errors.New("db down")
	}}
	rec := xenocanto.Recording{ID: "123", Quality: "A", Type: "song"}

	err := uploadRecording(context.Background(), q, nil, rec, "spotto", 0, discard)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}

func TestUploadImage_Success_UpsertsWithImageFields(t *testing.T) {
	var got store.UpsertSpeciesImageParams
	q := &stubUpserter{t: t, upsertSpeciesImage: func(arg store.UpsertSpeciesImageParams) (store.SpeciesImage, error) {
		got = arg
		return store.SpeciesImage{}, nil
	}}
	photo := macaulay.Photo{AssetID: "456", UserDisplayName: "Photographer"}

	err := uploadImage(context.Background(), q, nil, nil, photo, "spotto", 0, discard)

	require.NoError(t, err)
	assert.Equal(t, "456", got.MacaulayID)
	assert.Equal(t, "spotto", got.SpeciesCode)
	assert.Equal(t, "Photographer", got.Credit)
	assert.Equal(t, "placeholder://images/spotto/456.jpg", got.FilePath)
}

func TestPartitionProtected(t *testing.T) {
	tests := []struct {
		name           string
		codes          []string
		locked         []string
		wantDeletable  []string
		wantProtected  []string
	}{
		{
			name:          "locked species are split out",
			codes:         []string{"amebar", "spotto", "foxspa"},
			locked:        []string{"spotto"},
			wantDeletable: []string{"amebar", "foxspa"},
			wantProtected: []string{"spotto"},
		},
		{
			name:          "nothing locked",
			codes:         []string{"amebar", "spotto"},
			locked:        nil,
			wantDeletable: []string{"amebar", "spotto"},
			wantProtected: []string{},
		},
		{
			name:          "everything locked",
			codes:         []string{"amebar"},
			locked:        []string{"amebar", "unrelated"},
			wantDeletable: []string{},
			wantProtected: []string{"amebar"},
		},
		{
			name:          "empty input",
			codes:         nil,
			locked:        []string{"amebar"},
			wantDeletable: []string{},
			wantProtected: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locked := make(map[string]struct{}, len(tt.locked))
			for _, c := range tt.locked {
				locked[c] = struct{}{}
			}

			deletable, protected := partitionProtected(tt.codes, locked)

			assert.Equal(t, tt.wantDeletable, deletable)
			assert.Equal(t, tt.wantProtected, protected)
		})
	}
}

func TestUploadImage_UpsertError_IsReturnedNotSwallowed(t *testing.T) {
	q := &stubUpserter{t: t, upsertSpeciesImage: func(store.UpsertSpeciesImageParams) (store.SpeciesImage, error) {
		return store.SpeciesImage{}, errors.New("db down")
	}}
	photo := macaulay.Photo{AssetID: "456"}

	err := uploadImage(context.Background(), q, nil, nil, photo, "spotto", 0, discard)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}
