package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// prefStubQuerier handles preference + card methods only.
type prefStubQuerier struct {
	store.Querier
	upsertPreferences func(ctx context.Context, arg store.UpsertPreferencesParams) (store.UserSpeciesPreference, error)
	upsertCard        func(ctx context.Context, arg store.UpsertCardParams) error
	deleteCard        func(ctx context.Context, arg store.DeleteCardParams) error
}

func (s *prefStubQuerier) UpsertPreferences(ctx context.Context, arg store.UpsertPreferencesParams) (store.UserSpeciesPreference, error) {
	return s.upsertPreferences(ctx, arg)
}
func (s *prefStubQuerier) UpsertCard(ctx context.Context, arg store.UpsertCardParams) error {
	return s.upsertCard(ctx, arg)
}
func (s *prefStubQuerier) DeleteCard(ctx context.Context, arg store.DeleteCardParams) error {
	return s.deleteCard(ctx, arg)
}

func TestUpdatePreferences_EnablesBothLanes(t *testing.T) {
	upsertCalls := map[string]bool{}
	q := &prefStubQuerier{
		upsertPreferences: func(_ context.Context, arg store.UpsertPreferencesParams) (store.UserSpeciesPreference, error) {
			assert.Equal(t, int64(1), arg.UserID)
			assert.Equal(t, "busti", arg.SpeciesCode)
			assert.True(t, arg.AudioEnabled)
			assert.True(t, arg.ImageEnabled)
			return store.UserSpeciesPreference{
				UserID:       arg.UserID,
				SpeciesCode:  arg.SpeciesCode,
				AudioEnabled: arg.AudioEnabled,
				ImageEnabled: arg.ImageEnabled,
			}, nil
		},
		upsertCard: func(_ context.Context, arg store.UpsertCardParams) error {
			upsertCalls[arg.Lane] = true
			return nil
		},
		deleteCard: func(_ context.Context, _ store.DeleteCardParams) error {
			t.Fatal("deleteCard should not be called when enabling lanes")
			return nil
		},
	}

	h := makeHandler(q)
	body := `{"audio_enabled":true,"image_enabled":true}`
	r := httptest.NewRequest(http.MethodPut, "/api/v1/species/busti/preferences", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "ebird_code", "busti")
	w := httptest.NewRecorder()

	h.updatePreferences(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, upsertCalls["audio"], "should upsert audio card")
	assert.True(t, upsertCalls["image"], "should upsert image card")
	var pref store.UserSpeciesPreference
	require.NoError(t, json.NewDecoder(w.Body).Decode(&pref))
	assert.Equal(t, "busti", pref.SpeciesCode)
}

func TestUpdatePreferences_DisablesAudioLane(t *testing.T) {
	deleteCalls := map[string]bool{}
	q := &prefStubQuerier{
		upsertPreferences: func(_ context.Context, arg store.UpsertPreferencesParams) (store.UserSpeciesPreference, error) {
			return store.UserSpeciesPreference{
				UserID: arg.UserID, SpeciesCode: arg.SpeciesCode,
				AudioEnabled: arg.AudioEnabled, ImageEnabled: arg.ImageEnabled,
			}, nil
		},
		upsertCard: func(_ context.Context, arg store.UpsertCardParams) error {
			assert.Equal(t, "image", arg.Lane, "only image should be upserted")
			return nil
		},
		deleteCard: func(_ context.Context, arg store.DeleteCardParams) error {
			deleteCalls[arg.Lane] = true
			return nil
		},
	}

	h := makeHandler(q)
	body := `{"audio_enabled":false,"image_enabled":true}`
	r := httptest.NewRequest(http.MethodPut, "/api/v1/species/busti/preferences", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "ebird_code", "busti")
	w := httptest.NewRecorder()

	h.updatePreferences(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, deleteCalls["audio"], "should delete audio card")
	assert.False(t, deleteCalls["image"], "should not delete image card")
}
