// ABOUTME: Interactive TUI wizard for configuring toki's storage backend.
// ABOUTME: 2-step bubbletea model collecting backend type and data directory.
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Step represents the current wizard step.
type Step int

const (
	StepBackend Step = iota
	StepDataDir
	StepDone
)

// SetupModel is the bubbletea model for the setup wizard.
type SetupModel struct {
	step     Step
	inputs   [2]textinput.Model
	quitting bool
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	brandStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	stepStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// defaultDataDir returns the default XDG data directory for toki.
func defaultDataDir() string {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "toki")
}

// NewSetupModel creates a new setup wizard model, pre-filling with existing config values.
func NewSetupModel(backend, dataDir string) SetupModel {
	backendInput := textinput.New()
	backendInput.Placeholder = "sqlite"
	backendInput.Focus()
	backendInput.Width = 50
	if backend != "" {
		backendInput.SetValue(backend)
	}

	dataDirInput := textinput.New()
	dataDirInput.Placeholder = defaultDataDir()
	dataDirInput.Width = 50
	if dataDir != "" {
		dataDirInput.SetValue(dataDir)
	}

	return SetupModel{
		step:   StepBackend,
		inputs: [2]textinput.Model{backendInput, dataDirInput},
	}
}

// Init implements tea.Model.
func (m SetupModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEscape {
			m.quitting = true
			return m, tea.Quit
		}

		if m.step == StepBackend || m.step == StepDataDir {
			return m.updateInput(msg)
		}
	default:
		// Forward other messages (e.g. cursor blink) to the active input
		if m.step == StepBackend || m.step == StepDataDir {
			idx := int(m.step)
			var cmd tea.Cmd
			m.inputs[idx], cmd = m.inputs[idx].Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m SetupModel) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type != tea.KeyEnter {
		// Forward to the active input
		idx := int(m.step)
		var cmd tea.Cmd
		m.inputs[idx], cmd = m.inputs[idx].Update(msg)
		return m, cmd
	}

	return m.handleEnter()
}

// handleEnter processes the Enter key for the current step.
func (m SetupModel) handleEnter() (tea.Model, tea.Cmd) {
	idx := int(m.step)

	if m.step == StepBackend {
		val := strings.TrimSpace(m.inputs[0].Value())
		if val == "" {
			val = "sqlite"
		}
		val = strings.ToLower(val)
		if val != "sqlite" && val != "markdown" {
			return m, nil
		}
		m.inputs[0].SetValue(val)
	}

	if m.step == StepDataDir {
		val := strings.TrimSpace(m.inputs[1].Value())
		if val == "" {
			m.inputs[1].SetValue(defaultDataDir())
		}
	}

	m.inputs[idx].Blur()

	switch m.step {
	case StepBackend:
		m.step = StepDataDir
		m.inputs[1].Focus()
		return m, textinput.Blink
	case StepDataDir:
		m.step = StepDone
		return m, tea.Quit
	}

	return m, nil
}

// View implements tea.Model.
func (m SetupModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(brandStyle.Render("   TURBO TASKZILLA"))
	b.WriteString(titleStyle.Render(" - Setup"))
	b.WriteString("\n\n")
	b.WriteString("Configure toki storage backend.\n\n")

	switch m.step {
	case StepBackend:
		b.WriteString(stepStyle.Render("Step 1 of 2: Storage Backend"))
		b.WriteString("\n")
		b.WriteString(promptStyle.Render("(sqlite or markdown, press Enter for default)"))
		b.WriteString("\n")
		b.WriteString(m.inputs[0].View())
		b.WriteString("\n")

	case StepDataDir:
		b.WriteString(fmt.Sprintf("  Backend: %s\n\n", m.inputs[0].Value()))
		b.WriteString(stepStyle.Render("Step 2 of 2: Data Directory"))
		b.WriteString("\n")
		b.WriteString(promptStyle.Render(fmt.Sprintf("(press Enter for default: %s)", defaultDataDir())))
		b.WriteString("\n")
		b.WriteString(m.inputs[1].View())
		b.WriteString("\n")

	case StepDone:
		b.WriteString(successStyle.Render("✓ Configuration saved!"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  Backend:        %s\n", m.inputs[0].Value()))
		b.WriteString(fmt.Sprintf("  Data directory:  %s\n", m.inputs[1].Value()))
		b.WriteString("\n")
	}

	return b.String()
}

// Result returns the entered values.
func (m SetupModel) Result() (backend, dataDir string) {
	return m.inputs[0].Value(), m.inputs[1].Value()
}

// ShouldSave returns true if the wizard completed and the user did not cancel.
func (m SetupModel) ShouldSave() bool {
	return m.step == StepDone && !m.quitting
}
