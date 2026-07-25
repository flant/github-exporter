package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v88/github"
)

type fakeRunnerService struct {
	pages []*github.Runners
	resps []*github.Response
	err   error
	calls int
}

func (f *fakeRunnerService) ListOrganizationRunners(_ context.Context, _ string, _ *github.ListRunnersOptions) (*github.Runners, *github.Response, error) {
	if f.err != nil {
		return nil, nil, f.err
	}
	i := f.calls
	f.calls++
	return f.pages[i], f.resps[i], nil
}

func ghRunner(id int64, name, os, status string, busy bool, labels ...string) *github.Runner {
	var rl []*github.RunnerLabels
	for _, l := range labels {
		name := l
		rl = append(rl, &github.RunnerLabels{Name: &name})
	}
	return &github.Runner{
		ID:     &id,
		Name:   &name,
		OS:     &os,
		Status: &status,
		Busy:   &busy,
		Labels: rl,
	}
}

func TestListRunnersPaginates(t *testing.T) {
	svc := &fakeRunnerService{
		pages: []*github.Runners{
			{Runners: []*github.Runner{ghRunner(1, "r1", "linux", "online", false, "self-hosted")}},
			{Runners: []*github.Runner{ghRunner(2, "r2", "linux", "offline", false)}},
		},
		resps: []*github.Response{
			{NextPage: 2},
			{NextPage: 0},
		},
	}
	c := &Client{org: "acme", runners: svc}

	runners, err := c.ListRunners(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runners) != 2 {
		t.Fatalf("want 2 runners, got %d", len(runners))
	}
	if svc.calls != 2 {
		t.Fatalf("want 2 API calls (pagination), got %d", svc.calls)
	}
	if !runners[0].Online() {
		t.Errorf("runner r1 should be online")
	}
	if runners[1].Online() {
		t.Errorf("runner r2 should be offline")
	}
	if len(runners[0].Labels) != 1 || runners[0].Labels[0] != "self-hosted" {
		t.Errorf("labels not mapped: %#v", runners[0].Labels)
	}
}

func TestListRunnersError(t *testing.T) {
	svc := &fakeRunnerService{err: errors.New("boom")}
	c := &Client{org: "acme", runners: svc}

	if _, err := c.ListRunners(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestToRunnerNil(t *testing.T) {
	if got := toRunner(nil); got.ID != 0 || got.Name != "" || got.Labels != nil {
		t.Errorf("toRunner(nil) = %#v, want zero value", got)
	}
}
