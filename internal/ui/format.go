// ABOUTME: Output formatting and color utilities
// ABOUTME: Formats todos and projects for display with colors

package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/harper/toki/internal/models"
)

var (
	boldCyan       = color.New(color.Bold, color.FgCyan)
	faint          = color.New(color.Faint)
	red            = color.New(color.FgRed)
	priorityHigh   = color.New(color.FgRed, color.Bold)
	priorityMedium = color.New(color.FgYellow)
	priorityLow    = color.New(color.Faint)
)

// FormatTodo formats a single todo for display.
func FormatTodo(todo *models.Todo, tags []*models.Tag) string {
	var builder strings.Builder

	// First line: ID + Done Status + Priority + Description
	builder.WriteString("  ")

	// Show checkmark for completed todos
	if todo.Done {
		builder.WriteString("✓ ")
	}

	builder.WriteString(faint.Sprint(todo.ID.String()[:6]))
	builder.WriteString("  ")

	if todo.Priority != nil {
		priority := strings.ToUpper(*todo.Priority)
		switch *todo.Priority {
		case "high":
			builder.WriteString(priorityHigh.Sprintf("[%s] ", priority))
		case "medium":
			builder.WriteString(priorityMedium.Sprintf("[%s] ", priority))
		case "low":
			builder.WriteString(priorityLow.Sprintf("[%s] ", priority))
		}
	}

	builder.WriteString(todo.Description)
	builder.WriteString("\n")

	// Second line: Metadata
	var metadata []string

	if todo.DueDate != nil {
		dueStr := todo.DueDate.Format("2006-01-02")
		// Compare dates only (not time) - truncate to start of day
		today := time.Now().Truncate(24 * time.Hour)
		dueDay := todo.DueDate.Truncate(24 * time.Hour)
		if dueDay.Before(today) {
			dueStr = red.Sprint(dueStr + " (overdue)")
		}
		metadata = append(metadata, "Due: "+dueStr)
	}

	if len(tags) > 0 {
		tagNames := make([]string, len(tags))
		for i, tag := range tags {
			tagNames[i] = tag.Name
		}
		metadata = append(metadata, "Tags: "+strings.Join(tagNames, ", "))
	}

	if len(metadata) > 0 {
		builder.WriteString("          ")
		builder.WriteString(faint.Sprint(strings.Join(metadata, " | ")))
		builder.WriteString("\n")
	}

	return builder.String()
}

// FormatProjectHeader formats a project header.
func FormatProjectHeader(project *models.Project) string {
	header := fmt.Sprintf("PROJECT: %s", boldCyan.Sprint(project.Name))
	if project.DirectoryPath != nil {
		header += faint.Sprintf(" (%s)", *project.DirectoryPath)
	}
	return header
}

// FormatSeparator creates a separator line.
func FormatSeparator() string {
	return faint.Sprint("─────────────────────────────────────────────")
}
