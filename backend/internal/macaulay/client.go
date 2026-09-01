package macaulay

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jameynakama/flockdeck/internal/httpx"
	"net/url"
	"strconv"
)

type Photo struct {
	AssetID         string
	UserDisplayName string
}

// apiPhoto is one element of the v2 search response, which is a bare JSON
// array. assetId arrives as a number; every consumer wants the string form
// (macaulay_id, R2 object keys, the banned-images list).
type apiPhoto struct {
	AssetID         int64  `json:"assetId"`
	UserDisplayName string `json:"userDisplayName"`
}

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func New(apiKey string) *Client {
	return NewWithBaseURL(apiKey, "https://search.macaulaylibrary.org")
}

// NewWithBaseURL constructs a client against an alternate host (httptest servers in tests).
func NewWithBaseURL(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: httpx.DefaultClient,
	}
}

// Photos returns up to max photos for the given eBird species code.
func (c *Client) Photos(ctx context.Context, speciesCode string, max int) ([]Photo, error) {
	params := url.Values{}
	params.Set("taxonCode", speciesCode)
	params.Set("mediaType", "photo")
	params.Set("sort", "rating_rank_desc")
	params.Set("count", strconv.Itoa(max))
	endpoint := fmt.Sprintf("%s/api/v2/search?%s", c.baseURL, params.Encode())
	var results []apiPhoto
	if err := httpx.GetJSON(ctx, c.httpClient, endpoint, http.Header{"X-eBirdApiToken": {c.apiKey}}, &results); err != nil {
		return nil, err
	}
	if len(results) > max {
		results = results[:max]
	}
	photos := make([]Photo, 0, len(results))
	for _, r := range results {
		photos = append(photos, Photo{
			AssetID:         strconv.FormatInt(r.AssetID, 10),
			UserDisplayName: r.UserDisplayName,
		})
	}
	return photos, nil
}

func (c *Client) PhotoURL(assetID string) string {
	return fmt.Sprintf("https://cdn.download.ams.birds.cornell.edu/api/v1/asset/%s/large", assetID)
}
