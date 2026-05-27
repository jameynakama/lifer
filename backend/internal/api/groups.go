package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jameynakama/lifer/internal/auth"
	"github.com/jameynakama/lifer/internal/store"
)

type createGroupRequest struct {
	Name string `json:"name"`
}

type updateGroupRequest struct {
	Name string `json:"name"`
}

type addSpeciesRequest struct {
	EbirdCode string `json:"ebird_code"`
}

// groupOwnerCheck fetches the group, writes 404/403 and returns false if the
// requesting user does not own it.
func (h *Handler) groupOwnerCheck(w http.ResponseWriter, r *http.Request, groupID, userID int64) bool {
	group, err := h.queries.GetGroup(r.Context(), groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return false
	}
	if err != nil {
		log.Printf("GetGroup error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return false
	}
	if !group.OwnerID.Valid || group.OwnerID.Int64 != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}

func (h *Handler) listGroups(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	groups, err := h.queries.ListUserGroups(r.Context(), userID)
	if err != nil {
		log.Printf("ListUserGroups error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if groups == nil {
		groups = []store.ListUserGroupsRow{}
	}
	writeJSON(w, http.StatusOK, groups)
}

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	group, err := h.queries.CreateGroup(r.Context(), store.CreateGroupParams{
		Name:    req.Name,
		OwnerID: pgtype.Int8{Int64: userID, Valid: true},
	})
	if err != nil {
		log.Printf("CreateGroup error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, group)
}

func (h *Handler) updateGroup(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	groupID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid group id", http.StatusBadRequest)
		return
	}

	if !h.groupOwnerCheck(w, r, groupID, userID) {
		return
	}

	var req updateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	group, err := h.queries.UpdateGroupName(r.Context(), store.UpdateGroupNameParams{
		ID:   groupID,
		Name: req.Name,
	})
	if err != nil {
		log.Printf("UpdateGroupName error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, group)
}

func (h *Handler) deleteGroup(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	groupID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid group id", http.StatusBadRequest)
		return
	}

	if !h.groupOwnerCheck(w, r, groupID, userID) {
		return
	}

	if err := h.queries.DeleteGroup(r.Context(), groupID); err != nil {
		log.Printf("DeleteGroup error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listGroupSpecies(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	groupID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid group id", http.StatusBadRequest)
		return
	}

	if !h.groupOwnerCheck(w, r, groupID, userID) {
		return
	}

	species, err := h.queries.ListGroupSpecies(r.Context(), groupID)
	if err != nil {
		log.Printf("ListGroupSpecies error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if species == nil {
		species = []store.ListGroupSpeciesRow{}
	}
	writeJSON(w, http.StatusOK, species)
}

func (h *Handler) addSpeciesToGroup(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	groupID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid group id", http.StatusBadRequest)
		return
	}

	if !h.groupOwnerCheck(w, r, groupID, userID) {
		return
	}

	var req addSpeciesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.EbirdCode == "" {
		http.Error(w, "ebird_code is required", http.StatusBadRequest)
		return
	}

	if err := h.queries.AddSpeciesToGroup(r.Context(), store.AddSpeciesToGroupParams{
		GroupID:     groupID,
		SpeciesCode: req.EbirdCode,
	}); err != nil {
		log.Printf("AddSpeciesToGroup error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	for _, lane := range []string{"audio", "image"} {
		if err := h.queries.UpsertCard(r.Context(), store.UpsertCardParams{
			UserID:      userID,
			SpeciesCode: req.EbirdCode,
			Lane:        lane,
		}); err != nil {
			log.Printf("UpsertCard error: %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) removeSpeciesFromGroup(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	groupID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid group id", http.StatusBadRequest)
		return
	}

	ebirdCode := chi.URLParam(r, "ebird_code")

	if !h.groupOwnerCheck(w, r, groupID, userID) {
		return
	}

	if err := h.queries.RemoveSpeciesFromGroup(r.Context(), store.RemoveSpeciesFromGroupParams{
		GroupID:     groupID,
		SpeciesCode: ebirdCode,
	}); err != nil {
		log.Printf("RemoveSpeciesFromGroup error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
