package updater

import (
	"fmt"
	"io"
	"net/http"
)

// HTTPDownloader represents http downloader client
type HTTPDownloader struct {
	client *http.Client
}

// NewHTTPDownloader creates new http downloader client instance.
// If the passed client is nil http.DefaultClient is used.
func NewHTTPDownloader(client *http.Client) *HTTPDownloader {
	if client == nil {
		client = http.DefaultClient
	}

	return &HTTPDownloader{client: client}
}

// Fetch downloads GH release
func (d *HTTPDownloader) Fetch(r Release) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, r.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create a request for the release download URL: %w", err)
	}

	req.Header.Add("Accept", "application/octet-stream")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to download release: %w", err)
	}

	return resp.Body, nil
}
