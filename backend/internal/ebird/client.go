package ebird

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jameynakama/flockdeck/internal/httpx"
)

type TaxonomyEntry struct {
	SpeciesCode   string `json:"speciesCode"`
	CommonName    string `json:"comName"`
	SciName       string `json:"sciName"`
	Category      string `json:"category"`
	FamilyComName string `json:"familyComName"`
}

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func New(apiKey string) *Client {
	return NewWithBaseURL(apiKey, "https://api.ebird.org")
}

// NewWithBaseURL constructs a client against an alternate host (httptest servers in tests).
func NewWithBaseURL(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: httpx.DefaultClient,
	}
}

func (c *Client) header() http.Header {
	return http.Header{"X-eBirdApiToken": {c.apiKey}}
}

func (c *Client) Taxonomy(ctx context.Context) ([]TaxonomyEntry, error) {
	var entries []TaxonomyEntry
	err := httpx.GetJSON(ctx, c.httpClient, c.baseURL+"/v2/ref/taxonomy/ebird?fmt=json&cat=species", c.header(), &entries)
	return entries, err
}

func (c *Client) SpeciesList(ctx context.Context, regionCode string) ([]string, error) {
	endpoint := fmt.Sprintf("%s/v2/product/spplist/%s", c.baseURL, url.PathEscape(regionCode))
	var codes []string
	err := httpx.GetJSON(ctx, c.httpClient, endpoint, c.header(), &codes)
	return codes, err
}
