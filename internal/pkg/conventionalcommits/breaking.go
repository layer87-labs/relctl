package conventionalcommits

import (
	"fmt"
	"strings"
)

// HasBreakingHeader reports whether a single header line carries the
// Conventional Commits breaking-change marker: "!" directly after the type or
// the scope ("fix!: ...", "fix(forms)!: ..."). A "!" anywhere else in the line
// ("fix: handle foo!") does not count. The type itself is not checked against
// the list of known types: an author who marks a change as breaking means it.
func HasBreakingHeader(header string) bool {
	m := headerRe.FindStringSubmatch(strings.TrimSpace(header))
	return m != nil && m[3] == "!"
}

// HasBreakingFooter reports whether text contains a "BREAKING CHANGE:" or
// "BREAKING-CHANGE:" footer. The token must start a line; a mention in the
// middle of a sentence, in an indented line or inside a fenced code block
// (``` or ~~~) does not count.
func HasBreakingFooter(text string) bool {
	fence := ""
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimLeft(line, " ")
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			fence = "```"
			continue
		}
		if strings.HasPrefix(trimmed, "~~~") {
			fence = "~~~"
			continue
		}
		if strings.HasPrefix(line, "BREAKING CHANGE:") || strings.HasPrefix(line, "BREAKING-CHANGE:") {
			return true
		}
	}
	return false
}

// BreakingMarker looks for a breaking-change marker in a pull request: a "!"
// after the type/scope of the title or of a commit subject, or a breaking-change
// footer in the description or in a commit message. commits must be the
// commits of the pull request only. It returns where the first marker was found
// ("the pull request title", "commit 1a2b3c4d") or "" if there is none.
func BreakingMarker(title, body string, commits []Commit) string {
	if HasBreakingHeader(title) {
		return "the pull request title"
	}
	if HasBreakingFooter(body) {
		return "the pull request description"
	}
	for _, c := range commits {
		subject, _, _ := strings.Cut(c.Message, "\n")
		if HasBreakingHeader(subject) || HasBreakingFooter(c.Message) {
			return fmt.Sprintf("commit %s", shortSHA(c.SHA))
		}
	}
	return ""
}

// Commit is a commit of a pull request.
type Commit struct {
	SHA     string
	Message string
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}
