// ABOUTME: Unit tests for the toki setup TUI wizard bubbletea model.
// ABOUTME: Uses synthetic tea.Msg values to test state machine transitions.
package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewSetupModel_DefaultValues(t *testing.T) {
	m := NewSetupModel("", "")
	if m.step != StepBackend {
		t.Errorf("expected initial step StepBackend, got %d", m.step)
	}
	if m.inputs[0].Value() != "" {
		t.Error("expected empty backend input for new config")
	}
	if m.inputs[1].Value() != "" {
		t.Error("expected empty data dir input for new config")
	}
}

func TestNewSetupModel_ExistingConfig(t *testing.T) {
	m := NewSetupModel("markdown", "/custom/path")
	if m.inputs[0].Value() != "markdown" {
		t.Errorf("expected pre-filled backend, got %q", m.inputs[0].Value())
	}
	if m.inputs[1].Value() != "/custom/path" {
		t.Errorf("expected pre-filled data dir, got %q", m.inputs[1].Value())
	}
}

func TestSetupModel_StepTransitions(t *testing.T) {
	m := NewSetupModel("", "")

	// Enter on empty backend field — should use default "sqlite" and advance
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(SetupModel)
	if m.step != StepDataDir {
		t.Errorf("expected StepDataDir after Enter on backend, got %d", m.step)
	}
	if m.inputs[0].Value() != "sqlite" {
		t.Errorf("expected default backend 'sqlite', got %q", m.inputs[0].Value())
	}
	_ = cmd

	// Enter on empty data dir — should use default and finish
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(SetupModel)
	if m.step != StepDone {
		t.Errorf("expected StepDone after Enter on data dir, got %d", m.step)
	}
}

func TestSetupModel_InvalidBackend(t *testing.T) {
	m := NewSetupModel("", "")
	m.inputs[0].SetValue("invalid")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(SetupModel)
	if m.step != StepBackend {
		t.Errorf("expected to stay on StepBackend with invalid backend, got %d", m.step)
	}
}

func TestSetupModel_MarkdownBackend(t *testing.T) {
	m := NewSetupModel("", "")
	m.inputs[0].SetValue("markdown")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(SetupModel)
	if m.step != StepDataDir {
		t.Errorf("expected StepDataDir with valid backend, got %d", m.step)
	}
}

func TestSetupModel_BackendCaseInsensitive(t *testing.T) {
	m := NewSetupModel("", "")
	m.inputs[0].SetValue("SQLite")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(SetupModel)
	if m.inputs[0].Value() != "sqlite" {
		t.Errorf("expected lowercased backend, got %q", m.inputs[0].Value())
	}
	if m.step != StepDataDir {
		t.Errorf("expected StepDataDir, got %d", m.step)
	}
}

func TestSetupModel_QuitOnCtrlC(t *testing.T) {
	m := NewSetupModel("", "")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(SetupModel)
	if cmd == nil {
		t.Error("expected quit cmd on ctrl+c")
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
	if m.ShouldSave() {
		t.Error("expected ShouldSave false after ctrl+c")
	}
}

func TestSetupModel_QuitOnEsc(t *testing.T) {
	m := NewSetupModel("", "")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = updated.(SetupModel)
	if cmd == nil {
		t.Error("expected quit cmd on escape")
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestSetupModel_Result(t *testing.T) {
	m := NewSetupModel("", "")
	m.inputs[0].SetValue("sqlite")
	m.inputs[1].SetValue("/data/toki")
	m.step = StepDone

	backend, dataDir := m.Result()
	if backend != "sqlite" {
		t.Errorf("expected backend from result, got %q", backend)
	}
	if dataDir != "/data/toki" {
		t.Errorf("expected data dir from result, got %q", dataDir)
	}
}

func TestSetupModel_ShouldSave(t *testing.T) {
	t.Run("done means save", func(t *testing.T) {
		m := NewSetupModel("", "")
		m.step = StepDone
		if !m.ShouldSave() {
			t.Error("expected ShouldSave true when done")
		}
	})

	t.Run("quit means no save", func(t *testing.T) {
		m := NewSetupModel("", "")
		m.quitting = true
		if m.ShouldSave() {
			t.Error("expected ShouldSave false when quitting")
		}
	})
}

func TestSetupModel_ViewContainsBranding(t *testing.T) {
	m := NewSetupModel("", "")
	view := m.View()
	if !strings.Contains(view, "TURBO TASKZILLA") {
		t.Error("expected view to contain TURBO TASKZILLA branding")
	}
}

func TestSetupModel_ViewShowsCurrentStep(t *testing.T) {
	m := NewSetupModel("", "")

	m.step = StepBackend
	if !strings.Contains(m.View(), "Backend") {
		t.Error("expected StepBackend view to mention Backend")
	}

	m.step = StepDataDir
	if !strings.Contains(m.View(), "Data Directory") {
		t.Error("expected StepDataDir view to mention Data Directory")
	}
}

func TestSetupModel_ViewDone(t *testing.T) {
	m := NewSetupModel("", "")
	m.inputs[0].SetValue("sqlite")
	m.inputs[1].SetValue("/data/toki")
	m.step = StepDone
	view := m.View()
	if !strings.Contains(view, "saved") {
		t.Error("expected StepDone view to mention saved")
	}
}

func TestSetupModel_FullPrefilledFlow(t *testing.T) {
	m := NewSetupModel("sqlite", "/data/toki")

	// Enter on pre-filled backend
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = u.(SetupModel)
	if m.step != StepDataDir {
		t.Fatalf("expected StepDataDir, got %d", m.step)
	}

	// Enter on pre-filled data dir
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = u.(SetupModel)
	if m.step != StepDone {
		t.Fatalf("expected StepDone, got %d", m.step)
	}

	if !m.ShouldSave() {
		t.Error("expected ShouldSave true after completing flow")
	}
}
