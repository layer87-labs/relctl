package rcpersist

import (
	"os"
	"strings"
	"testing"

	"github.com/layer87-labs/relctl/internal/pkg/tools"

	"gopkg.in/yaml.v2"
)

func TestSaveStoresNoToken(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GITHUB_TOKEN", "ghp_secret")

	rc := NewRcInstance()
	rc.UpdateSCMPortalType(SCMPortalTypeGitHub)
	rc.UpdateCreds("https://github.com", "octo/repo")
	if err := rc.Save(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(rcFileName)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "ghp_secret") || strings.Contains(string(data), "token") {
		t.Fatalf("connection file must not contain a token:\n%s", data)
	}

	loaded := NewRcInstance()
	creds, err := loaded.Load()
	if err != nil {
		t.Fatal(err)
	}
	if creds.ServerURL != "https://github.com" || creds.Repository != "octo/repo" || loaded.SCMPortalType != SCMPortalTypeGitHub {
		t.Fatalf("unexpected content: %+v", loaded)
	}
}

func TestPlainTokenFromEnvironment(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GITHUB_TOKEN", "from-env")
	rc := NewRcInstance()
	got, err := rc.PlainToken()
	if err != nil || got != "from-env" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestPlainTokenMissing(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GITHUB_TOKEN", "")
	if _, err := NewRcInstance().PlainToken(); err != ErrNoToken {
		t.Fatalf("want ErrNoToken, got %v", err)
	}
}

func writeLegacy(t *testing.T, token string) {
	t.Helper()
	enc, err := tools.EncryptString(token, legacyTokenKey)
	if err != nil {
		t.Fatal(err)
	}
	// written the way earlier versions did: the encrypted value goes through the YAML marshaller
	legacy := RcFile{
		ConnectCreds:  ConnectCredentials{ServerURL: "https://github.com", Repository: "octo/legacy", Token: Token(enc)},
		SCMPortalType: SCMPortalTypeGitHub,
	}
	content, err := yaml.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyRcFileName, content, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyFileIsReadAndEnvironmentWins(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GITHUB_TOKEN", "")
	writeLegacy(t, "legacy-token")

	rc := NewRcInstance()
	creds, err := rc.Load()
	if err != nil {
		t.Fatal(err)
	}
	if creds.Repository != "octo/legacy" {
		t.Fatalf("unexpected repository %q", creds.Repository)
	}
	if got, err := rc.PlainToken(); err != nil || got != "legacy-token" {
		t.Fatalf("legacy token: got %q, %v", got, err)
	}

	t.Setenv("GITHUB_TOKEN", "env-token")
	if got, _ := rc.PlainToken(); got != "env-token" {
		t.Fatalf("environment must win, got %q", got)
	}
}

func TestSaveReplacesLegacyFile(t *testing.T) {
	t.Chdir(t.TempDir())
	writeLegacy(t, "legacy-token")

	rc := NewRcInstance()
	if _, err := rc.Load(); err != nil {
		t.Fatal(err)
	}
	if err := rc.Save(); err != nil {
		t.Fatal(err)
	}
	if tools.CheckFileExists(legacyRcFileName) {
		t.Fatal("legacy file with a token must be removed on save")
	}
	data, _ := os.ReadFile(rcFileName)
	if strings.Contains(string(data), "token") {
		t.Fatalf("new file must not contain a token:\n%s", data)
	}
}

func TestCurrentFileWinsOverLegacy(t *testing.T) {
	t.Chdir(t.TempDir())
	writeLegacy(t, "legacy-token")
	if err := os.WriteFile(rcFileName, []byte("connectcreds:\n  repository: octo/current\n"), 0600); err != nil {
		t.Fatal(err)
	}
	creds, err := NewRcInstance().Load()
	if err != nil || creds.Repository != "octo/current" {
		t.Fatalf("got %+v, %v", creds, err)
	}
}

func TestLoadWithoutFile(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := NewRcInstance().Load(); err != ErrRcFileNotExists {
		t.Fatalf("want ErrRcFileNotExists, got %v", err)
	}
}
