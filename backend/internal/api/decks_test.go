package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deckStubQuerier stubs only deck-related methods.
type deckStubQuerier struct {
	store.Querier
	listUserDecks            func(ctx context.Context, userID int64) ([]store.ListUserDecksRow, error)
	createDeck               func(ctx context.Context, arg store.CreateDeckParams) (store.Deck, error)
	createPresetDeck         func(ctx context.Context, arg store.CreatePresetDeckParams) (store.Deck, error)
	getDeck                  func(ctx context.Context, id int64) (store.Deck, error)
	updateDeck               func(ctx context.Context, arg store.UpdateDeckParams) (store.Deck, error)
	deleteDeck               func(ctx context.Context, id int64) error
	listDeckSpeciesWithPrefs func(ctx context.Context, arg store.ListDeckSpeciesWithPrefsParams) ([]store.ListDeckSpeciesWithPrefsRow, error)
	addSpeciesToDeck         func(ctx context.Context, arg store.AddSpeciesToDeckParams) error
	removeSpeciesFromDeck    func(ctx context.Context, arg store.RemoveSpeciesFromDeckParams) error
	upsertCard               func(ctx context.Context, arg store.UpsertCardParams) error
	getNextDueAt             func(ctx context.Context, userID int64) (pgtype.Timestamptz, error)
	listPresetDecks          func(ctx context.Context) ([]store.ListPresetDecksRow, error)
	cloneDeckSpecies         func(ctx context.Context, arg store.CloneDeckSpeciesParams) error
	upsertCardsForDeck       func(ctx context.Context, arg store.UpsertCardsForDeckParams) error
	bulkAddSpeciesToDeck     func(ctx context.Context, arg store.BulkAddSpeciesToDeckParams) (int64, error)
	bulkUpsertCards          func(ctx context.Context, arg store.BulkUpsertCardsParams) error
}

func (s *deckStubQuerier) ListUserDecks(ctx context.Context, userID int64) ([]store.ListUserDecksRow, error) {
	return s.listUserDecks(ctx, userID)
}
func (s *deckStubQuerier) CreateDeck(ctx context.Context, arg store.CreateDeckParams) (store.Deck, error) {
	return s.createDeck(ctx, arg)
}
func (s *deckStubQuerier) GetDeck(ctx context.Context, id int64) (store.Deck, error) {
	return s.getDeck(ctx, id)
}
func (s *deckStubQuerier) UpdateDeck(ctx context.Context, arg store.UpdateDeckParams) (store.Deck, error) {
	return s.updateDeck(ctx, arg)
}
func (s *deckStubQuerier) DeleteDeck(ctx context.Context, id int64) error {
	return s.deleteDeck(ctx, id)
}
func (s *deckStubQuerier) ListDeckSpeciesWithPrefs(ctx context.Context, arg store.ListDeckSpeciesWithPrefsParams) ([]store.ListDeckSpeciesWithPrefsRow, error) {
	return s.listDeckSpeciesWithPrefs(ctx, arg)
}
func (s *deckStubQuerier) AddSpeciesToDeck(ctx context.Context, arg store.AddSpeciesToDeckParams) error {
	return s.addSpeciesToDeck(ctx, arg)
}
func (s *deckStubQuerier) RemoveSpeciesFromDeck(ctx context.Context, arg store.RemoveSpeciesFromDeckParams) error {
	return s.removeSpeciesFromDeck(ctx, arg)
}
func (s *deckStubQuerier) UpsertCard(ctx context.Context, arg store.UpsertCardParams) error {
	return s.upsertCard(ctx, arg)
}
func (s *deckStubQuerier) GetNextDueAt(ctx context.Context, userID int64) (pgtype.Timestamptz, error) {
	if s.getNextDueAt != nil {
		return s.getNextDueAt(ctx, userID)
	}
	return pgtype.Timestamptz{}, nil
}
func (s *deckStubQuerier) ListPresetDecks(ctx context.Context) ([]store.ListPresetDecksRow, error) {
	return s.listPresetDecks(ctx)
}
func (s *deckStubQuerier) CreatePresetDeck(ctx context.Context, arg store.CreatePresetDeckParams) (store.Deck, error) {
	return s.createPresetDeck(ctx, arg)
}
func (s *deckStubQuerier) CloneDeckSpecies(ctx context.Context, arg store.CloneDeckSpeciesParams) error {
	return s.cloneDeckSpecies(ctx, arg)
}
func (s *deckStubQuerier) UpsertCardsForDeck(ctx context.Context, arg store.UpsertCardsForDeckParams) error {
	return s.upsertCardsForDeck(ctx, arg)
}
func (s *deckStubQuerier) BulkAddSpeciesToDeck(ctx context.Context, arg store.BulkAddSpeciesToDeckParams) (int64, error) {
	return s.bulkAddSpeciesToDeck(ctx, arg)
}
func (s *deckStubQuerier) BulkUpsertCards(ctx context.Context, arg store.BulkUpsertCardsParams) error {
	return s.bulkUpsertCards(ctx, arg)
}

