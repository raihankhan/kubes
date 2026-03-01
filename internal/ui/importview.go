package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/raihankhan/kubes/internal/kube"
)

// ── Messages ─────────────────────────────────────────────────────────────────

// importDoneMsg is sent when an import succeeds or is cancelled.
type importDoneMsg struct {
	alias string
	err   error
}

// importCancelMsg is sent when user presses esc/ctrl+c during import.
type importCancelMsg struct{}

// ── Import steps ──────────────────────────────────────────────────────────────

type importStep int

const (
	importStepPath  importStep = iota // typing file path
	importStepAlias                   // typing alias
	importStepDone                    // submitting
)

// ── Model ────────────────────────────────────────────────────────────────────

// ImportModel is the interactive kubeconfig import form.
type ImportModel struct {
	styles       Styles
	step         importStep
	filepicker   filepicker.Model
	selectedPath string
	aliasIn      textinput.Model
	errMsg       string
	width        int
	height       int
}

func newImportModel(styles Styles) ImportModel {
	fp := filepicker.New()
	fp.AllowedTypes = []string{".yaml", ".yml", ""} // allow yaml configs and extensionless config files
	fp.CurrentDirectory, _ = os.UserHomeDir()
	fp.Height = 10

	aliasIn := textinput.New()
	aliasIn.Placeholder = "e.g. prod, staging, my-cluster"
	aliasIn.CharLimit = 64
	aliasIn.Width = 48
	aliasIn.Prompt = "  "
	aliasIn.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.Theme.Primary))
	aliasIn.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.Theme.Text))

	return ImportModel{
		styles:     styles,
		step:       importStepPath,
		filepicker: fp,
		aliasIn:    aliasIn,
	}
}

func (m ImportModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.filepicker.Init())
}

func (m ImportModel) Update(msg tea.Msg) (ImportModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c", "esc":
			return m, func() tea.Msg { return importCancelMsg{} }

		case "enter":
			switch m.step {
			case importStepAlias:
				alias := strings.TrimSpace(m.aliasIn.Value())
				if alias == "" {
					m.errMsg = "Alias cannot be empty."
					return m, nil
				}
				path := m.selectedPath
				m.step = importStepDone
				return m, func() tea.Msg {
					err := kube.ImportConfig(path, alias)
					return importDoneMsg{alias: alias, err: err}
				}
			}
		}
	}

	var cmd tea.Cmd
	switch m.step {
	case importStepPath:
		var fpCmd tea.Cmd
		m.filepicker, fpCmd = m.filepicker.Update(msg)

		if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
			m.selectedPath = path
			m.errMsg = ""
			m.step = importStepAlias
			m.aliasIn.Focus()
			return m, textinput.Blink
		}

		if didSelect, _ := m.filepicker.DidSelectDisabledFile(msg); didSelect {
			m.errMsg = "Selected file is not allowed. Needs to be a valid file."
		}
		cmd = fpCmd
	case importStepAlias:
		m.aliasIn, cmd = m.aliasIn.Update(msg)
	}
	return m, cmd
}

func (m ImportModel) View() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.Primary)).
		Bold(true)
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.Secondary)).
		Bold(true)
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.Subtle)).
		Italic(true)
	errStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.Error)).
		Bold(true)

	var rows []string

	rows = append(rows,
		titleStyle.Render("󰒋  Import External Kubeconfig"),
		"",
	)

	if m.step == importStepDone {
		rows = append(rows, dimStyle.Render("  Importing…"))
	} else {
		// Step 1 — path
		pathLabel := "  1. Select Kubeconfig"
		if m.step > importStepPath {
			pathLabel = dimStyle.Render("  ✓ " + m.selectedPath)
			rows = append(rows, labelStyle.Render("  Path"), pathLabel, "")
		} else {
			rows = append(rows, labelStyle.Render("  Step 1 — Browse and select Kubeconfig file"))
			rows = append(rows, m.filepicker.View())
			rows = append(rows, "")
			_ = pathLabel
		}

		// Step 2 — alias
		if m.step >= importStepAlias {
			rows = append(rows, labelStyle.Render("  Step 2 — Alias (filename saved to ~/.kubes/context/config/)"))
			rows = append(rows, m.aliasIn.View())
			rows = append(rows, "")
		}

		if m.errMsg != "" {
			rows = append(rows, errStyle.Render("  ⚠ "+m.errMsg))
			rows = append(rows, "")
		}

		hint := "enter = next   esc = cancel"
		if m.step == importStepAlias {
			hint = "enter = import   esc = cancel"
		}
		rows = append(rows, dimStyle.Render("  "+hint))
	}

	content := strings.Join(rows, "\n")

	cardWidth := 60
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.styles.Theme.Secondary)).
		Padding(1, 3).
		Width(cardWidth).
		Render(content)

	if m.width == 0 {
		return card
	}
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		card,
		lipgloss.WithWhitespaceBackground(lipgloss.Color(m.styles.Theme.BG)),
	)
}

func (m *ImportModel) setSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *ImportModel) applyStyles(styles Styles) {
	m.styles = styles
	m.aliasIn.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.Theme.Primary))
	m.aliasIn.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.Theme.Text))
}

// expandHome replaces leading ~ with the user's home directory.
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home := userHome()
		if home != "" {
			return home + path[1:]
		}
	}
	return path
}
