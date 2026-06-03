package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type adminStubQuerier struct {
	store.Querier
	getImageByID         func(ctx context.Context, macaulayID string) (store.SpeciesImage, error)
	getRecordingByID     func(ctx context.Context, xenoCantoID string) (store.SpeciesRecording, error)
	getSpeciesImages     func(ctx context.Context, speciesCode string) ([]store.GetSpeciesImagesRow, error)
	getSpeciesRecordings func(ctx context.Context, speciesCode string) ([]store.GetSpeciesRecordingsRow, error)
	deleteImage          func(ctx context.Context, macaulayID string) error
	deleteRecording      func(ctx context.Context, xenoCantoID string) error
	upsertSpeciesImage   func(ctx context.Context, arg store.UpsertSpeciesImageParams) (store.SpeciesImage, error)
	upsertRecording      func(ctx context.Context, arg store.UpsertRecordingParams) (store.SpeciesRecording, error)
}

func (s *adminStubQuerier) GetImageByID(ctx context.Context, macaulayID string) (store.SpeciesImage, error) {
	return s.getImageByID(ctx, macaulayID)
}
func (s *adminStubQuerier) GetRecordingByID(ctx context.Context, xenoCantoID string) (store.SpeciesRecording, error) {
	return s.getRecordingByID(ctx, xenoCantoID)
}
func (s *adminStubQuerier) GetSpeciesImages(ctx context.Context, speciesCode string) ([]store.GetSpeciesImagesRow, error) {
	return s.getSpeciesImages(ctx, speciesCode)
}
func (s *adminStubQuerier) GetSpeciesRecordings(ctx context.Context, speciesCode string) ([]store.GetSpeciesRecordingsRow, error) {
	return s.getSpeciesRecordings(ctx, speciesCode)
}
func (s *adminStubQuerier) DeleteImage(ctx context.Context, macaulayID string) error {
	return s.deleteImage(ctx, macaulayID)
}
func (s *adminStubQuerier) DeleteRecording(ctx context.Context, xenoCantoID string) error {
	return s.deleteRecording(ctx, xenoCantoID)
}
func (s *adminStubQuerier) UpsertSpeciesImage(ctx context.Context, arg store.UpsertSpeciesImageParams) (store.SpeciesImage, error) {
	return s.upsertSpeciesImage(ctx, arg)
}
func (s *adminStubQuerier) UpsertRecording(ctx context.Context, arg store.UpsertRecordingParams) (store.SpeciesRecording, error) {
	return s.upsertRecording(ctx, arg)
}

// injectAdmin sets userID and isAdmin=true in the request context.
func injectAdmin(r *http.Request, userID int64) *http.Request {
	ctx := context.WithValue(r.Context(), auth.UserIDKey(), userID)
	ctx = context.WithValue(ctx, auth.IsAdminKey(), true)
	return r.WithContext(ctx)
}

func TestAdminGetSpeciesDetail_ReturnsImagesAndRecordings(t *testing.T) {
	q := &adminStubQuerier{
		getSpeciesImages: func(_ context.Context, code string) ([]store.GetSpeciesImagesRow, error) {
			assert.Equal(t, "sonspa", code)
			return []store.GetSpeciesImagesRow{
				{MacaulayID: "img1", FilePath: "https://pub.example.com/images/sonspa/img1.jpg", Credit: "Photographer"},
			}, nil
		},
		getSpeciesRecordings: func(_ context.Context, code string) ([]store.GetSpeciesRecordingsRow, error) {
			assert.Equal(t, "sonspa", code)
			return []store.GetSpeciesRecordingsRow{
				{XenoCantoID: "rec1", FilePath: "https://pub.example.com/recordings/sonspa/rec1.mp3", Quality: "A", Type: "song"},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/species/sonspa", nil)
	r = injectAdmin(r, 1)
	r = withChiParam(r, "ebird_code", "sonspa")
	w := httptest.NewRecorder()

	h.adminGetSpeciesDetail(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Len(t, body["images"], 1)
	assert.Len(t, body["recordings"], 1)
}
