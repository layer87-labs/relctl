package rcpersist

import (
	"errors"
	"fmt"
	"os"

	"github.com/layer87-labs/relctl/internal/pkg/tools"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func (t Token) ToString() string {
	return string(t)
}

// Decrypt reads a token that an earlier version stored in the legacy file.
func (t Token) Decrypt() (string, error) {
	decryptedToken, err := tools.DecryptString(t.ToString(), legacyTokenKey)
	if err != nil {
		return "", fmt.Errorf("could not decrypt token: %v", err)
	}
	return decryptedToken, nil
}

func NewRcInstance() *RcFile {
	return &RcFile{}
}

// activeFile returns the connection file in use: the current one, else the legacy one.
func activeFile() (name string, legacy bool) {
	if tools.CheckFileExists(rcFileName) {
		return rcFileName, false
	}
	if tools.CheckFileExists(legacyRcFileName) {
		return legacyRcFileName, true
	}
	return "", false
}

func (rc *RcFile) Exists() bool {
	name, _ := activeFile()
	return name != ""
}

// Load reads the connection file. A token is resolved separately by PlainToken.
func (rc *RcFile) Load() (creds *ConnectCredentials, err error) {
	name, legacy := activeFile()
	if name == "" {
		return nil, ErrRcFileNotExists
	}

	yamlFile, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("error while opening existing connection file. %v", err)
	}
	if err = yaml.Unmarshal(yamlFile, rc); err != nil {
		return nil, fmt.Errorf("error at reading file %s: %v", name, err)
	}
	log.Tracef("Successfully read file %s", name)

	if legacy {
		log.Warnf("%s is a legacy connection file; run 'relctl connect github' to replace it with %s (no token is stored any more)", name, rcFileName)
	}
	return &rc.ConnectCreds, nil
}

// PlainToken returns the token: GITHUB_TOKEN first, then a token from a legacy file.
func (rc *RcFile) PlainToken() (string, error) {
	if token := os.Getenv(tokenEnvName); token != "" {
		return token, nil
	}
	if rc.ConnectCreds.Token != "" {
		return rc.ConnectCreds.Token.Decrypt()
	}
	return "", ErrNoToken
}

func (rc *RcFile) UpdateSCMPortalType(scmPortalType SCMPortalType) {
	rc.SCMPortalType = scmPortalType
}

func (rc *RcFile) UpdateCESType(cesType CESType) {
	rc.CESType = cesType
}

func (rc *RcFile) UpdateCreds(server string, repo string) {
	rc.ConnectCreds.ServerURL = server
	rc.ConnectCreds.Repository = repo
}

// Save writes server and repository. It never writes a token and removes a legacy file.
func (rc *RcFile) Save() (err error) {
	rc.ConnectCreds.Token = ""
	rc.ConnectCreds.TokenPlain = nil

	yamlData, err := yaml.Marshal(rc)
	if err != nil {
		return fmt.Errorf("error while Marshaling. %v", err)
	}
	if err := os.WriteFile(rcFileName, yamlData, 0600); err != nil {
		return fmt.Errorf("unable to write data into %s: %v", rcFileName, err)
	}
	log.Tracef("Successfully written config to %s", rcFileName)

	if err := os.Remove(legacyRcFileName); err == nil {
		log.Infof("removed legacy %s, it contained a token", legacyRcFileName)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("could not remove legacy %s: %v", legacyRcFileName, err)
	}
	return nil
}

// RemoveRcFile removes the connection file and a legacy file, if present.
func RemoveRcFile() {
	removed := false
	for _, name := range []string{rcFileName, legacyRcFileName} {
		err := os.Remove(name)
		switch {
		case err == nil:
			removed = true
		case !errors.Is(err, os.ErrNotExist):
			log.Fatalf("error at removing %s: %v", name, err)
		}
	}
	if !removed {
		log.Fatalf("no connection file found (%s)", rcFileName)
	}
}
