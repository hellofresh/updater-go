package updater

import (
	"archive/tar"
	"io"
	"os"
	"strings"
)

type tarExtractor struct{}

func init() {
	RegisterFormat("Tar", tarExtractor{})
}

func (tarExtractor) Match(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".tar")
}

// Read a .tar from a Reader and locates the needed binary
func (tarExtractor) FetchBinary(input io.Reader, isBinary BinaryFilter) (io.Reader, error) {
	tr := tar.NewReader(input)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if isBinary(header.FileInfo()) {
			return tr, nil
		}
	}

	return nil, os.ErrNotExist
}
