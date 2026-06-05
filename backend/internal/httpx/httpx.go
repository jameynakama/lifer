// Package httpx holds the shared HTTP plumbing for the external API clients
// (eBird, xeno-canto, Macaulay): one GET-decode path and one default client
// with a timeout, so a hung upstream can never block a worker forever.
package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DefaultClient is shared by the API clients. The timeout bounds the whole
// request (dial, headers, body).
var DefaultClient = &http.Client{Timeout: 30 * time.Second}

// GetJSON GETs url with optional headers and decodes the JSON response into
// dest. Non-200 responses become errors carrying the status code.
func GetJSON(ctx context.Context, client *http.Client, url string, header http.Header, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get %s: status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}
