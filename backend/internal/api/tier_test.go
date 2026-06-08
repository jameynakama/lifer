package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStatsTier_RejectsUnknownStage(t *testing.T) {
	h := makeHandler(&stubQuerier{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/tier/dinosaur", nil)
	req = withChiParam(req, "stage", "dinosaur")
	req = injectUserID(req, 1)
	w := httptest.NewRecorder()
	h.getStatsTier(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetStatsTier_RejectsInvalidLane(t *testing.T) {
	h := makeHandler(&stubQuerier{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/tier/juvenile?lane=sonar", nil)
	req = withChiParam(req, "stage", "juvenile")
	req = injectUserID(req, 1)
	w := httptest.NewRecorder()
	h.getStatsTier(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetStatsTier_ReturnsBirdsInStage(t *testing.T) {
	q := &stubQuerier{
		getCardsInTier: func(_ context.Context, arg store.GetCardsInTierParams) ([]store.GetCardsInTierRow, error) {
			return []store.GetCardsInTierRow{
				{
					SpeciesCode:    "_tst1",
					CommonName:     "Test Species",
					ScientificName: "Testus specius",
					Lane:           "audio",
					Stability:      10,
				},
			}, nil
		},
	}

	h := makeHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/tier/juvenile", nil)
	req = withChiParam(req, "stage", "juvenile")
	req = injectUserID(req, 1)
	w := httptest.NewRecorder()

	h.getStatsTier(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var birds []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &birds))
	require.NotEmpty(t, birds)
	assert.Equal(t, "_tst1", birds[0]["ebird_code"])
	assert.Equal(t, "audio", birds[0]["lane"])
	assert.Equal(t, float64(10), birds[0]["stability"])
}

func TestGetStatsTier_EggStage_PassesEggFlag(t *testing.T) {
	var gotParams store.GetCardsInTierParams
	q := &stubQuerier{
		getCardsInTier: func(_ context.Context, arg store.GetCardsInTierParams) ([]store.GetCardsInTierRow, error) {
			gotParams = arg
			return []store.GetCardsInTierRow{}, nil
		},
	}

	h := makeHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/tier/egg", nil)
	req = withChiParam(req, "stage", "egg")
	req = injectUserID(req, 1)
	w := httptest.NewRecorder()

	h.getStatsTier(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, gotParams.Egg, "egg stage must pass Egg=true to query")
}

func TestGetStatsTier_LaneFilter_PassedToQuery(t *testing.T) {
	var gotParams store.GetCardsInTierParams
	q := &stubQuerier{
		getCardsInTier: func(_ context.Context, arg store.GetCardsInTierParams) ([]store.GetCardsInTierRow, error) {
			gotParams = arg
			return []store.GetCardsInTierRow{}, nil
		},
	}

	h := makeHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/tier/juvenile?lane=audio", nil)
	req = withChiParam(req, "stage", "juvenile")
	req = injectUserID(req, 1)
	w := httptest.NewRecorder()

	h.getStatsTier(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, gotParams.Lane.Valid, "lane filter must be passed to query")
	assert.Equal(t, "audio", gotParams.Lane.String)
}

func TestGetStatsTier_NoCards_ReturnsEmptyArray(t *testing.T) {
	q := &stubQuerier{
		getCardsInTier: func(_ context.Context, _ store.GetCardsInTierParams) ([]store.GetCardsInTierRow, error) {
			return []store.GetCardsInTierRow{}, nil
		},
	}

	h := makeHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/tier/adult", nil)
	req = withChiParam(req, "stage", "adult")
	req = injectUserID(req, 1)
	w := httptest.NewRecorder()

	h.getStatsTier(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var birds []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &birds))
	assert.Empty(t, birds, "empty stage should return an empty JSON array")
}
