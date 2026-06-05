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
	AssetID         string `json:"assetId"`
	UserDisplayName string `json:"userDisplayName"`
}

type apiResults struct {
	Content []Photo `json:"content"`
}

type apiResponse struct {
	Results apiResults `json:"results"`
}

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func New(apiKey string) *Client {
	return newWithBaseURL(apiKey, "https://search.macaulaylibrary.org")
}

func newWithBaseURL(apiKey, baseURL string) *Client {
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
	endpoint := fmt.Sprintf("%s/api/v1/search?%s", c.baseURL, params.Encode())
	var r apiResponse
	if err := httpx.GetJSON(ctx, c.httpClient, endpoint, http.Header{"X-eBirdApiToken": {c.apiKey}}, &r); err != nil {
		return nil, err
	}
	photos := r.Results.Content
	if len(photos) > max {
		photos = photos[:max]
	}
	return photos, nil
}

func (c *Client) PhotoURL(assetID string) string {
	return fmt.Sprintf("https://cdn.download.ams.birds.cornell.edu/api/v1/asset/%s/large", assetID)
}
