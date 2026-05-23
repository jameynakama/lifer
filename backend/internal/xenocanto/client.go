package xenocanto

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Recording struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Quality string `json:"q"`
	FileURL string `json:"file"`
}

type apiResponse struct {
	Recordings []Recording `json:"recordings"`
}

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func New(apiKey string) *Client {
	return newWithBaseURL(apiKey, "https://xeno-canto.org")
}

func newWithBaseURL(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// Search returns recordings for a species of the given type ("song" or "call"),
// filtered to quality A or B, A-first. Uses the English common name with xeno-canto's
// en:"..." syntax so taxonomy mismatches between eBird and xeno-canto don't cause
// empty results. url.PathEscape is required (not url.Values) because xeno-canto needs
// %20 for spaces inside quotes, not +.
func (c *Client) Search(ctx context.Context, commonName, recType string) ([]Recording, error) {
	queryStr := fmt.Sprintf("type:%s en:\"%s\"", recType, strings.ToLower(commonName))
	endpoint := fmt.Sprintf("%s/api/3/recordings?key=%s&query=%s",
		c.baseURL,
		url.QueryEscape(c.apiKey),
		url.PathEscape(queryStr),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("xeno-canto search: status %d", resp.StatusCode)
	}
	var r apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	return filterAndNormalize(r.Recordings), nil
}

func filterAndNormalize(recs []Recording) []Recording {
	var as, bs []Recording
	for _, r := range recs {
		if r.FileURL == "" {
			continue
		}
		// xeno-canto sometimes returns protocol-relative URLs
		if strings.HasPrefix(r.FileURL, "//") {
			r.FileURL = "https:" + r.FileURL
		}
		switch r.Quality {
		case "A":
			as = append(as, r)
		case "B":
			bs = append(bs, r)
		}
	}
	return append(as, bs...)
}
