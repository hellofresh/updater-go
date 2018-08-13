package updater

import (
	"io"
	"net/http"

	"github.com/palantir/stacktrace"
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
		return nil, stacktrace.Propagate(err, "could not create a request for the release download URL")
	}

	req.Header.Add("Accept", "application/octet-stream")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to download release")
	}

	return resp.Body, nil
}
