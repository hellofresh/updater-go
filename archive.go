package updater

import "log"

// supportedExtractors contains all supported archive formats
var supportedExtractors = map[string]Extractor{}

// RegisterFormat adds a supported archive format
func RegisterFormat(name string, extractor Extractor) {
	if _, ok := supportedExtractors[name]; ok {
		log.Printf("Format %s already exists, skip!\n", name)
		return
	}
	supportedExtractors[name] = extractor
}

// MatchingExtractor returns the first extractor that matches
// the given file, or nil if there is no match
func MatchingExtractor(path string) Extractor {
	for _, fmt := range supportedExtractors {
		if fmt.Match(path) {
			return fmt
		}
	}
	return nil
}
