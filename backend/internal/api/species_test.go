package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jameynakama/lifer/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// speciesStubQuerier stubs species list/search methods.
type speciesStubQuerier struct {
	store.Querier
	searchSpecies func(ctx context.Context, query pgtype.Text) ([]store.SearchSpeciesRow, error)
	listSpecies   func(ctx context.Context, arg store.ListSpeciesParams) ([]store.ListSpeciesRow, error)
}

func (s *speciesStubQuerier) SearchSpecies(ctx context.Context, query pgtype.Text) ([]store.SearchSpeciesRow, error) {
	return s.searchSpecies(ctx, query)
}

func (s *speciesStubQuerier) ListSpecies(ctx context.Context, arg store.ListSpeciesParams) ([]store.ListSpeciesRow, error) {
	return s.listSpecies(ctx, arg)
}

func TestListSpecies_NoPagination_ReturnsFirstPage(t *testing.T) {
	q := &speciesStubQuerier{
		listSpecies: func(_ context.Context, arg store.ListSpeciesParams) ([]store.ListSpeciesRow, error) {
			assert.Equal(t, int32(20), arg.Limit)
			assert.Equal(t, int32(0), arg.Offset)
			return []store.ListSpeciesRow{
				{EbirdCode: "amro", CommonName: "American Robin", ScientificName: "Turdus migratorius", TotalCount: 1},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body PaginatedSpecies
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, int64(1), body.Count)
	assert.Nil(t, body.Next)
	assert.Nil(t, body.Previous)
	assert.Len(t, body.Results, 1)
	assert.Equal(t, "American Robin", body.Results[0].CommonName)
}

func TestListSpecies_WithOffset_SetsPreviousAndNext(t *testing.T) {
	q := &speciesStubQuerier{
		listSpecies: func(_ context.Context, arg store.ListSpeciesParams) ([]store.ListSpeciesRow, error) {
			assert.Equal(t, int32(20), arg.Limit)
			assert.Equal(t, int32(20), arg.Offset)
			return []store.ListSpeciesRow{
				{EbirdCode: "amro", CommonName: "American Robin", ScientificName: "Turdus migratorius", TotalCount: 50},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species?offset=20", nil)
	r.Host = "localhost:8080"
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body PaginatedSpecies
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, int64(50), body.Count)
	require.NotNil(t, body.Next)
	assert.Contains(t, *body.Next, "offset=40")
	require.NotNil(t, body.Previous)
	assert.Contains(t, *body.Previous, "offset=0")
}

func TestListSpecies_SearchMode_NoPaginationLinks(t *testing.T) {
	q := &speciesStubQuerier{
		searchSpecies: func(_ context.Context, query pgtype.Text) ([]store.SearchSpeciesRow, error) {
			assert.Equal(t, "sparrow", query.String)
			return []store.SearchSpeciesRow{
				{EbirdCode: "sonspa", CommonName: "Song Sparrow", ScientificName: "Melospiza melodia"},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species?q=sparrow", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body PaginatedSpecies
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, int64(1), body.Count)
	assert.Nil(t, body.Next)
	assert.Nil(t, body.Previous)
	assert.Len(t, body.Results, 1)
	assert.Equal(t, "Song Sparrow", body.Results[0].CommonName)
}

func TestListSpecies_EmptyResults_ReturnsEmptySlice(t *testing.T) {
	q := &speciesStubQuerier{
		listSpecies: func(_ context.Context, arg store.ListSpeciesParams) ([]store.ListSpeciesRow, error) {
			return []store.ListSpeciesRow{}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body PaginatedSpecies
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, int64(0), body.Count)
	assert.NotNil(t, body.Results) // must be [] not null
}

// speciesDetailStubQuerier stubs the three queries used by getSpeciesDetail.
type speciesDetailStubQuerier struct {
	store.Querier
	getSpeciesByCode     func(ctx context.Context, ebirdCode string) (store.GetSpeciesByCodeRow, error)
	getSpeciesRecordings func(ctx context.Context, speciesCode string) ([]store.GetSpeciesRecordingsRow, error)
	getSpeciesImages     func(ctx context.Context, speciesCode string) ([]store.GetSpeciesImagesRow, error)
}

func (s *speciesDetailStubQuerier) GetSpeciesByCode(ctx context.Context, ebirdCode string) (store.GetSpeciesByCodeRow, error) {
	return s.getSpeciesByCode(ctx, ebirdCode)
}
func (s *speciesDetailStubQuerier) GetSpeciesRecordings(ctx context.Context, speciesCode string) ([]store.GetSpeciesRecordingsRow, error) {
	return s.getSpeciesRecordings(ctx, speciesCode)
}
func (s *speciesDetailStubQuerier) GetSpeciesImages(ctx context.Context, speciesCode string) ([]store.GetSpeciesImagesRow, error) {
	return s.getSpeciesImages(ctx, speciesCode)
}

func TestGetSpeciesDetail_ReturnsFullDetail(t *testing.T) {
	q := &speciesDetailStubQuerier{
		getSpeciesByCode: func(_ context.Context, ebirdCode string) (store.GetSpeciesByCodeRow, error) {
			assert.Equal(t, "amro", ebirdCode)
			return store.GetSpeciesByCodeRow{
				EbirdCode:      "amro",
				CommonName:     "American Robin",
				ScientificName: "Turdus migratorius",
			}, nil
		},
		getSpeciesRecordings: func(_ context.Context, speciesCode string) ([]store.GetSpeciesRecordingsRow, error) {
			return []store.GetSpeciesRecordingsRow{
				{XenoCantoID: "xc123", FilePath: "https://r2.example.com/xc123.mp3", Quality: "A", Type: "song"},
			}, nil
		},
		getSpeciesImages: func(_ context.Context, speciesCode string) ([]store.GetSpeciesImagesRow, error) {
			return []store.GetSpeciesImagesRow{
				{MacaulayID: "ml456", FilePath: "https://r2.example.com/ml456.jpg", Credit: "J. Doe"},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species/amro", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "ebird_code", "amro")
	w := httptest.NewRecorder()

	h.getSpeciesDetail(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body SpeciesDetail
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "amro", body.EbirdCode)
	assert.Equal(t, "American Robin", body.CommonName)
	assert.Len(t, body.Recordings, 1)
	assert.Equal(t, "xc123", body.Recordings[0].XenoCantoID)
	assert.Equal(t, "song", body.Recordings[0].Type)
	assert.Len(t, body.Images, 1)
	assert.Equal(t, "ml456", body.Images[0].MacaulayID)
}

func TestGetSpeciesDetail_NotFound_Returns404(t *testing.T) {
	q := &speciesDetailStubQuerier{
		getSpeciesByCode: func(_ context.Context, ebirdCode string) (store.GetSpeciesByCodeRow, error) {
			return store.GetSpeciesByCodeRow{}, pgx.ErrNoRows
		},
		getSpeciesRecordings: func(_ context.Context, _ string) ([]store.GetSpeciesRecordingsRow, error) {
			return nil, nil
		},
		getSpeciesImages: func(_ context.Context, _ string) ([]store.GetSpeciesImagesRow, error) {
			return nil, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species/nope", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "ebird_code", "nope")
	w := httptest.NewRecorder()

	h.getSpeciesDetail(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// speciesGroupsStubQuerier stubs GetGroupsForSpecies.
type speciesGroupsStubQuerier struct {
	store.Querier
	getGroupsForSpecies func(ctx context.Context, arg store.GetGroupsForSpeciesParams) ([]int64, error)
}

func (s *speciesGroupsStubQuerier) GetGroupsForSpecies(ctx context.Context, arg store.GetGroupsForSpeciesParams) ([]int64, error) {
	return s.getGroupsForSpecies(ctx, arg)
}

func TestGetSpeciesGroups_ReturnsMembership(t *testing.T) {
	q := &speciesGroupsStubQuerier{
		getGroupsForSpecies: func(_ context.Context, arg store.GetGroupsForSpeciesParams) ([]int64, error) {
			assert.Equal(t, "amro", arg.SpeciesCode)
			assert.Equal(t, int64(1), arg.OwnerID.Int64)
			assert.True(t, arg.OwnerID.Valid)
			return []int64{2, 5}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species/amro/groups", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "ebird_code", "amro")
	w := httptest.NewRecorder()

	h.getSpeciesGroups(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body SpeciesGroupsResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.ElementsMatch(t, []int64{2, 5}, body.GroupIDs)
}

func TestGetSpeciesGroups_NoMembership_ReturnsEmptySlice(t *testing.T) {
	q := &speciesGroupsStubQuerier{
		getGroupsForSpecies: func(_ context.Context, arg store.GetGroupsForSpeciesParams) ([]int64, error) {
			return []int64{}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species/amro/groups", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "ebird_code", "amro")
	w := httptest.NewRecorder()

	h.getSpeciesGroups(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body SpeciesGroupsResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.NotNil(t, body.GroupIDs)
	assert.Empty(t, body.GroupIDs)
}

// allSpeciesStubQuerier stubs ListAllSpecies.
type allSpeciesStubQuerier struct {
	store.Querier
	listAllSpecies func(ctx context.Context) ([]store.ListAllSpeciesRow, error)
}

func (s *allSpeciesStubQuerier) ListAllSpecies(ctx context.Context) ([]store.ListAllSpeciesRow, error) {
	return s.listAllSpecies(ctx)
}

func TestListAllSpecies_ReturnsFullCatalog(t *testing.T) {
	q := &allSpeciesStubQuerier{
		listAllSpecies: func(_ context.Context) ([]store.ListAllSpeciesRow, error) {
			return []store.ListAllSpeciesRow{
				{EbirdCode: "amro", CommonName: "American Robin", ScientificName: "Turdus migratorius", ImageUrl: "https://r2.example.com/amro.jpg"},
				{EbirdCode: "bcch", CommonName: "Black-capped Chickadee", ScientificName: "Poecile atricapillus", ImageUrl: ""},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species/all", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listAllSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []SpeciesItem
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Len(t, body, 2)
	assert.Equal(t, "American Robin", body[0].CommonName)
	require.NotNil(t, body[0].ImageURL)
	assert.Equal(t, "https://r2.example.com/amro.jpg", *body[0].ImageURL)
	assert.Equal(t, "Black-capped Chickadee", body[1].CommonName)
	assert.Nil(t, body[1].ImageURL)
}

func TestListAllSpecies_EmptyDB_ReturnsEmptyArray(t *testing.T) {
	q := &allSpeciesStubQuerier{
		listAllSpecies: func(_ context.Context) ([]store.ListAllSpeciesRow, error) {
			return []store.ListAllSpeciesRow{}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species/all", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listAllSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []SpeciesItem
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.NotNil(t, body)
	assert.Empty(t, body)
}

func TestListAllSpecies_DBError_Returns500(t *testing.T) {
	q := &allSpeciesStubQuerier{
		listAllSpecies: func(_ context.Context) ([]store.ListAllSpeciesRow, error) {
			return nil, errors.New("db down")
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species/all", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listAllSpecies(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListAllSpecies_Unauthenticated_Returns401(t *testing.T) {
	h := makeHandler(&allSpeciesStubQuerier{
		listAllSpecies: func(_ context.Context) ([]store.ListAllSpeciesRow, error) {
			return nil, nil
		},
	})
	// No injectUserID -- simulates a request without a valid JWT
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species/all", nil)
	w := httptest.NewRecorder()

	// Route through the full router so RequireAuth middleware runs
	router := NewRouter(RouterConfig{
		Queries:     h.queries,
		OAuthConfig: nil,
		JWTSecret:   []byte("test-secret"),
		FrontendURL: "http://localhost:5173",
	})
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
