package ebird

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type TaxonomyEntry struct {
	SpeciesCode string `json:"speciesCode"`
	CommonName  string `json:"comName"`
	SciName     string `json:"sciName"`
	Category    string `json:"category"`
}

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func New(apiKey string) *Client {
	return newWithBaseURL(apiKey, "https://api.ebird.org")
}

func newWithBaseURL(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (c *Client) Taxonomy(ctx context.Context) ([]TaxonomyEntry, error) {
	url := c.baseURL + "/v2/ref/taxonomy/ebird?fmt=json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-eBirdApiToken", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ebird taxonomy: status %d", resp.StatusCode)
	}
	var entries []TaxonomyEntry
	return entries, json.NewDecoder(resp.Body).Decode(&entries)
}

func (c *Client) SpeciesList(ctx context.Context, regionCode string) ([]string, error) {
	endpoint := fmt.Sprintf("%s/v2/product/spplist/%s", c.baseURL, url.PathEscape(regionCode))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-eBirdApiToken", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ebird spplist %s: status %d", regionCode, resp.StatusCode)
	}
	var codes []string
	return codes, json.NewDecoder(resp.Body).Decode(&codes)
}
