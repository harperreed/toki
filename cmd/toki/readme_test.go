// ABOUTME: Tests that README documents every implemented command and flag
// ABOUTME: Guards against README drifting out of sync with the CLI surface

package main

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// skipDocCommands are built-in cobra commands that do not need README coverage.
var skipDocCommands = map[string]bool{
	"help":       true,
	"completion": true,
}

// readmeContents returns the repository README.
func readmeContents(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}
	return string(data)
}

// walkCommands returns every non-hidden command path (e.g. "project add").
func walkCommands(cmd *cobra.Command, prefix string) []string {
	var paths []string
	for _, child := range cmd.Commands() {
		if child.Hidden || skipDocCommands[child.Name()] {
			continue
		}
		path := strings.TrimSpace(prefix + " " + child.Name())
		paths = append(paths, path)
		paths = append(paths, walkCommands(child, path)...)
	}
	return paths
}

// TestReadmeDocumentsAllCommands ensures each command path appears in the README
// as an invocable `toki <path>` example.
func TestReadmeDocumentsAllCommands(t *testing.T) {
	readme := readmeContents(t)

	commands := walkCommands(rootCmd, "")
	sort.Strings(commands)

	if len(commands) == 0 {
		t.Fatal("no commands discovered on root command")
	}

	for _, path := range commands {
		if !documentsCommand(readme, path) {
			t.Errorf("README does not document command %q (expected %q)", path, "toki "+path)
		}
	}
}

// flagNames returns every non-help flag applicable to cmd, including
// persistent and inherited flags (e.g. a root persistent flag inherited by a
// subcommand). LocalFlags covers local and local-persistent flags, while
// InheritedFlags covers persistent flags declared on parent commands.
func flagNames(cmd *cobra.Command) []string {
	var names []string
	seen := map[string]bool{}
	add := func(f *pflag.Flag) {
		if f.Name == "help" || seen[f.Name] {
			return
		}
		seen[f.Name] = true
		names = append(names, f.Name)
	}
	cmd.LocalFlags().VisitAll(add)
	cmd.InheritedFlags().VisitAll(add)
	sort.Strings(names)
	return names
}

// documentsFlag reports whether readme mentions the long flag --name at a token
// boundary, so a flag like "data" cannot falsely match an unrelated
// "--data-dir" mention.
func documentsFlag(readme, name string) bool {
	pattern := `(^|[^\w-])--` + regexp.QuoteMeta(name) + `([^\w-]|$)`
	return regexp.MustCompile(pattern).MatchString(readme)
}

// documentsCommand reports whether readme documents the invocable command
// phrase "toki <path>" at a token boundary, so a command name cannot be
// considered documented merely because it is a prefix of a longer documented
// invocation (e.g. "toki export" matching inside "toki export yaml").
func documentsCommand(readme, path string) bool {
	pattern := `(^|[^\w-])toki ` + regexp.QuoteMeta(path) + `([^\w-]|$)`
	return regexp.MustCompile(pattern).MatchString(readme)
}

// TestReadmeDocumentsAllFlags ensures every non-help flag on every command is
// mentioned somewhere in the README, either via its long (`--flag`) name.
func TestReadmeDocumentsAllFlags(t *testing.T) {
	readme := readmeContents(t)

	seen := map[string]bool{}
	var missing []string

	commands := append([]string{""}, walkCommands(rootCmd, "")...)
	for _, path := range commands {
		cmd := rootCmd
		if path != "" {
			var err error
			cmd, _, err = rootCmd.Find(strings.Fields(path))
			if err != nil {
				t.Fatalf("failed to find command %q: %v", path, err)
			}
		}

		for _, name := range flagNames(cmd) {
			if seen[name] {
				continue
			}
			seen[name] = true
			if !documentsFlag(readme, name) {
				missing = append(missing, "--"+name)
			}
		}
	}

	sort.Strings(missing)
	for _, flag := range missing {
		t.Errorf("README does not document flag %q", flag)
	}
}

// TestFlagNamesIncludesPersistentAndInherited verifies the guard enumerates
// persistent and inherited flags, not just local non-persistent ones, so a
// future root/command persistent flag cannot go undocumented unnoticed.
func TestFlagNamesIncludesPersistentAndInherited(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().String("global", "", "a root persistent flag")
	child := &cobra.Command{Use: "child"}
	child.Flags().String("local", "", "a local flag")
	child.PersistentFlags().String("child-persistent", "", "a child persistent flag")
	root.AddCommand(child)

	names := flagNames(child)
	want := map[string]bool{"global": true, "local": true, "child-persistent": true}
	got := map[string]bool{}
	for _, n := range names {
		got[n] = true
	}
	for name := range want {
		if !got[name] {
			t.Errorf("flagNames(child) missing %q; got %v", name, names)
		}
	}
}

// TestDocumentsFlagTokenBoundary verifies flags are matched at token
// boundaries, so a short flag name cannot pass on a substring coincidence.
func TestDocumentsFlagTokenBoundary(t *testing.T) {
	readme := "  --data-dir <dir>   # target directory\n"
	if documentsFlag(readme, "data") {
		t.Errorf("documentsFlag matched --data inside --data-dir")
	}
	if !documentsFlag("  --data <text>\n", "data") {
		t.Errorf("documentsFlag failed to match an exact --data mention")
	}
}

// TestDocumentsCommandTokenBoundary verifies command paths are matched at token
// boundaries, so a command name cannot pass on a coincidence where a longer
// token merely starts with it.
func TestDocumentsCommandTokenBoundary(t *testing.T) {
	if documentsCommand("  toki exportyaml\n", "export") {
		t.Errorf("documentsCommand matched \"export\" inside \"exportyaml\"")
	}
	if !documentsCommand("  toki export yaml\n", "export yaml") {
		t.Errorf("documentsCommand failed to match an exact \"toki export yaml\" mention")
	}
}
