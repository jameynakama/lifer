package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteError_EmitsJSONEnvelope(t *testing.T) {
	w := httptest.NewRecorder()

	writeError(w, http.StatusForbidden, "forbidden")

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "forbidden", body["error"])
}

func TestParseID_ValidID(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/decks/42", nil)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	id, ok := parseID(w, r, "id", "deck")

	assert.True(t, ok)
	assert.Equal(t, int64(42), id)
}

func TestParseID_Invalid_Writes400(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/decks/nope", nil)
	r = withChiParam(r, "id", "nope")
	w := httptest.NewRecorder()

	_, ok := parseID(w, r, "id", "deck")

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid deck id")
}

func TestDecodeJSON_ValidBody(t *testing.T) {
	type req struct {
		Name string `json:"name"`
	}
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Yard Birds"}`))
	w := httptest.NewRecorder()

	got, ok := decodeJSON[req](w, r)

	assert.True(t, ok)
	assert.Equal(t, "Yard Birds", got.Name)
}

func TestDecodeJSON_MalformedBody_Writes400(t *testing.T) {
	type req struct {
		Name string `json:"name"`
	}
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":`))
	w := httptest.NewRecorder()

	_, ok := decodeJSON[req](w, r)

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDecodeJSON_UnknownField_Writes400(t *testing.T) {
	type req struct {
		Name string `json:"name"`
	}
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"x","sneaky":true}`))
	w := httptest.NewRecorder()

	_, ok := decodeJSON[req](w, r)

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDecodeJSON_OversizedBody_Rejected(t *testing.T) {
	type req struct {
		Name string `json:"name"`
	}
	huge := `{"name":"` + strings.Repeat("x", 2<<20) + `"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(huge))
	w := httptest.NewRecorder()

	_, ok := decodeJSON[req](w, r)

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOrEmpty(t *testing.T) {
	assert.Equal(t, []int{}, orEmpty[int](nil), "nil becomes empty slice")
	assert.Equal(t, []int{1}, orEmpty([]int{1}), "non-nil passes through")
}
