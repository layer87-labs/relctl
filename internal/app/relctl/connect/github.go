package connect

import (
	"os"

	"github.com/layer87-labs/relctl/internal/app/relctl/scm-portal/github"
	"github.com/layer87-labs/relctl/internal/pkg/rcpersist"

	log "github.com/sirupsen/logrus"
)

// UpdateRcFileForGitHub verifies the connection and stores server and repository.
// The token is used for the check only and is never stored; later calls read GITHUB_TOKEN.
func UpdateRcFileForGitHub(server string, repo string, token string) {
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		log.Fatalln("a token is required to verify the connection: use --token or set GITHUB_TOKEN")
	}

	rcFile := rcpersist.NewRcInstance()

	_, err := rcFile.Load()
	if err != nil && err != rcpersist.ErrRcFileNotExists {
		log.Fatalln(err)
	}

	// testing connection before anything is written
	if _, err := github.NewGitHubClient(&server, &repo, &token); err != nil {
		log.Fatalln(err)
	}

	rcFile.UpdateSCMPortalType(rcpersist.SCMPortalTypeGitHub)
	rcFile.UpdateCreds(server, repo)
	if err := rcFile.Save(); err != nil {
		log.Fatalln(err)
	}
	log.Infof("successfully connected to github at %s; the token is not stored, provide it as GITHUB_TOKEN on every call", server)
}
