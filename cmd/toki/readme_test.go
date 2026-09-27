// ABOUTME: Tests that README.md documents every CLI command and flag.
// ABOUTME: Walks the Cobra command tree so new commands/flags fail until documented.

package main

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// readmePath is relative to this test file (cmd/toki/).
const readmePath = "../../README.md"

// skippedCommands are Cobra-generated commands that don't need README coverage.
var skippedCommands = map[string]bool{
	"help":       true,
	"completion": true,
}

// skippedFlags are always-present flags that don't need explicit documentation.
var skippedFlags = map[string]bool{
	"help": true,
}

// walkCommands invokes fn for every non-hidden, non-skipped command in the tree,
// passing the command and its full invocation path (e.g. "project add").
func walkCommands(fn func(cmd *cobra.Command, path string)) {
	var walk func(cmd *cobra.Command, path string)
	walk = func(cmd *cobra.Command, path string) {
		for _, sub := range cmd.Commands() {
			name := sub.Name()
			if sub.Hidden || skippedCommands[name] {
				continue
			}
			full := strings.TrimSpace(path + " " + name)
			fn(sub, full)
			walk(sub, full)
		}
	}
	walk(rootCmd, "")
}

func loadReadme(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", readmePath, err)
	}
	return string(data)
}

// readmeSection returns only the README text that documents the given command
// path: every fenced code-block line whose trimmed text begins with
// "toki <path>" plus any indented continuation lines immediately following it
// (the flag list for that command).
//
// Scoping the search this way means a flag is only considered documented for
// the command that actually lists it, so reusing a flag name on another
// command does not silently satisfy the coverage check. Requiring the line to
// begin with the command invocation (rather than merely mentioning it) keeps
// prose such as "supplied via `toki add --tags`" from counting as
// documentation for a flag.
func readmeSection(readme, path string) string {
	lines := strings.Split(readme, "\n")
	needle := "toki " + path

	var b strings.Builder
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if !inFence || !strings.HasPrefix(trimmed, needle) {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
		for j := i + 1; j < len(lines); j++ {
			next := lines[j]
			if next == "" || !strings.HasPrefix(next, " ") {
				break
			}
			b.WriteString(next)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// undocumentedFlags returns the names of cmd's flags that are not mentioned in
// the README section documenting path.
func undocumentedFlags(readme string, cmd *cobra.Command, path string) []string {
	section := readmeSection(readme, path)

	var missing []string
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if skippedFlags[f.Name] {
			return
		}
		if !strings.Contains(section, "--"+f.Name) {
			missing = append(missing, f.Name)
		}
	})
	return missing
}

// TestReadmeDocumentsAllCommands ensures every command in the CLI is documented
// in README.md using its full invocation (e.g. "toki project add").
func TestReadmeDocumentsAllCommands(t *testing.T) {
	readme := loadReadme(t)

	walkCommands(func(cmd *cobra.Command, path string) {
		if !strings.Contains(readme, "toki "+path) {
			t.Errorf("README.md does not document command %q (expected to find %q)", path, "toki "+path)
		}
	})
}

// TestReadmeDocumentsAllFlags ensures every flag on every command is mentioned
// in the README section documenting that command.
func TestReadmeDocumentsAllFlags(t *testing.T) {
	readme := loadReadme(t)

	walkCommands(func(cmd *cobra.Command, path string) {
		for _, name := range undocumentedFlags(readme, cmd, path) {
			t.Errorf("README.md does not document flag --%s on command %q", name, path)
		}
	})
}

// TestReadmeFlagScope guards against the coverage check accepting a flag just
// because some other command's section mentions its name.
func TestReadmeFlagScope(t *testing.T) {
	// A command that reuses flag names documented for other commands
	// (--force lives under "toki migrate"), so the check must not be fooled
	// by a global substring match.
	cmd := &cobra.Command{Use: "add"}
	cmd.Flags().String("force", "", "not really documented for add")
	cmd.Flags().String("priority", "", "documented for add")

	readme := strings.Join([]string{
		"```bash",
		"toki add <description> [flags]",
		"  --priority <level>",
		"",
		"toki migrate --to <backend> [flags]",
		"  --force",
		"```",
		"",
		"Prose that merely mentions `toki add --force` and `toki add --priority` must not",
		"count as documenting those flags for add.",
		"",
	}, "\n")

	missing := undocumentedFlags(readme, cmd, "add")

	if !slices.Contains(missing, "force") {
		t.Error("expected --force on add to be reported as undocumented when only migrate documents it")
	}
	if slices.Contains(missing, "priority") {
		t.Error("expected --priority on add to be recognized in add's documentation section")
	}
}

// TestReadmeFlagScopeIgnoresProse guards against prose that mentions a command
// and a flag on the same line being counted as documentation for that command.
func TestReadmeFlagScopeIgnoresProse(t *testing.T) {
	cmd := &cobra.Command{Use: "add"}
	cmd.Flags().String("tags", "", "only mentioned in prose")

	readme := strings.Join([]string{
		"```bash",
		"toki add <description> [flags]",
		"```",
		"",
		"Tags supplied via `toki add --tags` are stored verbatim.",
		"",
	}, "\n")

	missing := undocumentedFlags(readme, cmd, "add")

	if !slices.Contains(missing, "tags") {
		t.Error("expected --tags on add to be reported as undocumented when only prose mentions it")
	}
}
