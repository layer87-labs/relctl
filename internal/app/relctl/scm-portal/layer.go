package scmportal

import (
	"fmt"

	"github.com/layer87-labs/relctl/internal/app/relctl/ces"
	"github.com/layer87-labs/relctl/internal/app/relctl/scm-portal/github"
	"github.com/layer87-labs/relctl/internal/pkg/rcpersist"
)

func LoadSCMPortalLayer() (scmLayer *SCMLayer, err error) {
	ces, scmPortalType, connCreds, err := ces.DetectCes()
	if err != nil {
		return &SCMLayer{
			CES: ces,
		}, err
	}

	switch scmPortalType {
	case rcpersist.SCMPortalTypeGitHub:
		ghrc, err := github.NewGitHubClient(&connCreds.ServerURL, &connCreds.Repository, &connCreds.Token)
		if err != nil {
			return nil, err
		}
		return &SCMLayer{
			CES: ces,
			Grc: ghrc,
		}, err
	}

	return nil, unsupportedSCMError(scmPortalType)
}

// IssuesURL is where a missing capability is requested.
const IssuesURL = "https://github.com/layer87-labs/relctl/issues"

// unsupportedSCMError explains why no SCM layer could be created, instead of
// letting the caller run into a nil layer.
func unsupportedSCMError(t rcpersist.SCMPortalType) error {
	switch t {
	case rcpersist.SCMPortalTypeGitLab:
		return fmt.Errorf("GitLab is not supported as source control system. relctl currently supports GitHub and GitHub Enterprise "+
			"(the CI system can be GitHub Actions, GitLab CI, Jenkins or a custom pipeline). "+
			"If you need GitLab, please open an issue and describe your use case: %s", IssuesURL)
	case "":
		return fmt.Errorf("no CI environment or connection detected. Run in GitHub Actions, in Jenkins or GitLab CI " +
			"with GITHUB_REPOSITORY and GITHUB_TOKEN set, or run 'relctl connect github' first")
	default:
		return fmt.Errorf("source control system %q is not supported. relctl currently supports GitHub and GitHub Enterprise. "+
			"If you need another one, please open an issue: %s", string(t), IssuesURL)
	}
}
