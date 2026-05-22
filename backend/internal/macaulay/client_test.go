package macaulay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhotos(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/ref/media/best", r.URL.Path)
		assert.Equal(t, "soospa", r.URL.Query().Get("species"))
		assert.Equal(t, "photo", r.URL.Query().Get("mediaType"))
		assert.Equal(t, "testkey", r.Header.Get("X-eBirdApiToken"))
		json.NewEncoder(w).Encode([]Photo{
			{AssetID: "111111111", UserDisplayName: "Jane Birder"},
			{AssetID: "222222222", UserDisplayName: "John Watcher"},
			{AssetID: "333333333", UserDisplayName: "Alice Finch"},
			{AssetID: "444444444", UserDisplayName: "Bob Sparrow"},
		})
	}))
	defer srv.Close()

	c := newWithBaseURL("testkey", srv.URL)
	photos, err := c.Photos(context.Background(), "soospa", 3)
	require.NoError(t, err)
	assert.Len(t, photos, 3)
	assert.Equal(t, "111111111", photos[0].AssetID)
	assert.Equal(t, "Jane Birder", photos[0].UserDisplayName)
}

func TestPhotosHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := newWithBaseURL("bad", srv.URL)
	_, err := c.Photos(context.Background(), "soospa", 3)
	assert.ErrorContains(t, err, "403")
}

func TestPhotoURL(t *testing.T) {
	c := New("key")
	assert.Equal(t,
		"https://cdn.download.ams.birds.cornell.edu/api/v1/asset/12345678/large",
		c.PhotoURL("12345678"),
	)
}
