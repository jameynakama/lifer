package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jameynakama/lifer/internal/store"
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
