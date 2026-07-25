// Package agent talks to the GitHub API using GitHub App authentication and
// collects the state of an organization's self-hosted Actions runners.
package agent

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v88/github"
)

// Runner is the exporter's domain representation of a self-hosted runner,
// decoupled from the go-github types.
type Runner struct {
	ID     int64
	Name   string
	OS     string
	Status string // "online" / "offline"
	Busy   bool
	Labels []string
}

// Online reports whether the runner is connected to GitHub.
func (r Runner) Online() bool { return r.Status == "online" }

// runnerService is the subset of the go-github Actions client used here. It is
// an interface so tests can supply a fake without hitting the network.
type runnerService interface {
	ListOrganizationRunners(ctx context.Context, org string, opts *github.ListRunnersOptions) (*github.Runners, *github.Response, error)
}

// Client fetches runner state for a single organization.
type Client struct {
	org     string
	runners runnerService
}

// New builds a Client authenticated as a GitHub App installation.
//
// appID / installationID identify the App and its installation; privateKeyPath
// points to the PEM private key. apiBaseURL may be empty for public github.com
// or set to a GitHub Enterprise Server API URL.
func New(appID, installationID int64, privateKeyPath, org, apiBaseURL string) (*Client, error) {
	tr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, appID, installationID, privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("build GitHub App transport: %w", err)
	}

	httpClient := &http.Client{Transport: tr}

	opts := []github.ClientOptionsFunc{github.WithHTTPClient(httpClient)}
	if apiBaseURL != "" {
		opts = append(opts, github.WithEnterpriseURLs(apiBaseURL, apiBaseURL))
		// Point the App transport at the same API host so JWT/installation
		// token requests go to the Enterprise Server too.
		tr.BaseURL = apiBaseURL
	}

	gh, err := github.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("build GitHub client: %w", err)
	}

	return &Client{org: org, runners: gh.Actions}, nil
}

// ListRunners returns all self-hosted runners registered at the organization
// level, following pagination.
func (c *Client) ListRunners(ctx context.Context) ([]Runner, error) {
	opts := &github.ListRunnersOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var out []Runner
	for {
		page, resp, err := c.runners.ListOrganizationRunners(ctx, c.org, opts)
		if err != nil {
			return nil, fmt.Errorf("list org runners for %q: %w", c.org, err)
		}
		for _, r := range page.Runners {
			out = append(out, toRunner(r))
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return out, nil
}

func toRunner(r *github.Runner) Runner {
	if r == nil {
		return Runner{}
	}
	labels := make([]string, 0, len(r.Labels))
	for _, l := range r.Labels {
		labels = append(labels, l.GetName())
	}
	return Runner{
		ID:     r.GetID(),
		Name:   r.GetName(),
		OS:     r.GetOS(),
		Status: r.GetStatus(),
		Busy:   r.GetBusy(),
		Labels: labels,
	}
}
