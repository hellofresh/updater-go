package updater

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
)

const instrumentationName = "github.com/hellofresh/updater-go"

type (
	// GithubLocator struct encapsulates information about github repo
	GithubLocator struct {
		client         *githubv4.Client
		tracer         trace.Tracer
		defaultTimeout time.Duration

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
			} `graphql:"releases(first: $limit, after: $releaseCursor, orderBy: {field:CREATED_AT, direction: DESC})"`
		} `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
	}
)

var (
	// ErrNoRepository error is returned if the repository is not found or user token has no access to it
	ErrNoRepository = errors.New("no repository")
	// ErrUnauthorized error is returned if user token does not have access to the repository or the token is invalid
	ErrUnauthorized = errors.New("no access to the repository, probably bad credentials")
)

// NewGithubClient creates new github locator instance
func NewGithubClient(
	ctx context.Context,
	owner string,
	repository string,
	token string,
	releaseFilter ReleaseFilter,
	assetFilter AssetFilter,
	connectionTimeout time.Duration,
	opts ...GitHubLocatorOption,
) *GithubLocator {
	l := &GithubLocator{
		owner:          owner,
		repo:           repository,
		acceptRelease:  releaseFilter,
		acceptAsset:    assetFilter,
		defaultTimeout: connectionTimeout,
		tracer:         trace.NewNoopTracerProvider().Tracer(instrumentationName),
	}

	for _, opt := range opts {
		opt.applyGitHubLocatorOption(l)
	}

	var httpClient *http.Client

	if token != "" {
		src := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		httpClient = oauth2.NewClient(ctx, src)
	}

	httpClient.Transport = otelhttp.NewTransport(httpClient.Transport, formatHTTPSpanName())
	l.client = githubv4.NewClient(httpClient)

	return l
}

// ListReleases returns available GH releases list.
func (g *GithubLocator) ListReleases(ctx context.Context, amount int) (_ []Release, err error) {
	ctx, span := g.tracer.Start(ctx, "github:list-releases", trace.WithAttributes(
		attribute.String("github.owner", g.owner),
		attribute.String("github.repository", g.repo),
		attribute.Int("github.limit", amount),
	))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()
	}()

	ctx, cancel := ensureTimeout(ctx, g.defaultTimeout)
	defer cancel()

	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(g.owner),
		"repositoryName":  githubv4.String(g.repo),
		"releaseCursor":   (*githubv4.String)(nil), // Null after argument to get first page.
		"limit":           githubv4.Int(amount),
	}

	var query queryRepoReleases
	var releases []Release
	for {
		if err := g.client.Query(ctx, &query, variables); err != nil {
			if strings.Contains(err.Error(), "non-200 OK status code: 401 Unauthorized") {
				return nil, ErrUnauthorized
			}

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

// GitHubLocatorOption is an option to configure GithubLocator.
type GitHubLocatorOption interface {
	applyGitHubLocatorOption(l *GithubLocator)
}

type gitHubLocatorOptionFunc func(l *GithubLocator)

func (f gitHubLocatorOptionFunc) applyGitHubLocatorOption(l *GithubLocator) {
	f(l)
}

// WithTracerProvider sets the TracerProvider.
func WithTracerProvider(tp trace.TracerProvider) GitHubLocatorOption {
	return gitHubLocatorOptionFunc(func(o *GithubLocator) {
		o.tracer = tp.Tracer(instrumentationName)
	})
}

func ensureTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); !ok {
		return context.WithTimeout(ctx, timeout)
	}

	return ctx, func() {}
}

func formatHTTPSpanName() otelhttp.Option {
	return otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
		return fmt.Sprintf("http:%s:%s", r.Method, r.URL.Path)
	})
}
