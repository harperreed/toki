// ABOUTME: Export commands for Toki data
// ABOUTME: Exports data in YAML, JSON, Markdown, or SQLite formats

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/harper/toki/internal/storage"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ExportData represents the full export structure.
type ExportData struct {
	Version    string          `yaml:"version" json:"version"`
	ExportedAt string          `yaml:"exported_at" json:"exported_at"`
	Tool       string          `yaml:"tool" json:"tool"`
	Projects   []ExportProject `yaml:"projects" json:"projects"`
}

// ExportProject represents a project in the export.
type ExportProject struct {
	ID    string       `yaml:"id" json:"id"`
	Name  string       `yaml:"name" json:"name"`
	Path  string       `yaml:"path,omitempty" json:"path,omitempty"`
	Todos []ExportTodo `yaml:"todos" json:"todos"`
}

// ExportTodo represents a todo in the export.
type ExportTodo struct {
	ID          string   `yaml:"id" json:"id"`
	Description string   `yaml:"description" json:"description"`
	Done        bool     `yaml:"done" json:"done"`
	Priority    string   `yaml:"priority,omitempty" json:"priority,omitempty"`
	Notes       string   `yaml:"notes,omitempty" json:"notes,omitempty"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	CreatedAt   string   `yaml:"created_at" json:"created_at"`
	UpdatedAt   string   `yaml:"updated_at" json:"updated_at"`
	CompletedAt string   `yaml:"completed_at,omitempty" json:"completed_at,omitempty"`
	DueDate     string   `yaml:"due_date,omitempty" json:"due_date,omitempty"`
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export toki data",
	Long: `Export your toki data in various formats.

Formats:
  yaml     - YAML format (default, human-readable)
  json     - JSON format (machine-readable)
  markdown - Markdown checklist format
  sqlite   - Copy SQLite database file

Examples:
  toki export yaml > todos.yaml
  toki export json --pretty > todos.json
  toki export markdown > TODOS.md
  toki export sqlite --output backup.db`,
}

var exportYAMLCmd = &cobra.Command{
	Use:   "yaml",
	Short: "Export as YAML",
	RunE: func(cmd *cobra.Command, args []string) error {
		return exportToWriter(os.Stdout, "yaml", false)
	},
}

var exportJSONCmd = &cobra.Command{
	Use:   "json",
	Short: "Export as JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		pretty, _ := cmd.Flags().GetBool("pretty")
		return exportToWriter(os.Stdout, "json", pretty)
	},
}

var exportMarkdownCmd = &cobra.Command{
	Use:     "markdown",
	Aliases: []string{"md"},
	Short:   "Export as Markdown",
	RunE: func(cmd *cobra.Command, args []string) error {
		return exportToWriter(os.Stdout, "markdown", false)
	},
}

var exportSQLiteCmd = &cobra.Command{
	Use:   "sqlite",
	Short: "Export SQLite database",
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		if output == "" {
			output = fmt.Sprintf("toki-backup-%s.db", time.Now().Format("2006-01-02"))
		}
		return exportSQLite(output)
	},
}

func init() {
	exportJSONCmd.Flags().Bool("pretty", false, "pretty-print JSON output")
	exportSQLiteCmd.Flags().StringP("output", "o", "", "output file path")

	exportCmd.AddCommand(exportYAMLCmd)
	exportCmd.AddCommand(exportJSONCmd)
	exportCmd.AddCommand(exportMarkdownCmd)
	exportCmd.AddCommand(exportSQLiteCmd)

	rootCmd.AddCommand(exportCmd)
}

func getStorageForExport() (*storage.SQLiteStorage, error) {
	dbPath := storage.DefaultDBPath()
	return storage.NewSQLiteStorage(dbPath)
}

func exportToWriter(w io.Writer, format string, pretty bool) error {
	store, err := getStorageForExport()
	if err != nil {
		return fmt.Errorf("failed to open storage: %w", err)
	}
	defer func() { _ = store.Close() }()

	data, err := buildExportData(store)
	if err != nil {
		return err
	}

	switch format {
	case "yaml":
		return writeYAML(w, data)
	case "json":
		return writeJSON(w, data, pretty)
	case "markdown":
		writeMarkdown(w, data)
		return nil
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

func buildExportData(store *storage.SQLiteStorage) (*ExportData, error) {
	projects, err := store.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	data := &ExportData{
		Version:    "1.0",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Tool:       "toki",
		Projects:   make([]ExportProject, 0, len(projects)),
	}

	for _, project := range projects {
		exportProj := ExportProject{
			ID:    project.ID.String(),
			Name:  project.Name,
			Path:  project.DirectoryPath,
			Todos: []ExportTodo{},
		}

		// Get todos for this project
		todos, err := store.ListTodos(&storage.TodoFilter{ProjectID: &project.ID})
		if err != nil {
			return nil, fmt.Errorf("failed to list todos for project %s: %w", project.Name, err)
		}

		for _, todo := range todos {
			exportTodo := ExportTodo{
				ID:          todo.ID.String(),
				Description: todo.Description,
				Done:        todo.Done,
				Priority:    todo.Priority,
				Notes:       todo.Notes,
				Tags:        todo.Tags,
				CreatedAt:   todo.CreatedAt.Format(time.RFC3339),
				UpdatedAt:   todo.UpdatedAt.Format(time.RFC3339),
			}

			if todo.CompletedAt != nil {
				exportTodo.CompletedAt = todo.CompletedAt.Format(time.RFC3339)
			}
			if todo.DueDate != nil {
				exportTodo.DueDate = todo.DueDate.Format(time.RFC3339)
			}

			exportProj.Todos = append(exportProj.Todos, exportTodo)
		}

		data.Projects = append(data.Projects, exportProj)
	}

	return data, nil
}

func writeYAML(w io.Writer, data *ExportData) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	return encoder.Encode(data)
}

func writeJSON(w io.Writer, data *ExportData, pretty bool) error {
	encoder := json.NewEncoder(w)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(data)
}

func writeMarkdown(w io.Writer, data *ExportData) {
	// Header
	_, _ = fmt.Fprintf(w, "# Toki Export - %s\n\n", time.Now().Format("2006-01-02"))
	_, _ = fmt.Fprintf(w, "Generated: %s\n\n", data.ExportedAt)

	for _, project := range data.Projects {
		_, _ = fmt.Fprintf(w, "## Project: %s\n\n", project.Name)
		if project.Path != "" {
			_, _ = fmt.Fprintf(w, "_Path: %s_\n\n", project.Path)
		}

		// Separate pending and completed
		var pending, completed []ExportTodo
		for _, todo := range project.Todos {
			if todo.Done {
				completed = append(completed, todo)
			} else {
				pending = append(pending, todo)
			}
		}

		if len(pending) > 0 {
			_, _ = fmt.Fprintf(w, "### Pending Todos\n\n")
			for _, todo := range pending {
				writeTodoMarkdown(w, todo, false)
			}
			_, _ = fmt.Fprintln(w)
		}

		if len(completed) > 0 {
			_, _ = fmt.Fprintf(w, "### Completed Todos\n\n")
			for _, todo := range completed {
				writeTodoMarkdown(w, todo, true)
			}
			_, _ = fmt.Fprintln(w)
		}

		if len(project.Todos) == 0 {
			_, _ = fmt.Fprintf(w, "_No todos in this project_\n\n")
		}
	}
}

func writeTodoMarkdown(w io.Writer, todo ExportTodo, done bool) {
	checkbox := "[ ]"
	if done {
		checkbox = "[x]"
	}

	// Priority badge
	priority := ""
	if todo.Priority != "" {
		priority = fmt.Sprintf(" **[%s]**", strings.ToUpper(todo.Priority))
	}

	// Description (strikethrough if done)
	desc := todo.Description
	if done {
		desc = "~~" + desc + "~~"
	}

	_, _ = fmt.Fprintf(w, "- %s%s %s\n", checkbox, priority, desc)

	// Details
	if todo.DueDate != "" {
		dueTime, _ := time.Parse(time.RFC3339, todo.DueDate)
		_, _ = fmt.Fprintf(w, "  - Due: %s\n", dueTime.Format("2006-01-02"))
	}
	if len(todo.Tags) > 0 {
		_, _ = fmt.Fprintf(w, "  - Tags: %s\n", strings.Join(todo.Tags, ", "))
	}
	if todo.CompletedAt != "" {
		completedTime, _ := time.Parse(time.RFC3339, todo.CompletedAt)
		_, _ = fmt.Fprintf(w, "  - Completed: %s\n", completedTime.Format("2006-01-02"))
	}
	if todo.Notes != "" {
		_, _ = fmt.Fprintf(w, "  - Notes: %s\n", todo.Notes)
	}
}

func exportSQLite(output string) error {
	srcPath := storage.DefaultDBPath()

	// Check if source exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("no database found at %s", srcPath)
	}

	// Copy the file
	src, err := os.Open(srcPath) // #nosec G304 - user-provided export path is intentional
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(output) // #nosec G304 - user-provided export path is intentional
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy database: %w", err)
	}

	fmt.Printf("Exported database to %s\n", output)
	return nil
}
