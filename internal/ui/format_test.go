package ui

import (
	"strings"
	"testing"

	"github.com/harper/toki/internal/models"
)

func TestFormatTodo_ShowsDoneStatus(t *testing.T) {
	project := models.NewProject("test", nil)

	// Test pending todo
	pendingTodo := models.NewTodo(project.ID, "pending task")
	pendingOutput := FormatTodo(pendingTodo, nil)

	if strings.Contains(pendingOutput, "✓") {
		t.Error("Pending todo should not show checkmark")
	}

	// Test completed todo
	doneTodo := models.NewTodo(project.ID, "completed task")
	doneTodo.MarkDone()
	doneOutput := FormatTodo(doneTodo, nil)

	if !strings.Contains(doneOutput, "✓") {
		t.Error("Completed todo should show checkmark")
	}

	// Ensure they're visually different
	if pendingOutput == doneOutput {
		t.Error("Pending and completed todos should look different")
	}
}

func TestFormatTodo_DoneWithPriority(t *testing.T) {
	project := models.NewProject("test", nil)
	priority := "high"

	todo := models.NewTodo(project.ID, "important completed task")
	todo.Priority = &priority
	todo.MarkDone()

	output := FormatTodo(todo, nil)

	if !strings.Contains(output, "✓") {
		t.Error("Completed todo with priority should show checkmark")
	}
	if !strings.Contains(output, "HIGH") {
		t.Error("Completed todo should still show priority")
	}
}

func TestFormatTodo_DoneWithTags(t *testing.T) {
	project := models.NewProject("test", nil)

	todo := models.NewTodo(project.ID, "tagged completed task")
	todo.MarkDone()

	tags := []*models.Tag{
		{ID: 1, Name: "bug"},
		{ID: 2, Name: "urgent"},
	}

	output := FormatTodo(todo, tags)

	if !strings.Contains(output, "✓") {
		t.Error("Completed todo with tags should show checkmark")
	}
	if !strings.Contains(output, "bug") || !strings.Contains(output, "urgent") {
		t.Error("Completed todo should still show tags")
	}
}
