package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	sha1 = "1111111111111111111111111111111111111111"
	sha2 = "2222222222222222222222222222222222222222"
)

func lookups(released bool, releasedErr error, candidates map[string]string, candErr error) gateLookups {
	return gateLookups{
		versionReleased:  func(string) (bool, error) { return released, releasedErr },
		candidateCommits: func(string) (map[string]string, error) { return candidates, candErr },
	}
}

func byName(checks []GateCheck) map[string]GateCheck {
	m := map[string]GateCheck{}
	for _, c := range checks {
		m[c.Name] = c
	}
	return m
}

const datedChangelog = "# Changelog\n\n## 1.2.0 — 2026-10-02\n\n- a change\n\n## 1.1.0 — 2026-09-01\n\n- old\n"

func TestEvaluateGatePasses(t *testing.T) {
	checks := evaluateGate(datedChangelog, sha1, true,
		lookups(false, nil, map[string]string{"1.2.0-rc.1": sha2, "1.2.0-rc.2": sha1}, nil))
	require.True(t, gatePassed(checks), "%+v", checks)
	require.Len(t, checks, 3)
	require.Contains(t, byName(checks)["release candidate for commit"].Detail, "1.2.0-rc.2")
}

func TestEvaluateGateWithoutCandidateRequirement(t *testing.T) {
	checks := evaluateGate(datedChangelog, "", false, lookups(false, nil, nil, nil))
	require.True(t, gatePassed(checks))
	require.Len(t, checks, 2, "no candidate check when not required")
}

func TestEvaluateGateFailures(t *testing.T) {
	tests := []struct {
		name      string
		changelog string
		lk        gateLookups
		failing   string
	}{
		{"undated heading", "## 1.2.0\n\n- x\n", lookups(false, nil, map[string]string{"1.2.0-rc.1": sha1}, nil), "changelog heading"},
		{"invalid heading", "## Unreleased\n", lookups(false, nil, nil, nil), "changelog heading"},
		{"empty notes", "## 1.2.0 — 2026-10-02\n\n## 1.1.0 — 2026-09-01\n- old\n", lookups(false, nil, map[string]string{"1.2.0-rc.1": sha1}, nil), "changelog heading"},
		{"already released", datedChangelog, lookups(true, nil, map[string]string{"1.2.0-rc.1": sha1}, nil), "version not released yet"},
		{"release lookup fails", datedChangelog, lookups(false, errors.New("api down"), map[string]string{"1.2.0-rc.1": sha1}, nil), "version not released yet"},
		{"candidate for another commit", datedChangelog, lookups(false, nil, map[string]string{"1.2.0-rc.1": sha2}, nil), "release candidate for commit"},
		{"no candidate", datedChangelog, lookups(false, nil, map[string]string{}, nil), "release candidate for commit"},
		{"candidate lookup fails", datedChangelog, lookups(false, nil, nil, errors.New("boom")), "release candidate for commit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checks := evaluateGate(tt.changelog, sha1, true, tt.lk)
			require.False(t, gatePassed(checks))
			require.False(t, byName(checks)[tt.failing].OK, "%s should fail: %+v", tt.failing, checks)
		})
	}
}

func TestEvaluateGateReportsEverything(t *testing.T) {
	// several things wrong at once: all are reported
	checks := evaluateGate("## 1.2.0\n\n- x\n", sha1, true, lookups(true, nil, map[string]string{}, nil))
	require.Len(t, checks, 3)
	for _, c := range checks {
		require.False(t, c.OK, c.Name)
	}
}
