package updater

import (
	"compress/gzip"
	"fmt"
	"io"
	"strings"
)

var tarGz tarGzExtractor

type tarGzExtractor struct {
	tarExtractor
}

func init() {
	RegisterFormat("TarGz", tarGz)
}

func (tarGzExtractor) Match(filename string) bool {
	filename = strings.ToLower(filename)
	return strings.HasSuffix(filename, ".tar.gz") ||
		strings.HasSuffix(filename, ".tgz")
}

// Read a .tar.gz from a Reader and locates the needed binary
func (t tarGzExtractor) FetchBinary(input io.Reader, isBinary BinaryFilter) (io.Reader, error) {
	gzr, err := gzip.NewReader(input)
	if err != nil {
		return nil, fmt.Errorf("error decompressing: %w", err)
	}
	defer gzr.Close()

	return t.tarExtractor.FetchBinary(gzr, isBinary)
}
