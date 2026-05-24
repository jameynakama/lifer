package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jameynakama/lifer/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// groupStubQuerier stubs only group-related methods.
type groupStubQuerier struct {
	store.Querier
	listUserGroups         func(ctx context.Context, userID int64) ([]store.ListUserGroupsRow, error)
	createGroup            func(ctx context.Context, arg store.CreateGroupParams) (store.Group, error)
	getGroup               func(ctx context.Context, id int64) (store.Group, error)
	updateGroupName        func(ctx context.Context, arg store.UpdateGroupNameParams) (store.Group, error)
	deleteGroup            func(ctx context.Context, id int64) error
	listGroupSpecies       func(ctx context.Context, groupID int64) ([]store.ListGroupSpeciesRow, error)
	addSpeciesToGroup      func(ctx context.Context, arg store.AddSpeciesToGroupParams) error
	removeSpeciesFromGroup func(ctx context.Context, arg store.RemoveSpeciesFromGroupParams) error
	upsertCard             func(ctx context.Context, arg store.UpsertCardParams) error
}

func (s *groupStubQuerier) ListUserGroups(ctx context.Context, userID int64) ([]store.ListUserGroupsRow, error) {
	return s.listUserGroups(ctx, userID)
}
func (s *groupStubQuerier) CreateGroup(ctx context.Context, arg store.CreateGroupParams) (store.Group, error) {
	return s.createGroup(ctx, arg)
}
func (s *groupStubQuerier) GetGroup(ctx context.Context, id int64) (store.Group, error) {
	return s.getGroup(ctx, id)
}
func (s *groupStubQuerier) UpdateGroupName(ctx context.Context, arg store.UpdateGroupNameParams) (store.Group, error) {
	return s.updateGroupName(ctx, arg)
}
func (s *groupStubQuerier) DeleteGroup(ctx context.Context, id int64) error {
	return s.deleteGroup(ctx, id)
}
func (s *groupStubQuerier) ListGroupSpecies(ctx context.Context, groupID int64) ([]store.ListGroupSpeciesRow, error) {
	return s.listGroupSpecies(ctx, groupID)
}
func (s *groupStubQuerier) AddSpeciesToGroup(ctx context.Context, arg store.AddSpeciesToGroupParams) error {
	return s.addSpeciesToGroup(ctx, arg)
}
func (s *groupStubQuerier) RemoveSpeciesFromGroup(ctx context.Context, arg store.RemoveSpeciesFromGroupParams) error {
	return s.removeSpeciesFromGroup(ctx, arg)
}
func (s *groupStubQuerier) UpsertCard(ctx context.Context, arg store.UpsertCardParams) error {
	return s.upsertCard(ctx, arg)
}

func ownerID(id int64) pgtype.Int8 {
	return pgtype.Int8{Int64: id, Valid: true}
}

