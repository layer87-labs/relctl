package conventionalcommits

import "testing"

func TestClassifyMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want Bump
	}{
		{"plain fix", "fix: correct heartbeat timestamp parsing", BumpPatch},
		{"plain feat", "feat: add node filtering by type", BumpMinor},
		{"scoped fix", "fix(api): correct pagination", BumpPatch},
		{"scoped feat", "feat(landscape): add node filtering by type", BumpMinor},
		{"feat with bang", "feat!: remove legacy v0 API endpoints", BumpMajor},
		{"scoped fix with bang", "fix(api)!: drop deprecated field", BumpMajor},
		{"breaking change footer", "feat: add new auth flow\n\nBREAKING CHANGE: old tokens are rejected", BumpMajor},
		{"breaking-change footer with dash", "fix: change response shape\n\nBREAKING-CHANGE: field renamed", BumpMajor},
		{"breaking footer not at line start is ignored", "fix: tweak docs\n\nsee also BREAKING CHANGE: unrelated mention", BumpPatch},
		{"unknown type", "wip: half finished thing", BumpNone},
		{"no type at all", "quick fix for the build", BumpNone},
		{"merge commit subject", "Merge pull request #3 from org/feature/awesome-feature", BumpNone},
		{"perf", "perf(db): add index on scope_nodes.parent_id", BumpPatch},
		{"refactor", "refactor: simplify handler", BumpPatch},
		{"docs", "docs: update readme", BumpPatch},
		{"style", "style: gofmt", BumpPatch},
		{"test", "test: add table-driven cases", BumpPatch},
		{"chore", "chore: bump deps", BumpPatch},
		{"ci", "ci: add workflow", BumpPatch},
		{"build", "build: update makefile", BumpPatch},
		{"revert", "revert: undo previous change", BumpPatch},
		{"empty message", "", BumpNone},
		{"type with no space after colon", "fix:no space", BumpNone},
		{"case insensitive type", "FIX: correct casing", BumpPatch},
		{"case insensitive feat", "FEAT: add thing", BumpMinor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyMessage(tt.msg)
			if got != tt.want {
				t.Errorf("ClassifyMessage(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		name     string
		messages []string
		want     Bump
	}{
		{"empty set", nil, BumpNone},
		{"only unrecognised", []string{"wip: stuff", "Merge pull request #1 from org/x"}, BumpNone},
		{"mixed fix and feat picks minor", []string{"fix: a", "feat: b"}, BumpMinor},
		{"mixed feat and breaking picks major", []string{"feat: a", "fix!: b"}, BumpMajor},
		{"single fix", []string{"fix: a"}, BumpPatch},
		{"breaking body among patches", []string{"fix: a", "chore: b\n\nBREAKING CHANGE: c"}, BumpMajor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.messages)
			if got != tt.want {
				t.Errorf("Classify(%v) = %v, want %v", tt.messages, got, tt.want)
			}
		})
	}
}

func TestBumpString(t *testing.T) {
	tests := map[Bump]string{
		BumpNone:  "none",
		BumpPatch: "patch",
		BumpMinor: "minor",
		BumpMajor: "major",
	}
	for b, want := range tests {
		if got := b.String(); got != want {
			t.Errorf("Bump(%d).String() = %q, want %q", b, got, want)
		}
	}
}
