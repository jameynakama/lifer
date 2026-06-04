package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/r2"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type adminStubQuerier struct {
	store.Querier
	getImageByID         func(ctx context.Context, macaulayID string) (store.GetImageByIDRow, error)
	getRecordingByID     func(ctx context.Context, xenoCantoID string) (store.GetRecordingByIDRow, error)
	getSpeciesImages     func(ctx context.Context, speciesCode string) ([]store.GetSpeciesImagesRow, error)
	getSpeciesRecordings func(ctx context.Context, speciesCode string) ([]store.GetSpeciesRecordingsRow, error)
	deleteImage          func(ctx context.Context, macaulayID string) error
	deleteRecording      func(ctx context.Context, xenoCantoID string) error
	upsertSpeciesImage   func(ctx context.Context, arg store.UpsertSpeciesImageParams) (store.SpeciesImage, error)
	upsertRecording      func(ctx context.Context, arg store.UpsertRecordingParams) (store.SpeciesRecording, error)
	setImageLocked       func(ctx context.Context, arg store.SetImageLockedParams) error
	setRecordingLocked   func(ctx context.Context, arg store.SetRecordingLockedParams) error
	getUsers             func(ctx context.Context) ([]store.User, error)
	setUserIsAdmin       func(ctx context.Context, arg store.SetUserIsAdminParams) error
	listAllUserDecks     func(ctx context.Context) ([]store.ListAllUserDecksRow, error)
	getDeckWithOwner     func(ctx context.Context, id int64) (store.GetDeckWithOwnerRow, error)
	listDeckSpeciesSimple func(ctx context.Context, deckID int64) ([]store.ListDeckSpeciesSimpleRow, error)
}

