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

// HasBreakingFooter reports whether text contains a "BREAKING CHANGE: " or
// "BREAKING-CHANGE: " footer. The token must start a line and be followed by
// ": ". Not counted: a mention in the middle of a sentence or in an indented
// line, and anything inside a fenced code block, an HTML comment
// (<!-- ... -->) or a <details> block. Fences follow CommonMark: an opening
// fence is three or more backticks or tildes indented by at most three spaces;
// it is closed by a fence of the same character, at least as long, without an
// info string. Four spaces of indentation are an indented code block and open
// no fence.
func HasBreakingFooter(text string) bool {
	var fence string // the opening fence while inside a fenced code block
	inComment := false
	details := 0 // nesting depth of <details> blocks
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case fence != "":
			if closesFence(line, fence) {
				fence = ""
			}
			continue
		case inComment:
			if strings.Contains(line, "-->") {
				inComment = false
			}
			continue
		}
		lower := strings.ToLower(line)
		details += strings.Count(lower, "<details") - strings.Count(lower, "</details>")
		if details < 0 {
			details = 0
		}
		if open := strings.LastIndex(line, "<!--"); open >= 0 && !strings.Contains(line[open:], "-->") {
			inComment = true
			continue
		}
		if details > 0 || strings.Contains(lower, "</details>") {
			continue
		}
		if f := openingFence(line); f != "" {
			fence = f
			continue
		}
		if strings.HasPrefix(line, "BREAKING CHANGE: ") || strings.HasPrefix(line, "BREAKING-CHANGE: ") {
			return true
		}
	}
	return false
}

// openingFence returns the fence (for example "````") a line opens, or "".
func openingFence(line string) string {
	rest, ok := fenceIndent(line)
	if !ok || (rest[0] != '`' && rest[0] != '~') {
		return ""
	}
	n := len(rest) - len(strings.TrimLeft(rest, rest[:1]))
	if n < 3 || (rest[0] == '`' && strings.Contains(rest[n:], "`")) {
		return ""
	}
	return rest[:n]
}

// closesFence reports whether line closes the code block opened by fence.
func closesFence(line, fence string) bool {
	rest, ok := fenceIndent(line)
	if !ok || rest[0] != fence[0] {
		return false
	}
	n := len(rest) - len(strings.TrimLeft(rest, rest[:1]))
	return n >= len(fence) && strings.TrimSpace(rest[n:]) == ""
}

// fenceIndent strips up to three spaces of indentation. It reports false for an
// empty line and for four or more spaces (an indented code block).
func fenceIndent(line string) (string, bool) {
	rest := strings.TrimLeft(line, " ")
	if rest == "" || len(line)-len(rest) > 3 {
		return "", false
	}
	return rest, true
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
