package xenocanto

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Recording struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Quality string `json:"q"`
	Length  string `json:"length"`
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
// filtered to quality A or B, A-first, and capped to maxLenSecs (0 = no cap).
// genus and species come from the eBird scientific name (or a manual override).
// url.PathEscape is required (not url.Values) because xeno-canto needs %20, not +.
func (c *Client) Search(ctx context.Context, genus, species, recType string, maxLenSecs int) ([]Recording, error) {
	queryStr := fmt.Sprintf("type:%s gen:%s sp:%s", recType, genus, species)
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
	return filterAndNormalize(r.Recordings, maxLenSecs), nil
}

func filterAndNormalize(recs []Recording, maxLenSecs int) []Recording {
	var as, bs []Recording
	for _, r := range recs {
		if r.FileURL == "" {
			continue
		}
		if maxLenSecs > 0 && parseLengthSecs(r.Length) > maxLenSecs {
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

// parseLengthSecs parses XC's "m:ss" or "mm:ss" length string into seconds.
// Returns 0 on any parse failure so the recording is not wrongly excluded.
func parseLengthSecs(s string) int {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0
	}
	m, err1 := strconv.Atoi(parts[0])
	sec, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0
	}
	return m*60 + sec
}
