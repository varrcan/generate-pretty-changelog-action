package git

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/varrcan/generate-pretty-changelog/pkg/context"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// Repo represents a repository
type Repo struct {
	Owner  string
	Name   string
	RawURL string
}

func (r Repo) String() string {
	if r.Owner == "" && r.Name == "" {
		return ""
	}
	return r.Owner + "/" + r.Name
}

// Client interface
type Client interface {
	Changelog(ctx *context.Context, repo Repo, prev, current string) (string, error)
}

// NewClient creates a new client depending on the token type
func NewClient(ctx *context.Context) (Client, error) {
	return newGitHub(ctx, ctx.Token)
}

type githubClient struct {
	client *github.Client
}

// newGitHub returns a github client implementation.
func newGitHub(ctx *context.Context, token string) (*githubClient, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	httpClient := oauth2.NewClient(ctx, ts)
	base := httpClient.Transport.(*oauth2.Transport).Base
	if base == nil || reflect.ValueOf(base).IsNil() {
		base = http.DefaultTransport
	}

	base.(*http.Transport).TLSClientConfig = &tls.Config{}
	base.(*http.Transport).Proxy = http.ProxyFromEnvironment
	httpClient.Transport.(*oauth2.Transport).Base = base

	client := github.NewClient(httpClient)

	return &githubClient{client: client}, nil
}

func (c *githubClient) checkRateLimit(ctx *context.Context) {
	limits, _, err := c.client.RateLimit.Get(ctx)
	if err != nil {
		fmt.Println("could not check rate limits, hoping for the best...")
		return
	}
	if limits.Core.Remaining > 100 { // 100 should be safe enough
		return
	}
	sleep := limits.Core.Reset.UTC().Sub(time.Now().UTC())
	if sleep <= 0 {
		sleep = 15 * time.Second
	}
	fmt.Printf("token too close to rate limiting, will sleep for %s before continuing...", sleep)
	time.Sleep(sleep)
	c.checkRateLimit(ctx)
}

// Changelog returns a changelog for the given repository
func (c *githubClient) Changelog(ctx *context.Context, repo Repo, prev, current string) (string, error) {
	c.checkRateLimit(ctx)
	var log []string
	opts := &github.ListOptions{PerPage: 100}

	for {
		result, resp, err := c.client.Repositories.CompareCommits(ctx, repo.Owner, repo.Name, prev, current, opts)
		if err != nil {
			return "", err
		}
		for _, commit := range result.Commits {
			log = append(log, fmt.Sprintf(
				"%s: %s (@%s)",
				commit.GetSHA(),
				strings.Split(commit.Commit.GetMessage(), "\n")[0],
				commit.GetAuthor().GetLogin(),
			))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return strings.Join(log, "\n"), nil
}
