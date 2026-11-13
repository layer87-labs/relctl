package scmportal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	gogithub "github.com/google/go-github/v86/github"
	"github.com/layer87-labs/relctl/internal/app/relctl/scm-portal/github"
)

// fakeGitHub serves one merged pull request (#7, branch fix/x, title "fix!: y").
// commitsStatus is the HTTP status of the commits endpoint; commitCalls counts
// the requests to it.
func fakeGitHub(t *testing.T, comment string, commitsStatus int, commitCalls *atomic.Int32) SCMLayer {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/repos/example-org/example-repo/pulls/7":
			fmt.Fprint(w, `{"number":7,"title":"fix!: y","body":"","head":{"ref":"fix/x","sha":"0123456789abcdef"}}`)
		case "/repos/example-org/example-repo/issues/7/comments":
			if comment == "" {
				fmt.Fprint(w, `[]`)
				return
			}
			fmt.Fprintf(w, `[{"id":1,"body":%q,"user":{"login":"someone"}}]`, comment)
		case "/repos/example-org/example-repo/pulls/7/commits":
			commitCalls.Add(1)
			w.WriteHeader(commitsStatus)
			fmt.Fprint(w, `[]`)
		case "/repos/example-org/example-repo/releases":
			fmt.Fprint(w, `[]`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("RELCTL_SILENT", "true")

	client := gogithub.NewClient(nil)
	base, err := url.Parse(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	client.BaseURL = base
	return SCMLayer{Grc: &github.GitHubRichClient{Client: client, Owner: "example-org", Repository: "example-repo"}}
}

func TestGetPrInfosBreakingMarkers(t *testing.T) {
	t.Run("major fails closed when the commits cannot be read", func(t *testing.T) {
		var calls atomic.Int32
		lay := fakeGitHub(t, "", http.StatusInternalServerError, &calls)
		lay.BreakingMarkers = "major"
		_, err := lay.GetPrInfos(7, "", "semver")
		if err == nil || !strings.Contains(err.Error(), "breaking_markers: major needs the commits") {
			t.Fatalf("err = %v, want the fail-closed error", err)
		}
	})
	t.Run("warn releases with a warning when the commits cannot be read", func(t *testing.T) {
		var calls atomic.Int32
		lay := fakeGitHub(t, "", http.StatusInternalServerError, &calls)
		lay.BreakingMarkers = "warn"
		infos, err := lay.GetPrInfos(7, "", "semver")
		if err != nil {
			t.Fatal(err)
		}
		if infos.NextVersion != "0.0.1" || infos.BreakingMarker != "the pull request title" {
			t.Errorf("next = %q, marker = %q", infos.NextVersion, infos.BreakingMarker)
		}
		if !strings.HasPrefix(infos.BreakingWarning, "The commits of the pull request were not checked") ||
			!strings.Contains(infos.BreakingWarning, "The pull request title marks a breaking change") {
			t.Errorf("warning = %q", infos.BreakingWarning)
		}
	})
	t.Run("major raises a fix branch with a breaking title", func(t *testing.T) {
		var calls atomic.Int32
		lay := fakeGitHub(t, "", http.StatusOK, &calls)
		lay.BreakingMarkers = "major"
		infos, err := lay.GetPrInfos(7, "", "semver")
		if err != nil {
			t.Fatal(err)
		}
		if infos.NextVersion != "1.0.0" || calls.Load() != 1 {
			t.Errorf("next = %q, commit requests = %d", infos.NextVersion, calls.Load())
		}
	})
	t.Run("a version override does not read the commits", func(t *testing.T) {
		var calls atomic.Int32
		lay := fakeGitHub(t, "relctl_version_override: 3.0.0", http.StatusInternalServerError, &calls)
		lay.BreakingMarkers = "major"
		infos, err := lay.GetPrInfos(7, "", "semver")
		if err != nil {
			t.Fatal(err)
		}
		if infos.NextVersion != "3.0.0" || calls.Load() != 0 {
			t.Errorf("next = %q, commit requests = %d", infos.NextVersion, calls.Load())
		}
	})
}
