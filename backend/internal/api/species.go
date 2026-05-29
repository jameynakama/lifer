package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jameynakama/lifer/internal/auth"
	"github.com/jameynakama/lifer/internal/store"
)

const defaultPageSize = 20

// PaginatedSpecies is the response shape for GET /api/v1/species.
type PaginatedSpecies struct {
	Count    int64         `json:"count"`
	Next     *string       `json:"next"`
	Previous *string       `json:"previous"`
	Results  []SpeciesItem `json:"results"`
}

// SpeciesItem is a single species in a list response.
type SpeciesItem struct {
	EbirdCode      string  `json:"ebird_code"`
	CommonName     string  `json:"common_name"`
	ScientificName string  `json:"scientific_name"`
	ImageURL       *string `json:"image_url"`
}

// SpeciesDetail is the response for GET /api/v1/species/:ebird_code.
type SpeciesDetail struct {
	EbirdCode      string      `json:"ebird_code"`
	CommonName     string      `json:"common_name"`
	ScientificName string      `json:"scientific_name"`
	Recordings     []Recording `json:"recordings"`
	Images         []Image     `json:"images"`
}

// Recording is a single audio recording entry.
type Recording struct {
	XenoCantoID string `json:"xeno_canto_id"`
	FilePath    string `json:"file_path"`
	Quality     string `json:"quality"`
	Type        string `json:"type"`
}

// Image is a single photo entry.
type Image struct {
	MacaulayID string `json:"macaulay_id"`
	FilePath   string `json:"file_path"`
	Credit     string `json:"credit"`
}

// SpeciesGroupsResponse is the response for GET /api/v1/species/:ebird_code/groups.
type SpeciesGroupsResponse struct {
	GroupIDs []int64 `json:"group_ids"`
}

// listSpecies handles GET /api/v1/species.
// With q param: search mode (up to 50 results, no pagination links).
// Without q param: browse mode (paginated, limit/offset).
func (h *Handler) listSpecies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit := int32(defaultPageSize)
	offset := int32(0)

	if l := r.URL.Query().Get("limit"); l != "" {
		v, err := strconv.Atoi(l)
		if err != nil || v < 1 || v > 100 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = int32(v)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		v, err := strconv.Atoi(o)
		if err != nil || v < 0 {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
		offset = int32(v)
	}

	if q != "" {
		rows, err := h.queries.SearchSpecies(r.Context(), pgtype.Text{String: q, Valid: true})
		if err != nil {
			log.Printf("SearchSpecies error: %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		results := make([]SpeciesItem, len(rows))
		for i, row := range rows {
			var imageURL *string
			if row.ImageUrl != "" {
				imageURL = &row.ImageUrl
			}
			results[i] = SpeciesItem{
				EbirdCode:      row.EbirdCode,
				CommonName:     row.CommonName,
				ScientificName: row.ScientificName,
				ImageURL:       imageURL,
			}
		}
		writeJSON(w, http.StatusOK, PaginatedSpecies{
			Count:    int64(len(rows)),
			Next:     nil,
			Previous: nil,
			Results:  results,
		})
		return
	}

	rows, err := h.queries.ListSpecies(r.Context(), store.ListSpeciesParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		log.Printf("ListSpecies error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	results := make([]SpeciesItem, len(rows))
	var totalCount int64
	for i, row := range rows {
		var imageURL *string
		if row.ImageUrl != "" {
			imageURL = &row.ImageUrl
		}
		results[i] = SpeciesItem{
			EbirdCode:      row.EbirdCode,
			CommonName:     row.CommonName,
			ScientificName: row.ScientificName,
			ImageURL:       imageURL,
		}
		totalCount = row.TotalCount
	}

	var next, prev *string
	if int64(offset)+int64(limit) < totalCount {
		u := buildAbsoluteURL(r, offset+limit, limit)
		next = &u
	}
	if offset > 0 {
		prevOffset := offset - limit
		if prevOffset < 0 {
			prevOffset = 0
		}
		u := buildAbsoluteURL(r, prevOffset, limit)
		prev = &u
	}

	writeJSON(w, http.StatusOK, PaginatedSpecies{
		Count:    totalCount,
		Next:     next,
		Previous: prev,
		Results:  results,
	})
}

// buildAbsoluteURL constructs a pagination URL from the incoming request.
func buildAbsoluteURL(r *http.Request, offset, limit int32) string {
	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return fmt.Sprintf("%s://%s%s?limit=%d&offset=%d", scheme, r.Host, r.URL.Path, limit, offset)
}

// getSpeciesDetail handles GET /api/v1/species/:ebird_code.
func (h *Handler) getSpeciesDetail(w http.ResponseWriter, r *http.Request) {
	ebirdCode := chi.URLParam(r, "ebird_code")

	sp, err := h.queries.GetSpeciesByCode(r.Context(), ebirdCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Printf("GetSpeciesByCode error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	recordings, err := h.queries.GetSpeciesRecordings(r.Context(), ebirdCode)
	if err != nil {
		log.Printf("GetSpeciesRecordings error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	images, err := h.queries.GetSpeciesImages(r.Context(), ebirdCode)
	if err != nil {
		log.Printf("GetSpeciesImages error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	recs := make([]Recording, len(recordings))
	for i, rec := range recordings {
		recs[i] = Recording{
			XenoCantoID: rec.XenoCantoID,
			FilePath:    rec.FilePath,
			Quality:     rec.Quality,
			Type:        rec.Type,
		}
	}

	imgs := make([]Image, len(images))
	for i, img := range images {
		imgs[i] = Image{
			MacaulayID: img.MacaulayID,
			FilePath:   img.FilePath,
			Credit:     img.Credit,
		}
	}

	writeJSON(w, http.StatusOK, SpeciesDetail{
		EbirdCode:      sp.EbirdCode,
		CommonName:     sp.CommonName,
		ScientificName: sp.ScientificName,
		Recordings:     recs,
		Images:         imgs,
	})
}

// getSpeciesGroups handles GET /api/v1/species/:ebird_code/groups.
func (h *Handler) getSpeciesGroups(w http.ResponseWriter, r *http.Request) {
	ebirdCode := chi.URLParam(r, "ebird_code")
	userID := auth.UserIDFromCtx(r.Context())

	groupIDs, err := h.queries.GetGroupsForSpecies(r.Context(), store.GetGroupsForSpeciesParams{
		SpeciesCode: ebirdCode,
		OwnerID:     pgtype.Int8{Int64: userID, Valid: true},
	})
	if err != nil {
		log.Printf("GetGroupsForSpecies error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if groupIDs == nil {
		groupIDs = []int64{}
	}

	writeJSON(w, http.StatusOK, SpeciesGroupsResponse{GroupIDs: groupIDs})
}
