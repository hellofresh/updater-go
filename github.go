package updater

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type (
	// GithubLocator struct encapsulates information about github repo
	GithubLocator struct {
		client *githubv4.Client

		owner string
		repo  string

		acceptRelease ReleaseFilter
		acceptAsset   AssetFilter
	}

	ghReleaseAsset struct {
		Name githubv4.String
		URL  githubv4.String
	}

	ghRelease struct {
		Name         githubv4.String
		IsDraft      githubv4.Boolean
		IsPrerelease githubv4.Boolean

		ReleaseAssets struct {
			Nodes []ghReleaseAsset
		} `graphql:"releaseAssets(first: 20)"`
	}

	queryRepoReleases struct {
		Repository *struct {
			Releases struct {
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage githubv4.Boolean
				}
				Nodes []ghRelease
			} `graphql:"releases(first: 50, after: $releaseCursor, orderBy: {field:CREATED_AT, direction: DESC})"`
		} `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
	}
)

var (
	// ErrNoRepository error is returned is the repository is not found or user token has no access to it
	ErrNoRepository = errors.New("no repository")
)

// NewGithubClient creates new github locator instance
func NewGithubClient(
	owner string,
	repository string,
	token string,
	releaseFilter ReleaseFilter,
	assetFilter AssetFilter,
	connectionTimeout time.Duration,
) *GithubLocator {
	var httpClient *http.Client
	if token != "" {
		src := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		httpClient = oauth2.NewClient(context.Background(), src)
	}

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	httpClient.Timeout = connectionTimeout

	client := githubv4.NewClient(httpClient)
	return &GithubLocator{
		client:        client,
		owner:         owner,
		repo:          repository,
		acceptRelease: releaseFilter,
		acceptAsset:   assetFilter,
	}
}

// ListReleases returns available GH releases list
func (g *GithubLocator) ListReleases(amount int) ([]Release, error) {
	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(g.owner),
		"repositoryName":  githubv4.String(g.repo),
		"releaseCursor":   (*githubv4.String)(nil), // Null after argument to get first page.
	}

	var query queryRepoReleases
	var releases []Release
	for {
		if err := g.client.Query(context.TODO(), &query, variables); err != nil {
			return nil, err
		}

		if query.Repository == nil {
			return nil, ErrNoRepository
		}

		for _, release := range query.Repository.Releases.Nodes {
			if !g.acceptRelease(string(release.Name), bool(release.IsDraft), bool(release.IsPrerelease)) {
				continue
			}

			for _, asset := range release.ReleaseAssets.Nodes {
				if !g.acceptAsset(string(asset.Name)) {
					continue
				}

				releases = append(
					releases,
					Release{
						Name:  string(release.Name),
						Asset: string(asset.Name),
						URL:   string(asset.URL),
					},
				)
				break
			}
		}

		if query.Repository.Releases.PageInfo.HasNextPage {
			variables["releaseCursor"] = githubv4.NewString(query.Repository.Releases.PageInfo.EndCursor)
			continue
		}

		break
	}

	return releases, nil
}
