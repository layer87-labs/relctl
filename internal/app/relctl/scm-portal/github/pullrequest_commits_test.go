package github

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-github/v86/github"
	"github.com/layer87-labs/relctl/internal/pkg/conventionalcommits"
	"github.com/stretchr/testify/require"
)

// The breaking-change check reads only the pull request's own commits. Any other
// request (the merge commit, the commits of the target branch) fails the test.
func TestPullRequestCommitsReadsOnlyThePullRequest(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/example-org/example-repo/pulls/7/commits" {
			t.Errorf("unexpected request %s", r.URL)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprint(w, `[{"sha":"bbbb","commit":{"message":"fix: y\n\nBREAKING CHANGE: z"}}]`)
			return
		}
		w.Header().Set("Link", fmt.Sprintf(`<%s%s?page=2>; rel="next"`, srv.URL, r.URL.Path))
		fmt.Fprint(w, `[{"sha":"aaaa","commit":{"message":"fix: x"}}]`)
	}))
	defer srv.Close()

	client := github.NewClient(nil)
	base, err := url.Parse(srv.URL + "/")
	require.NoError(t, err)
	client.BaseURL = base
	ghrc := &GitHubRichClient{Client: client, Owner: "example-org", Repository: "example-repo"}

	commits, err := ghrc.PullRequestCommits(7)
	require.NoError(t, err)
	require.Equal(t, []conventionalcommits.Commit{
		{SHA: "aaaa", Message: "fix: x"},
		{SHA: "bbbb", Message: "fix: y\n\nBREAKING CHANGE: z"},
	}, commits)
}

func commitsServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request, base string)) *GitHubRichClient {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		handler(w, r, srv.URL)
	}))
	t.Cleanup(srv.Close)
	client := github.NewClient(nil)
	base, err := url.Parse(srv.URL + "/")
	require.NoError(t, err)
	client.BaseURL = base
	return &GitHubRichClient{Client: client, Owner: "example-org", Repository: "example-repo"}
}

func TestPullRequestCommitsCappedAt250(t *testing.T) {
	ghrc := commitsServer(t, func(w http.ResponseWriter, r *http.Request, base string) {
		page := r.URL.Query().Get("page")
		n := 100
		switch page {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(`<%s%s?page=2>; rel="next"`, base, r.URL.Path))
		case "2":
			w.Header().Set("Link", fmt.Sprintf(`<%s%s?page=3>; rel="next"`, base, r.URL.Path))
		default:
			n = 50
		}
		items := make([]string, n)
		for i := range items {
			items[i] = fmt.Sprintf(`{"sha":"p%s-%d","commit":{"message":"fix: x"}}`, page, i)
		}
		fmt.Fprint(w, "["+strings.Join(items, ",")+"]")
	})
	commits, err := ghrc.PullRequestCommits(7)
	require.ErrorIs(t, err, ErrCommitsCapped)
	require.Len(t, commits, 250)
}

func TestPullRequestCommitsServerError(t *testing.T) {
	ghrc := commitsServer(t, func(w http.ResponseWriter, r *http.Request, _ string) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"message":"boom"}`)
	})
	commits, err := ghrc.PullRequestCommits(7)
	require.Error(t, err)
	require.Nil(t, commits)
}
