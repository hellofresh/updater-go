package updater

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver"
	"github.com/palantir/stacktrace"
)

type (
	// Release contains information about a release for the current system
	Release struct {
		// Name the name of the release. In most cases this will be the version number
		Name string
		// Assert the name of the asset related to the URL
		Asset string
		// URL the download location of the Asset
		URL string
	}

	// ReleaseLocator describing a release locator that will fetch releases.
	// A release locator should use the ReleaseFilter and AssetFilter during initialization.
	ReleaseLocator interface {
		ListReleases(amount int) ([]Release, error)
	}

	// ReleaseDownloader describes a way to download/load a release
	ReleaseDownloader interface {
		// Fetch downloads the release
		Fetch(r Release) (io.ReadCloser, error)
	}

	// Extractor represent a archive extractor
	Extractor interface {
		// Match checks supported files
		Match(filename string) bool
		// FetchBinary reads an archive and find return the reader for the binary based on the filter
		FetchBinary(input io.Reader, isBinary BinaryFilter) (io.Reader, error)
	}

	// ReleaseFilter is a function that will filter out releases.
	// This is very useful when you want to support stable, beta and dev channels.
	ReleaseFilter func(name string, draft bool, preRelease bool) bool

	// AssetFilter is a function that will filter out unsupported assets for the current system
	AssetFilter func(asset string) bool

	// BinaryFilter is a function used to check if a given path/file is the binary needed
	BinaryFilter func(path os.FileInfo) bool
)

var (
	// ErrNoRelease error is returned in case no available releases were found.
	ErrNoRelease = errors.New("no releases were found")

	// DefaultDownloader the default downloaded to use.
	DefaultDownloader ReleaseDownloader
)

func init() {
	DefaultDownloader = NewHTTPDownloader(http.DefaultClient)
}

// SelfUpdateToLatest update the current executable to it's latest version
func SelfUpdateToLatest(locator ReleaseLocator) (Release, error) {
	latest, err := LatestRelease(locator)
	if err != nil {
		return latest, err
	}

	return latest, SelfUpdate(latest)
}

// SelfUpdate update the current executable to the release
func SelfUpdate(release Release) error {
	// Fetch binary information
	binaryPath, binaryMode, err := executableInfo()
	if err != nil {
		return err
	}

	// Create binary matcher
	isExecutingBinary := func(path os.FileInfo) bool {
		return !path.IsDir() && filepath.Base(path.Name()) == filepath.Base(binaryPath)
	}

	// Download the release
	archive, err := DefaultDownloader.Fetch(release)
	if err != nil {
		return stacktrace.Propagate(err, "failed to fetch the release")
	}
	defer archive.Close()

	// Extract the release
	extractor := MatchingExtractor(release.Asset)
	if extractor == nil {
		return stacktrace.Propagate(os.ErrNotExist, "no extractor is available for the release asset")
	}

	binary, err := extractor.FetchBinary(archive, isExecutingBinary)
	if err != nil {
		return stacktrace.Propagate(err, "unable to locate binary in release asset")
	}

	// Apply update
	if err := Apply(binary, binaryPath, binaryMode); err != nil {
		return stacktrace.Propagate(err, "unable to apply update")
	}

	return nil
}

// LatestRelease retrieve the latest release from the locator using semver
func LatestRelease(locator ReleaseLocator) (Release, error) {
	var latestRelease Release

	releases, err := locator.ListReleases(50)
	if err != nil {
		return latestRelease, stacktrace.Propagate(err, "unable to fetch releases")
	}

	if len(releases) == 0 {
		return latestRelease, ErrNoRelease
	}

	var latestVersion *semver.Version
	for _, release := range releases {
		releaseVersion, err := semver.NewVersion(release.Name)
		if err != nil {
			continue
		}

		if latestVersion == nil || releaseVersion.GreaterThan(latestVersion) {
			latestRelease = release
			latestVersion = releaseVersion
		}
	}

	if latestVersion == nil {
		return latestRelease, stacktrace.Propagate(err, "unable to find the latest release")
	}

	return latestRelease, nil
}

// StableRelease filters out any release that is a draft or pre-release
func StableRelease(_ string, draft bool, preRelease bool) bool {
	return !draft && !preRelease
}

// executableInfo retrieve the current executable and it's file mode
func executableInfo() (string, os.FileMode, error) {
	binaryPath, err := os.Executable()
	if err != nil {
		return "", 0755, stacktrace.Propagate(err, "unable to get executable")
	}

	binaryStats, err := os.Stat(binaryPath)
	if err != nil {
		return binaryPath, 0755, stacktrace.Propagate(err, "unable to stat executable")
	}

	return binaryPath, binaryStats.Mode(), nil
}