func (s *adminStubQuerier) GetImageByID(ctx context.Context, macaulayID string) (store.GetImageByIDRow, error) {
	return s.getImageByID(ctx, macaulayID)
}
func (s *adminStubQuerier) GetRecordingByID(ctx context.Context, xenoCantoID string) (store.GetRecordingByIDRow, error) {
	return s.getRecordingByID(ctx, xenoCantoID)
}
func (s *adminStubQuerier) SetImageLocked(ctx context.Context, arg store.SetImageLockedParams) error {
	if s.setImageLocked != nil {
		return s.setImageLocked(ctx, arg)
	}
	return nil
}
func (s *adminStubQuerier) SetRecordingLocked(ctx context.Context, arg store.SetRecordingLockedParams) error {
	if s.setRecordingLocked != nil {
		return s.setRecordingLocked(ctx, arg)
	}
	return nil
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
func (s *adminStubQuerier) GetUsers(ctx context.Context) ([]store.User, error) {
	return s.getUsers(ctx)
}
func (s *adminStubQuerier) SetUserIsAdmin(ctx context.Context, arg store.SetUserIsAdminParams) error {
	if s.setUserIsAdmin != nil {
		return s.setUserIsAdmin(ctx, arg)
	}
	return nil
}
func (s *adminStubQuerier) ListAllUserDecks(ctx context.Context) ([]store.ListAllUserDecksRow, error) {
	if s.listAllUserDecks != nil {
		return s.listAllUserDecks(ctx)
	}
	return nil, nil
}
func (s *adminStubQuerier) GetDeckWithOwner(ctx context.Context, id int64) (store.GetDeckWithOwnerRow, error) {
	if s.getDeckWithOwner != nil {
		return s.getDeckWithOwner(ctx, id)
	}
	return store.GetDeckWithOwnerRow{}, nil
}
func (s *adminStubQuerier) ListDeckSpeciesSimple(ctx context.Context, deckID int64) ([]store.ListDeckSpeciesSimpleRow, error) {
	if s.listDeckSpeciesSimple != nil {
		return s.listDeckSpeciesSimple(ctx, deckID)
	}
	return nil, nil
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

func TestAdminDeleteImage_DeletesFromR2AndDB(t *testing.T) {
	var deletedKey string
	var deletedID string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletedKey = r.URL.Path
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	r2c, err := r2.NewWithEndpoint(ts.URL, "key", "secret", "flockdeck", "https://pub.example.com")
	require.NoError(t, err)

	q := &adminStubQuerier{
		getImageByID: func(_ context.Context, id string) (store.GetImageByIDRow, error) {
			return store.GetImageByIDRow{
				MacaulayID: id,
				FilePath:   "https://pub.example.com/images/sonspa/img1.jpg",
			}, nil
		},
		deleteImage: func(_ context.Context, id string) error {
			deletedID = id
			return nil
		},
	}
	h := &Handler{queries: q, r2Client: r2c}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/species/sonspa/images/img1", nil)
	req = injectAdmin(req, 1)
	req = withChiParams(req, map[string]string{"ebird_code": "sonspa", "macaulay_id": "img1"})
	w := httptest.NewRecorder()

	h.adminDeleteImage(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "img1", deletedID)
	assert.Contains(t, deletedKey, "images/sonspa/img1.jpg")
}

func TestAdminDeleteRecording_DeletesFromR2AndDB(t *testing.T) {
	var deletedKey string
	var deletedID string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletedKey = r.URL.Path
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	r2c, err := r2.NewWithEndpoint(ts.URL, "key", "secret", "flockdeck", "https://pub.example.com")
	require.NoError(t, err)

	q := &adminStubQuerier{
		getRecordingByID: func(_ context.Context, id string) (store.GetRecordingByIDRow, error) {
			return store.GetRecordingByIDRow{
				XenoCantoID: id,
				FilePath:    "https://pub.example.com/recordings/sonspa/rec1.mp3",
			}, nil
		},
		deleteRecording: func(_ context.Context, id string) error {
			deletedID = id
			return nil
		},
	}
	h := &Handler{queries: q, r2Client: r2c}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/species/sonspa/recordings/rec1", nil)
	req = injectAdmin(req, 1)
	req = withChiParams(req, map[string]string{"ebird_code": "sonspa", "xeno_canto_id": "rec1"})
	w := httptest.NewRecorder()

	h.adminDeleteRecording(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "rec1", deletedID)
	assert.Contains(t, deletedKey, "recordings/sonspa/rec1.mp3")
}

func TestAdminUploadImage_UploadsToR2AndInsertsDB(t *testing.T) {
	var uploadedKey, uploadedContentType string
	var insertedParams store.UpsertSpeciesImageParams

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uploadedKey = r.URL.Path
		uploadedContentType = r.Header.Get("Content-Type")
		w.Header().Set("ETag", `"abc"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	r2c, err := r2.NewWithEndpoint(ts.URL, "key", "secret", "flockdeck", "https://pub.example.com")
	require.NoError(t, err)

	q := &adminStubQuerier{
		upsertSpeciesImage: func(_ context.Context, arg store.UpsertSpeciesImageParams) (store.SpeciesImage, error) {
			insertedParams = arg
			return store.SpeciesImage{MacaulayID: arg.MacaulayID}, nil
		},
	}
	h := &Handler{queries: q, r2Client: r2c}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "sparrow.jpg")
	fw.Write([]byte("fake image data"))
	mw.WriteField("credit", "John Doe")
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/species/sonspa/images", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req = injectAdmin(req, 1)
	req = withChiParam(req, "ebird_code", "sonspa")
	w := httptest.NewRecorder()

	h.adminUploadImage(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, uploadedKey, "images/sonspa/")
	assert.Contains(t, uploadedKey, ".jpg")
	assert.Equal(t, "image/jpeg", uploadedContentType)
	assert.Equal(t, "sonspa", insertedParams.SpeciesCode)
	assert.Equal(t, "John Doe", insertedParams.Credit)
	assert.True(t, strings.HasPrefix(insertedParams.MacaulayID, "admin-"))
}

func TestAdminUploadRecording_UploadsToR2AndInsertsDB(t *testing.T) {
	var uploadedKey string
	var insertedParams store.UpsertRecordingParams

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uploadedKey = r.URL.Path
		w.Header().Set("ETag", `"abc"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	r2c, err := r2.NewWithEndpoint(ts.URL, "key", "secret", "flockdeck", "https://pub.example.com")
	require.NoError(t, err)

	q := &adminStubQuerier{
		upsertRecording: func(_ context.Context, arg store.UpsertRecordingParams) (store.SpeciesRecording, error) {
			insertedParams = arg
			return store.SpeciesRecording{XenoCantoID: arg.XenoCantoID}, nil
		},
	}
	h := &Handler{queries: q, r2Client: r2c}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "sparrow.mp3")
	fw.Write([]byte("fake audio data"))
	mw.WriteField("quality", "A")
	mw.WriteField("type", "song")
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/species/sonspa/recordings", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req = injectAdmin(req, 1)
	req = withChiParam(req, "ebird_code", "sonspa")
	w := httptest.NewRecorder()

	h.adminUploadRecording(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, uploadedKey, "recordings/sonspa/")
	assert.Contains(t, uploadedKey, ".mp3")
	assert.Equal(t, "sonspa", insertedParams.SpeciesCode)
	assert.Equal(t, "A", insertedParams.Quality)
	assert.Equal(t, "song", insertedParams.Type)
	assert.True(t, strings.HasPrefix(insertedParams.XenoCantoID, "admin-"))
}

func TestAdminCreatePresetDeck_CreatesAndReturns(t *testing.T) {
	q := &deckStubQuerier{
		createPresetDeck: func(_ context.Context, arg store.CreatePresetDeckParams) (store.Deck, error) {
			assert.Equal(t, "Confusing Woodpeckers", arg.Name)
			assert.Equal(t, "Those rattle calls", arg.Description)
			return store.Deck{ID: 1, Name: arg.Name, Description: arg.Description}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":"Confusing Woodpeckers","description":"Those rattle calls"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/decks", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectAdmin(r, 1)
	w := httptest.NewRecorder()

	h.adminCreatePresetDeck(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	var got store.Deck
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, "Confusing Woodpeckers", got.Name)
}

func TestAdminUpdatePresetDeck_Updates(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, Name: "Old Name"}, nil // preset: no owner
		},
		updateDeck: func(_ context.Context, arg store.UpdateDeckParams) (store.Deck, error) {
			return store.Deck{ID: arg.ID, Name: arg.Name, Description: arg.Description}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":"New Name","description":"Updated desc"}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/decks/1", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectAdmin(r, 1)
	r = withChiParam(r, "id", "1")
	w := httptest.NewRecorder()

	h.adminUpdatePresetDeck(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var got store.Deck
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, "New Name", got.Name)
	assert.Equal(t, "Updated desc", got.Description)
}

func TestAdminUpdatePresetDeck_UserOwnedDeck_Returns400(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, OwnerID: ownerID(99)}, nil // user-owned
		},
	}
	h := makeHandler(q)
	body := `{"name":"New Name"}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/decks/1", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectAdmin(r, 1)
	r = withChiParam(r, "id", "1")
	w := httptest.NewRecorder()

	h.adminUpdatePresetDeck(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdminDeletePresetDeck_Deletes(t *testing.T) {
	deleted := false
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, Name: "Preset"}, nil // preset: no owner
		},
		deleteDeck: func(_ context.Context, id int64) error {
			assert.Equal(t, int64(1), id)
			deleted = true
			return nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/decks/1", nil)
	r = injectAdmin(r, 1)
	r = withChiParam(r, "id", "1")
	w := httptest.NewRecorder()

	h.adminDeletePresetDeck(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, deleted)
}

func TestAdminDeletePresetDeck_UserOwnedDeck_Returns400(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, OwnerID: ownerID(99)}, nil // user-owned
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/decks/1", nil)
	r = injectAdmin(r, 1)
	r = withChiParam(r, "id", "1")
	w := httptest.NewRecorder()

	h.adminDeletePresetDeck(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdminGetUsers_ReturnsUsers(t *testing.T) {
	q := &adminStubQuerier{
		getUsers: func(_ context.Context) ([]store.User, error) {
			return []store.User{
				{ID: 1, Name: "Alice", Email: "alice@example.com", IsAdmin: true},
				{ID: 2, Name: "Bob", Email: "bob@example.com", IsAdmin: false},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	r = injectAdmin(r, 1)
	w := httptest.NewRecorder()

	h.adminGetUsers(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var got []store.User
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Len(t, got, 2)
	assert.Equal(t, "Alice", got[0].Name)
	assert.Equal(t, "Bob", got[1].Name)
}

func TestAdminGetUsers_StoreError_Returns500(t *testing.T) {
	q := &adminStubQuerier{
		getUsers: func(_ context.Context) ([]store.User, error) {
			return nil, errors.New("db down")
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	r = injectAdmin(r, 1)
	w := httptest.NewRecorder()

	h.adminGetUsers(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAdminSetUserIsAdmin_SetsAdmin(t *testing.T) {
	var called store.SetUserIsAdminParams
	q := &adminStubQuerier{
		setUserIsAdmin: func(_ context.Context, arg store.SetUserIsAdminParams) error {
			called = arg
			return nil
		},
	}
	h := makeHandler(q)
	body := `{"is_admin":true}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/42", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectAdmin(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.adminSetUserIsAdmin(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, int64(42), called.ID)
	assert.True(t, called.IsAdmin)
}

