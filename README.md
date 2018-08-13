# updater-go

Library that heps verifying/updating go binary with new version

## Usage example

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/hellofresh/updater-go"
	"github.com/palantir/stacktrace"
	log "github.com/sirupsen/logrus"
)

const (
	githubOwner = "hellofresh"
	githubRepo  = "jetstream"
)

func main() {
	var (
		updateToVersion string
		ghToken string
	)
	flag.StringVar(&updateToVersion, "version", "", "update to a particular version instead of the latest stable")
	flag.StringVar(&ghToken, "token", "", "GitHub token to use for Github access")

	flag.Parse()

	// Check to which version we need to update
	versionFilter := updater.StableRelease
	if updateToVersion != "" {
		versionFilter = func(name string, _ bool, _ bool) bool {
		return updateToVersion == name
		}
	}

	// Create release locator
	locator := updater.NewGithubClient(
		githubOwner,
		githubRepo,
		ghToken,
		versionFilter,
		func(asset string) bool {
			return strings.Contains(asset, fmt.Sprintf("-%s-%s-", runtime.GOARCH, runtime.GOOS))
		},
	)

	// Find the release
	updateTo, err := locateRelease(locator, updateToVersion)
	if rootErr := stacktrace.RootCause(err); rootErr == updater.ErrNoRepository {
		log.Error("Unable to access the Jetstream repository.\n  This is probably due to insufficient privileges of the access token.")
		os.Exit(1)
	}
	failOnError(err, "failed to retrieve the update release")

	// Fetch the release and update
	err = updater.SelfUpdate(updateTo)
	failOnError(err, "failed to update to version %s")

	fmt.Printf("Successfully updated to version %s!\n", updateTo.Name)
}

func failOnError(err error, message string) {
	if err != nil {
		log.WithError(err).Error(message)
		os.Exit(1)
	}
}

func locateRelease(locator updater.ReleaseLocator, version string) (updater.Release, error) {
	// No specific version use the latest
	if version == "" {
		return updater.LatestRelease(locator)
	}

	// Find a specific release
	var release updater.Release
	updates, err := locator.ListReleases(1)
	if err != nil {
		return release, err
	}

	if len(updates) == 0 {
		return release, fmt.Errorf("unable to locate release %s", version)
	}

	if len(updates) > 1 {
		return release, fmt.Errorf("multiple releases locate for %s", version)
	}

	return updates[0], nil
}
```
