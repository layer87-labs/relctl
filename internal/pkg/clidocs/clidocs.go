// Package clidocs writes the command line reference of a cobra command tree as
// Markdown pages for a Docusaurus site. The output is deterministic: no dates, no
// absolute paths, so a regeneration without a change of the CLI changes nothing.
package clidocs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// Generate writes index.md and one page per top-level command into dir. Markdown
// files that are already in dir are removed first, so a command that no longer
// exists does not leave a page behind. Other files (for example _category_.json)
// stay.
func Generate(root *cobra.Command, dir string) error {
	if dir == "" {
		return fmt.Errorf("output directory is empty")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	old, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return err
	}
	for _, f := range old {
		if err := os.Remove(f); err != nil {
			return err
		}
	}

	cmds := visibleCommands(root)
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(indexPage(root, cmds)), 0o644); err != nil {
		return err
	}
	for i, c := range cmds {
		page := commandPage(root, c, i+2)
		if err := os.WriteFile(filepath.Join(dir, c.Name()+".md"), []byte(page), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func visibleCommands(c *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	for _, sub := range c.Commands() {
		if sub.Hidden || sub.IsAdditionalHelpTopicCommand() || sub.Name() == "help" || sub.Name() == "completion" {
			continue
		}
		out = append(out, sub)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

func indexPage(root *cobra.Command, cmds []*cobra.Command) string {
	var b strings.Builder
	b.WriteString("---\nsidebar_position: 1\ntitle: CLI reference\n---\n\n")
	b.WriteString(generatedNotice)
	b.WriteString("# relctl\n\n")
	b.WriteString(fence(strings.TrimSpace(root.Long)))
	b.WriteString("\n\n")
	b.WriteString("| Command | Description |\n| --- | --- |\n")
	for _, c := range cmds {
		fmt.Fprintf(&b, "| [`relctl %s`](./%s.md) | %s |\n", c.Name(), c.Name(), escapeCell(c.Short))
	}
	b.WriteString("\n## Flags of every command\n\n")
	b.WriteString(fence(strings.TrimRight(root.PersistentFlags().FlagUsages(), "\n")))
	b.WriteString("\n")
	return b.String()
}

func commandPage(root, c *cobra.Command, position int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "---\nsidebar_position: %d\ntitle: %s\n---\n\n", position, c.Name())
	b.WriteString(generatedNotice)
	section(&b, "#", root, c)
	for _, sub := range visibleCommands(c) {
		b.WriteString("\n")
		section(&b, "##", root, sub)
		for _, subsub := range visibleCommands(sub) {
			b.WriteString("\n")
			section(&b, "###", root, subsub)
		}
	}
	return b.String()
}

// section writes one command: heading, what it does, usage and its own flags.
func section(b *strings.Builder, level string, root, c *cobra.Command) {
	fmt.Fprintf(b, "%s %s\n\n", level, c.CommandPath())
	if c.Short != "" {
		fmt.Fprintf(b, "%s\n\n", c.Short)
	}
	if long := strings.TrimSpace(c.Long); long != "" && long != c.Short {
		b.WriteString(fence(long))
		b.WriteString("\n\n")
	}
	if c.Runnable() {
		b.WriteString(fence(c.UseLine()))
		b.WriteString("\n\n")
	}
	if flags := strings.TrimRight(c.LocalNonPersistentFlags().FlagUsages(), "\n"); flags != "" {
		b.WriteString("Flags:\n\n")
		b.WriteString(fence(flags))
		b.WriteString("\n\n")
	}
	if flags := strings.TrimRight(c.PersistentFlags().FlagUsages(), "\n"); flags != "" && c.HasSubCommands() {
		b.WriteString("Flags of this command and of its subcommands:\n\n")
		b.WriteString(fence(flags))
		b.WriteString("\n\n")
	}
	if c.Example != "" {
		b.WriteString("Examples:\n\n")
		b.WriteString(fence(strings.TrimSpace(c.Example)))
		b.WriteString("\n\n")
	}
}

// fence wraps text in a code block. Descriptions contain characters such as '<'
// and '{' that a Markdown/MDX renderer would take for markup.
func fence(text string) string {
	text = strings.ReplaceAll(text, "\t", "    ")
	marker := "```"
	for strings.Contains(text, marker) {
		marker += "`"
	}
	return marker + "text\n" + text + "\n" + marker
}

// generatedNotice is shown at the top of every page.
const generatedNotice = ":::note\nThis page is generated from the command line of relctl. Do not edit it by hand.\n:::\n\n"

func escapeCell(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
