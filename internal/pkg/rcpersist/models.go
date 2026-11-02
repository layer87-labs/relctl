package rcpersist

import "fmt"

var (
	// legacyTokenKey decrypts tokens stored by earlier versions in .awesomerc.yaml.
	// The key is public, so such a file protected nothing; relctl only reads it
	// for migration and never writes a token any more.
	legacyTokenKey = []byte("TZPtSIacEJG18IpqQSkTE6luYmnCNKgR")

	ErrRcFileNotExists = fmt.Errorf("connection file does not exist. please connect first")
	ErrNoToken         = fmt.Errorf("no token available: set GITHUB_TOKEN (the connection file stores no token)")
)

const (
	// rcFileName holds server and repository only, never a token.
	rcFileName = ".relctl-connection.yaml"
	// legacyRcFileName is the file of awesome-ci and of relctl before 2.10.
	legacyRcFileName                     = ".awesomerc.yaml"
	tokenEnvName                         = "GITHUB_TOKEN"
	SCMPortalTypeGitHub    SCMPortalType = "github"
	SCMPortalTypeGitLab    SCMPortalType = "gitlab"
	CESTypeGitHubRunner    CESType       = "github_runner"
	CESTypeGitLabRunner    CESType       = "gitlab_runner"
	CESTypeJenkinsPipeline CESType       = "jenkins_pipeline"
	CESTypeLoMa            CESType       = "locally_manually"
)

type SCMPortalType string
type CESType string
type Token string

type RcFile struct {
	ConnectCreds  ConnectCredentials
	CESType       CESType
	SCMPortalType SCMPortalType
}

type ConnectCredentials struct {
	ServerURL  string `yaml:"server_url"`
	Repository string `yaml:"repository"`
	// Token is only read from a legacy file and never written.
	Token      Token   `yaml:"token,omitempty"`
	TokenPlain *string `yaml:"tplain,omitempty"`
}
