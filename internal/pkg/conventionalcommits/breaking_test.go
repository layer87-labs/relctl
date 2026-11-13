package conventionalcommits

import "testing"

func TestHasBreakingHeader(t *testing.T) {
	cases := map[string]bool{
		"fix!: y":                    true,
		"fix(forms)!: y":             true,
		"feat(api)!: drop v1":        true,
		"  refactor!: y":             true,
		"fix: y":                     false,
		"fix(forms): y":              false,
		"fix: handle foo!":           false,
		"fix: y!: z":                 false,
		"fix!:y":                     false, // no space after the colon: not a header
		"Merge pull request #1 from": false,
		`Revert "feat!: x"`:          false,
		"":                           false,
	}
	for in, want := range cases {
		if got := HasBreakingHeader(in); got != want {
			t.Errorf("HasBreakingHeader(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestHasBreakingFooter(t *testing.T) {
	cases := map[string]bool{
		"BREAKING CHANGE: z":                      true,
		"BREAKING-CHANGE: z":                      true,
		"feat: y\n\nbody\n\nBREAKING CHANGE: z":   true,
		"feat: y\r\n\r\nBREAKING CHANGE: z\r\n":   true,
		"BREAKING CHANGE:z":                       false, // needs ": "
		"BREAKING CHANGE:":                        false,
		"This is no BREAKING CHANGE: just a fix.": false,
		"  BREAKING CHANGE: indented":             false,
		"BREAKING CHANGES: plural":                false,
		"breaking change: lower case":             false,
		"":                                        false,

		// fenced code blocks (CommonMark)
		"```\nBREAKING CHANGE: z\n```":                  false,
		"~~~\nBREAKING CHANGE: z\n~~~":                  false,
		"```go\nx := 1\n```\nBREAKING CHANGE: z":        true, // after the block
		"   ```\nBREAKING CHANGE: z\n   ```":            false,
		"text\n  ```\nBREAKING CHANGE: z\n  ```\n":      false,
		"```\nunterminated block\nBREAKING CHANGE: z\n": false,
		"````\n```\nBREAKING CHANGE: z\n````":           false, // a shorter fence does not close
		"```\n~~~\nBREAKING CHANGE: z\n```":             false, // another character does not close
		"```\n``` text\nBREAKING CHANGE: z\n```":        false, // a closing fence has no info string
		"```\nx\n`````\nBREAKING CHANGE: z":             true,  // a longer fence closes
		"    ```\nBREAKING CHANGE: z\n    ```":          true,  // indented code, no fence

		// HTML comments and <details>
		"<!--\nBREAKING CHANGE: z\n-->":                                                           false,
		"<!-- template:\nBREAKING CHANGE: z\n-->\n":                                               false,
		"<!-- one line -->\nBREAKING CHANGE: z":                                                   true,
		"<details>\n<summary>Release notes</summary>\n\nBREAKING CHANGE: z\n</details>":           false,
		"<details><summary>x</summary>\n<details>\nx\n</details>\nBREAKING CHANGE: z\n</details>": false,
		"<details>\nx\n</details>\n\nBREAKING CHANGE: z":                                          true,
	}
	for in, want := range cases {
		if got := HasBreakingFooter(in); got != want {
			t.Errorf("HasBreakingFooter(%q) = %v, want %v", in, got, want)
		}
	}
}
