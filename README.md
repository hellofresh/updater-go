# updater-go

Library that helps verifying/updating go binary with new version

## Usage example

```go
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/hellofresh/updater-go/v3"
	log "github.com/sirupsen/logrus"
)

const (
	githubOwner = "hellofresh"
	githubRepo  = "github-cli"
)

func main() {
	var (
		updateToVersion string
		ghToken         string
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
		context.TODO(),
		githubOwner,
		githubRepo,
		ghToken,
		versionFilter,
		func(asset string) bool {
			return strings.Contains(asset, fmt.Sprintf("-%s-%s-", runtime.GOARCH, runtime.GOOS))
		},
		10 * time.Second,
	)

	// Find the release
	updateTo, err := locateRelease(locator, updateToVersion)
	if errors.Is(err, updater.ErrNoRepository) || errors.Is(err, updater.ErrUnauthorized) {
		log.WithError(err).Error("Unable to access the repository.\n  This is probably due to insufficient privileges of the access token.")
		os.Exit(1)
	}
	failOnError(err, "failed to retrieve the update release")

	// Use context with deadlines to specify different timeouts (optional)
	ctx, cancel := context.WithTimeout(context.TODO(), 30 * time.Second)
	defer cancel()

	// Fetch the release and update
	err = updater.SelfUpdate(ctx, updateTo)
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
		return updater.LatestRelease(context.TODO(), locator)
	}

	// Find a specific release
	var release updater.Release
	updates, err := locator.ListReleases(context.TODO(), 1)
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

---
> GitHub [@hellofresh](https://github.com/hellofresh) &nbsp;&middot;&nbsp;
> Medium [@engineering.hellofresh](https://engineering.hellofresh.com)