func TestListGroups_ReturnsList(t *testing.T) {
	q := &groupStubQuerier{
		listUserGroups: func(_ context.Context, userID int64) ([]store.ListUserGroupsRow, error) {
			assert.Equal(t, int64(1), userID)
			return []store.ListUserGroupsRow{
				{ID: 1, Name: "My Warblers", OwnerID: ownerID(1), AudioDue: 3, ImageDue: 1},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listGroups(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []store.ListUserGroupsRow
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Len(t, body, 1)
	assert.Equal(t, "My Warblers", body[0].Name)
	assert.Equal(t, int64(3), body[0].AudioDue)
}

func TestCreateGroup_ReturnsGroup(t *testing.T) {
	q := &groupStubQuerier{
		createGroup: func(_ context.Context, arg store.CreateGroupParams) (store.Group, error) {
			assert.Equal(t, "Pacific Northwest", arg.Name)
			assert.Equal(t, int64(1), arg.OwnerID.Int64)
			return store.Group{ID: 42, Name: "Pacific Northwest", OwnerID: ownerID(1)}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":"Pacific Northwest"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/groups", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.createGroup(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	var got store.Group
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, int64(42), got.ID)
	assert.Equal(t, "Pacific Northwest", got.Name)
}

func TestCreateGroup_EmptyName_Returns400(t *testing.T) {
	h := makeHandler(&groupStubQuerier{})
	body := `{"name":""}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/groups", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.createGroup(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateGroup_RenamesGroup(t *testing.T) {
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{ID: id, Name: "Old Name", OwnerID: ownerID(1)}, nil
		},
		updateGroupName: func(_ context.Context, arg store.UpdateGroupNameParams) (store.Group, error) {
			assert.Equal(t, int64(42), arg.ID)
			assert.Equal(t, "New Name", arg.Name)
			return store.Group{ID: 42, Name: "New Name", OwnerID: ownerID(1)}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":"New Name"}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/groups/42", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.updateGroup(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var got store.Group
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, "New Name", got.Name)
}

func TestUpdateGroup_WrongOwner_Returns403(t *testing.T) {
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{ID: id, OwnerID: ownerID(999)}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":"New Name"}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/groups/42", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.updateGroup(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateGroup_EmptyName_Returns400(t *testing.T) {
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{ID: id, OwnerID: ownerID(1)}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":""}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/groups/42", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.updateGroup(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateGroup_InvalidGroupID_Returns400(t *testing.T) {
	h := makeHandler(&groupStubQuerier{})
	body := `{"name":"New Name"}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/groups/notanumber", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "notanumber")
	w := httptest.NewRecorder()

	h.updateGroup(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateGroup_InvalidBody_Returns400(t *testing.T) {
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{ID: id, OwnerID: ownerID(1)}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/groups/42", strings.NewReader("not-json"))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.updateGroup(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteGroup_DeletesGroup(t *testing.T) {
	deleted := false
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{ID: id, OwnerID: ownerID(1)}, nil
		},
		deleteGroup: func(_ context.Context, id int64) error {
			assert.Equal(t, int64(42), id)
			deleted = true
			return nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/42", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.deleteGroup(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, deleted)
}

func TestDeleteGroup_NotFound_Returns404(t *testing.T) {
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{}, pgx.ErrNoRows
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/99", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "99")
	w := httptest.NewRecorder()

	h.deleteGroup(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListGroupSpecies_ReturnsList(t *testing.T) {
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{ID: id, OwnerID: ownerID(1)}, nil
		},
		listGroupSpecies: func(_ context.Context, groupID int64) ([]store.ListGroupSpeciesRow, error) {
			assert.Equal(t, int64(42), groupID)
			return []store.ListGroupSpeciesRow{
				{ID: 7, CommonName: "Song Sparrow", ScientificName: "Melospiza melodia", EbirdCode: "sonspa"},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/species", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.listGroupSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []store.ListGroupSpeciesRow
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Len(t, body, 1)
	assert.Equal(t, "Song Sparrow", body[0].CommonName)
}

func TestAddSpeciesToGroup_InsertsAndUpsertsBothCards(t *testing.T) {
	upsertedLanes := []string{}
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{ID: id, OwnerID: ownerID(1)}, nil
		},
		addSpeciesToGroup: func(_ context.Context, arg store.AddSpeciesToGroupParams) error {
			assert.Equal(t, int64(42), arg.GroupID)
			assert.Equal(t, int64(7), arg.SpeciesID)
			return nil
		},
		upsertCard: func(_ context.Context, arg store.UpsertCardParams) error {
			upsertedLanes = append(upsertedLanes, arg.Lane)
			return nil
		},
	}
	h := makeHandler(q)
	body := `{"species_id":7}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/groups/42/species", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.addSpeciesToGroup(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.ElementsMatch(t, []string{"audio", "image"}, upsertedLanes)
}

func TestRemoveSpeciesFromGroup_RemovesEntry(t *testing.T) {
	removed := false
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{ID: id, OwnerID: ownerID(1)}, nil
		},
		removeSpeciesFromGroup: func(_ context.Context, arg store.RemoveSpeciesFromGroupParams) error {
			assert.Equal(t, int64(42), arg.GroupID)
			assert.Equal(t, int64(7), arg.SpeciesID)
			removed = true
			return nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/42/species/7", nil)
	r = injectUserID(r, 1)
	r = withChiParams(r, map[string]string{"id": "42", "species_id": "7"})
	w := httptest.NewRecorder()

	h.removeSpeciesFromGroup(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, removed)
}