func TestAdminSetUserIsAdmin_RemovesAdmin(t *testing.T) {
	var called store.SetUserIsAdminParams
	q := &adminStubQuerier{
		setUserIsAdmin: func(_ context.Context, arg store.SetUserIsAdminParams) error {
			called = arg
			return nil
		},
	}
	h := makeHandler(q)
	body := `{"is_admin":false}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/7", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectAdmin(r, 1)
	r = withChiParam(r, "id", "7")
	w := httptest.NewRecorder()

	h.adminSetUserIsAdmin(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, int64(7), called.ID)
	assert.False(t, called.IsAdmin)
}

func TestAdminSetUserIsAdmin_InvalidID_Returns400(t *testing.T) {
	h := makeHandler(&adminStubQuerier{})
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/notanid", strings.NewReader(`{"is_admin":true}`))
	r.Header.Set("Content-Type", "application/json")
	r = injectAdmin(r, 1)
	r = withChiParam(r, "id", "notanid")
	w := httptest.NewRecorder()

	h.adminSetUserIsAdmin(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdminSetUserIsAdmin_InvalidBody_Returns400(t *testing.T) {
	h := makeHandler(&adminStubQuerier{})
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/1", strings.NewReader(`not json`))
	r.Header.Set("Content-Type", "application/json")
	r = injectAdmin(r, 1)
	r = withChiParam(r, "id", "1")
	w := httptest.NewRecorder()

	h.adminSetUserIsAdmin(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdminSetUserIsAdmin_StoreError_Returns500(t *testing.T) {
	q := &adminStubQuerier{
		setUserIsAdmin: func(_ context.Context, arg store.SetUserIsAdminParams) error {
			return errors.New("db down")
		},
	}
	h := makeHandler(q)
	body := `{"is_admin":true}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/1", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectAdmin(r, 1)
	r = withChiParam(r, "id", "1")
	w := httptest.NewRecorder()

	h.adminSetUserIsAdmin(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAdminListUserDecks_ReturnsList(t *testing.T) {
	q := &adminStubQuerier{
		listAllUserDecks: func(_ context.Context) ([]store.ListAllUserDecksRow, error) {
			return []store.ListAllUserDecksRow{
				{ID: 1, Name: "My Warblers", OwnerName: "Alice", OwnerEmail: "alice@example.com", SpeciesCount: 12},
				{ID: 2, Name: "Shore Birds", OwnerName: "Bob", OwnerEmail: "bob@example.com", SpeciesCount: 7},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/decks", nil)
	r = injectAdmin(r, 1)
	w := httptest.NewRecorder()

	h.adminListUserDecks(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var got []store.ListAllUserDecksRow
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Len(t, got, 2)
	assert.Equal(t, "Alice", got[0].OwnerName)
}

func TestAdminListUserDecks_Empty_ReturnsEmptyArray(t *testing.T) {
	q := &adminStubQuerier{
		listAllUserDecks: func(_ context.Context) ([]store.ListAllUserDecksRow, error) {
			return nil, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/decks", nil)
	r = injectAdmin(r, 1)
	w := httptest.NewRecorder()

	h.adminListUserDecks(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var got []store.ListAllUserDecksRow
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.NotNil(t, got)
	assert.Len(t, got, 0)
}

func TestAdminListUserDecks_StoreError_Returns500(t *testing.T) {
	q := &adminStubQuerier{
		listAllUserDecks: func(_ context.Context) ([]store.ListAllUserDecksRow, error) {
			return nil, errors.New("db down")
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/decks", nil)
	r = injectAdmin(r, 1)
	w := httptest.NewRecorder()

	h.adminListUserDecks(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
