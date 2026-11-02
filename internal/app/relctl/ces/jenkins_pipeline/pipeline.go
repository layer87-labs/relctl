package jenkinspipeline

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/layer87-labs/relctl/internal/pkg/models"
)

func DetectJenkinsPipeline() (connects *models.ConnectCredentials, err error) {
	ci, ciPresent := os.LookupEnv("CI")
	isCI := ci == "true" && ciPresent

	_, isjenkinsURL := os.LookupEnv("JENKINS_URL")

	gitURL, isServerURL := os.LookupEnv("GIT_URL")
	serverURL := ServerURL(os.Getenv("GITHUB_SERVER_URL"), gitURL)
	repository, isRepository := os.LookupEnv("GITHUB_REPOSITORY")
	token, isToken := os.LookupEnv("GITHUB_TOKEN")

	if isCI && isjenkinsURL && isServerURL && isRepository && isToken {
		return &models.ConnectCredentials{
			ServerURL:  serverURL,
			Repository: repository,
			Token:      token,
		}, nil

	} else {
		return nil, fmt.Errorf("no jenkins pipeline detected")
	}
}

var scpLikeRe = regexp.MustCompile(`^[^@/\s]+@([^:/\s]+):`)

// ServerURL returns the base URL of the GitHub server (scheme and host, no
// repository path). An explicit GITHUB_SERVER_URL wins; otherwise it is derived
// from Jenkins' GIT_URL, which is the URL of the repository and may be an https,
// ssh or scp-like URL. The repository path must not be part of the server URL: it
// would end up in the API address of GitHub Enterprise.
func ServerURL(explicit, gitURL string) string {
	if explicit != "" {
		return strings.TrimRight(explicit, "/")
	}
	if m := scpLikeRe.FindStringSubmatch(gitURL); m != nil {
		return "https://" + m[1]
	}
	u, err := url.Parse(gitURL)
	if err != nil || u.Host == "" {
		return gitURL
	}
	scheme := u.Scheme
	if scheme != "http" && scheme != "https" {
		scheme = "https"
	}
	return scheme + "://" + u.Hostname()
}