func ownerID(id int64) pgtype.Int8 {
	return pgtype.Int8{Int64: id, Valid: true}
}

func TestListDecks_ReturnsList(t *testing.T) {
	q := &deckStubQuerier{
		listUserDecks: func(_ context.Context, userID int64) ([]store.ListUserDecksRow, error) {
			assert.Equal(t, int64(1), userID)
			return []store.ListUserDecksRow{
				{ID: 1, Name: "My Warblers", OwnerID: ownerID(1), AudioDue: 3, ImageDue: 1},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listDecks(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listDecksResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Len(t, body.Decks, 1)
	assert.Equal(t, "My Warblers", body.Decks[0].Name)
	assert.Equal(t, int64(3), body.Decks[0].AudioDue)
	assert.Nil(t, body.NextDueAt)
}

func TestListDecks_NextDueAtSetWhenFutureCardExists(t *testing.T) {
	futureTime := time.Now().Add(2 * time.Hour)
	q := &deckStubQuerier{
		listUserDecks: func(_ context.Context, _ int64) ([]store.ListUserDecksRow, error) {
			return []store.ListUserDecksRow{}, nil
		},
		getNextDueAt: func(_ context.Context, _ int64) (pgtype.Timestamptz, error) {
			return pgtype.Timestamptz{Time: futureTime, Valid: true}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listDecks(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listDecksResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.NotNil(t, body.NextDueAt)
	parsed, err := time.Parse(time.RFC3339, *body.NextDueAt)
	require.NoError(t, err)
	assert.WithinDuration(t, futureTime, parsed, time.Second)
}

func TestListDecks_NextDueAtNullWhenNoFutureCards(t *testing.T) {
	q := &deckStubQuerier{
		listUserDecks: func(_ context.Context, _ int64) ([]store.ListUserDecksRow, error) {
			return []store.ListUserDecksRow{}, nil
		},
		getNextDueAt: func(_ context.Context, _ int64) (pgtype.Timestamptz, error) {
			return pgtype.Timestamptz{}, pgx.ErrNoRows
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listDecks(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body listDecksResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Nil(t, body.NextDueAt)
}

func TestCreateDeck_ReturnsDeck(t *testing.T) {
	q := &deckStubQuerier{
		createDeck: func(_ context.Context, arg store.CreateDeckParams) (store.Deck, error) {
			assert.Equal(t, "Pacific Northwest", arg.Name)
			assert.Equal(t, int64(1), arg.OwnerID.Int64)
			return store.Deck{ID: 42, Name: "Pacific Northwest", OwnerID: ownerID(1)}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":"Pacific Northwest"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/decks", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.createDeck(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	var got store.Deck
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, int64(42), got.ID)
	assert.Equal(t, "Pacific Northwest", got.Name)
}

func TestCreateDeck_EmptyName_Returns400(t *testing.T) {
	h := makeHandler(&deckStubQuerier{})
	body := `{"name":""}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/decks", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.createDeck(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateDeck_RenamesDeck(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, Name: "Old Name", OwnerID: ownerID(1)}, nil
		},
		updateDeck: func(_ context.Context, arg store.UpdateDeckParams) (store.Deck, error) {
			assert.Equal(t, int64(42), arg.ID)
			assert.Equal(t, "New Name", arg.Name)
			return store.Deck{ID: 42, Name: "New Name", OwnerID: ownerID(1)}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":"New Name"}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/decks/42", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.updateDeck(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var got store.Deck
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, "New Name", got.Name)
}

func TestUpdateDeck_PassesDescriptionToStore(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, Name: "My Deck", OwnerID: ownerID(1)}, nil
		},
		updateDeck: func(_ context.Context, arg store.UpdateDeckParams) (store.Deck, error) {
			assert.Equal(t, "Updated Name", arg.Name)
			assert.Equal(t, "Birds that look alike", arg.Description)
			return store.Deck{ID: 42, Name: "Updated Name", Description: "Birds that look alike", OwnerID: ownerID(1)}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":"Updated Name","description":"Birds that look alike"}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/decks/42", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.updateDeck(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var got store.Deck
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, "Birds that look alike", got.Description)
}

func TestCreateDeck_PassesDescriptionToStore(t *testing.T) {
	q := &deckStubQuerier{
		createDeck: func(_ context.Context, arg store.CreateDeckParams) (store.Deck, error) {
			assert.Equal(t, "Confusing Woodpeckers", arg.Name)
			assert.Equal(t, "All with that laughing rattle call", arg.Description)
			assert.Equal(t, int64(1), arg.OwnerID.Int64)
			return store.Deck{ID: 7, Name: "Confusing Woodpeckers", Description: "All with that laughing rattle call", OwnerID: ownerID(1)}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":"Confusing Woodpeckers","description":"All with that laughing rattle call"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/decks", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.createDeck(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	var got store.Deck
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, "All with that laughing rattle call", got.Description)
}

func TestUpdateDeck_WrongOwner_Returns403(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, OwnerID: ownerID(999)}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":"New Name"}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/decks/42", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.updateDeck(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateDeck_EmptyName_Returns400(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, OwnerID: ownerID(1)}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":""}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/decks/42", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.updateDeck(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateDeck_InvalidDeckID_Returns400(t *testing.T) {
	h := makeHandler(&deckStubQuerier{})
	body := `{"name":"New Name"}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/decks/notanumber", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "notanumber")
	w := httptest.NewRecorder()

	h.updateDeck(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateDeck_InvalidBody_Returns400(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, OwnerID: ownerID(1)}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/decks/42", strings.NewReader("not-json"))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.updateDeck(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteDeck_DeletesDeck(t *testing.T) {
	deleted := false
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, OwnerID: ownerID(1)}, nil
		},
		deleteDeck: func(_ context.Context, id int64) error {
			assert.Equal(t, int64(42), id)
			deleted = true
			return nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/decks/42", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.deleteDeck(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, deleted)
}

func TestDeleteDeck_NotFound_Returns404(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{}, pgx.ErrNoRows
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/decks/99", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "99")
	w := httptest.NewRecorder()

	h.deleteDeck(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListDeckSpecies_ReturnsList(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, OwnerID: ownerID(1)}, nil
		},
		listDeckSpeciesWithPrefs: func(_ context.Context, arg store.ListDeckSpeciesWithPrefsParams) ([]store.ListDeckSpeciesWithPrefsRow, error) {
			assert.Equal(t, int64(42), arg.DeckID)
			assert.Equal(t, int64(1), arg.UserID)
			return []store.ListDeckSpeciesWithPrefsRow{
				{EbirdCode: "sonspa", CommonName: "Song Sparrow", ScientificName: "Melospiza melodia", AudioEnabled: true, ImageEnabled: true},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/species", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.listDeckSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []store.ListDeckSpeciesWithPrefsRow
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Len(t, body, 1)
	assert.Equal(t, "Song Sparrow", body[0].CommonName)
	assert.True(t, body[0].AudioEnabled)
	assert.True(t, body[0].ImageEnabled)
}

func TestListDeckSpecies_PassesThroughDisabledLanes(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, OwnerID: ownerID(1)}, nil
		},
		listDeckSpeciesWithPrefs: func(_ context.Context, arg store.ListDeckSpeciesWithPrefsParams) ([]store.ListDeckSpeciesWithPrefsRow, error) {
			assert.Equal(t, int64(1), arg.UserID)
			return []store.ListDeckSpeciesWithPrefsRow{
				{EbirdCode: "amcro", CommonName: "American Crow", ScientificName: "Corvus brachyrhynchos", AudioEnabled: false, ImageEnabled: true},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/species", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.listDeckSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []store.ListDeckSpeciesWithPrefsRow
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Len(t, body, 1)
	assert.False(t, body[0].AudioEnabled)
	assert.True(t, body[0].ImageEnabled)
}

func TestAddSpeciesToDeck_InsertsAndUpsertsBothCards(t *testing.T) {
	upsertedLanes := []string{}
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, OwnerID: ownerID(1)}, nil
		},
		addSpeciesToDeck: func(_ context.Context, arg store.AddSpeciesToDeckParams) error {
			assert.Equal(t, int64(42), arg.DeckID)
			assert.Equal(t, "sonspa", arg.SpeciesCode)
			return nil
		},
		upsertCard: func(_ context.Context, arg store.UpsertCardParams) error {
			upsertedLanes = append(upsertedLanes, arg.Lane)
			return nil
		},
	}
	h := makeHandler(q)
	body := `{"ebird_code":"sonspa"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/decks/42/species", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.addSpeciesToDeck(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.ElementsMatch(t, []string{"audio", "image"}, upsertedLanes)
}

func TestRemoveSpeciesFromDeck_RemovesEntry(t *testing.T) {
	removed := false
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, OwnerID: ownerID(1)}, nil
		},
		removeSpeciesFromDeck: func(_ context.Context, arg store.RemoveSpeciesFromDeckParams) error {
			assert.Equal(t, int64(42), arg.DeckID)
			assert.Equal(t, "sonspa", arg.SpeciesCode)
			removed = true
			return nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/decks/42/species/sonspa", nil)
	r = injectUserID(r, 1)
	r = withChiParams(r, map[string]string{"id": "42", "ebird_code": "sonspa"})
	w := httptest.NewRecorder()

	h.removeSpeciesFromDeck(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, removed)
}

func TestListPresetDecks_ReturnsList(t *testing.T) {
	q := &deckStubQuerier{
		listPresetDecks: func(_ context.Context) ([]store.ListPresetDecksRow, error) {
			return []store.ListPresetDecksRow{
				{ID: 1, Name: "Confusing Woodpeckers", Description: "Those rattle calls", SpeciesCount: 5},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/presets", nil)
	w := httptest.NewRecorder()

	h.listPresetDecks(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []store.ListPresetDecksRow
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Len(t, body, 1)
	assert.Equal(t, "Confusing Woodpeckers", body[0].Name)
	assert.Equal(t, int64(5), body[0].SpeciesCount)
}

func TestCloneDeck_ClonesPresetForUser(t *testing.T) {
	var clonedFrom, clonedTo int64
	var cardsForDeck store.UpsertCardsForDeckParams
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			// preset: no owner
			return store.Deck{ID: id, Name: "Confusing Woodpeckers", Description: "Rattle calls"}, nil
		},
		createDeck: func(_ context.Context, arg store.CreateDeckParams) (store.Deck, error) {
			assert.Equal(t, "Confusing Woodpeckers", arg.Name)
			assert.Equal(t, "Rattle calls", arg.Description)
			assert.Equal(t, int64(1), arg.OwnerID.Int64)
			return store.Deck{ID: 99, Name: arg.Name, Description: arg.Description, OwnerID: ownerID(1)}, nil
		},
		cloneDeckSpecies: func(_ context.Context, arg store.CloneDeckSpeciesParams) error {
			clonedFrom = arg.DeckID
			clonedTo = arg.DeckID_2
			return nil
		},
		upsertCardsForDeck: func(_ context.Context, arg store.UpsertCardsForDeckParams) error {
			cardsForDeck = arg
			return nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/decks/7/clone", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "7")
	w := httptest.NewRecorder()

	h.cloneDeck(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	var got store.Deck
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, int64(99), got.ID)
	assert.Equal(t, int64(7), clonedFrom)
	assert.Equal(t, int64(99), clonedTo)
	assert.Equal(t, int64(1), cardsForDeck.UserID)
	assert.Equal(t, int64(99), cardsForDeck.DeckID)
}

func TestCloneDeck_UserOwnedDeck_Returns400(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, OwnerID: ownerID(1)}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/decks/7/clone", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "7")
	w := httptest.NewRecorder()

	h.cloneDeck(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCloneDeck_NotFound_Returns404(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, _ int64) (store.Deck, error) {
			return store.Deck{}, pgx.ErrNoRows
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/decks/99/clone", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "99")
	w := httptest.NewRecorder()

	h.cloneDeck(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateDeck_AdminCanUpdatePresetDeck(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			// preset: no owner
			return store.Deck{ID: id, Name: "Old Name"}, nil
		},
		updateDeck: func(_ context.Context, arg store.UpdateDeckParams) (store.Deck, error) {
			return store.Deck{ID: 7, Name: arg.Name}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":"New Name"}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/decks/7", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectAdmin(r, 1)
	r = withChiParam(r, "id", "7")
	w := httptest.NewRecorder()

	h.updateDeck(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBulkAddSpeciesToDeck_AddsAllSpeciesAndCreatesCards(t *testing.T) {
	var bulkArg store.BulkAddSpeciesToDeckParams
	var cardsArg store.BulkUpsertCardsParams
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, OwnerID: ownerID(1)}, nil
		},
		bulkAddSpeciesToDeck: func(_ context.Context, arg store.BulkAddSpeciesToDeckParams) (int64, error) {
			bulkArg = arg
			return 2, nil
		},
		bulkUpsertCards: func(_ context.Context, arg store.BulkUpsertCardsParams) error {
			cardsArg = arg
			return nil
		},
	}
	h := makeHandler(q)
	body := `{"species_codes":["amcro","sonspa"]}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/decks/42/species/bulk", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.bulkAddSpeciesToDeck(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int64(42), bulkArg.DeckID)
	assert.ElementsMatch(t, []string{"amcro", "sonspa"}, bulkArg.Column2)
	assert.Equal(t, int64(1), cardsArg.UserID)
	assert.ElementsMatch(t, []string{"amcro", "sonspa"}, cardsArg.Column2)
	var resp map[string]int64
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, int64(2), resp["added"])
}

func TestBulkAddSpeciesToDeck_EmptyList_Returns400(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, OwnerID: ownerID(1)}, nil
		},
	}
	h := makeHandler(q)
	body := `{"species_codes":[]}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/decks/42/species/bulk", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.bulkAddSpeciesToDeck(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkAddSpeciesToDeck_WrongOwner_Returns403(t *testing.T) {
	q := &deckStubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{ID: id, OwnerID: ownerID(999)}, nil
		},
	}
	h := makeHandler(q)
	body := `{"species_codes":["amcro"]}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/decks/42/species/bulk", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.bulkAddSpeciesToDeck(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
