package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raihankhan/kubes/internal/kube"
)

// greetingDismissMsg is sent when the user dismisses the greeting.
type greetingDismissMsg struct{}

// GreetingModel is the startup modal showing current context & namespace.
type GreetingModel struct {
	styles    Styles
	context   string
	namespace string
	cluster   string
	width     int
	height    int
}

func newGreetingModel(styles Styles) GreetingModel {
	ctx, _ := kube.GetCurrentContext()
	ns, _ := kube.GetCurrentNamespace()

	// Resolve cluster name for the active context.
	cluster := ""
	if ctx != "" {
		if ctxs, err := kube.GetInternalContexts(); err == nil {
			for _, c := range ctxs {
				if c.IsActive {
					cluster = c.Cluster
					break
				}
			}
		}
	}
	if ctx == "" {
		ctx = "(none)"
	}
	if ns == "" {
		ns = "default"
	}

	return GreetingModel{
		styles:    styles,
		context:   ctx,
		namespace: ns,
		cluster:   cluster,
	}
}

func (m GreetingModel) Init() tea.Cmd { return nil }

func (m GreetingModel) Update(msg tea.Msg) (GreetingModel, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		return m, func() tea.Msg { return greetingDismissMsg{} }
	}
	return m, nil
}

func (m GreetingModel) View() string {
	// ── Build card content ────────────────────────────────────────────────────
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.Primary)).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.Subtle))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.Text)).
		Bold(true)

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.Subtle)).
		Italic(true)

	activeBadge := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.BG)).
		Background(lipgloss.Color(m.styles.Theme.Active)).
		Bold(true).
		Padding(0, 1).
		Render("ACTIVE")

	logo := titleStyle.Render("󱃾  kubes")

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.Primary)).
		Bold(true)

	accentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.Secondary))

	divider := dimStyle.Render("  " + strings.Repeat("─", 38))

	step := func(n, k, desc string) string {
		return "  " + accentStyle.Render(n) + "  " + keyStyle.Render(k) + "  " + dimStyle.Render(desc)
	}

	rows := []string{
		logo,
		"",
		labelStyle.Render("  Context   ") + valueStyle.Render(m.context) + "  " + activeBadge,
	}
	if m.cluster != "" {
		rows = append(rows, labelStyle.Render("  Cluster   ")+valueStyle.Render(m.cluster))
	}
	rows = append(rows,
		labelStyle.Render("  Namespace ")+valueStyle.Render("󰋘 "+m.namespace),
		"",
		divider,
		"",
		dimStyle.Render("  Quick Start"),
		"",
		step("1", "↑ ↓", "navigate contexts"),
		step("2", "↵ ", "browse namespaces"),
		step("3", "↵ ", "set active namespace"),
		"  "+dimStyle.Render("or  ")+keyStyle.Render("s")+dimStyle.Render("  switch context directly"),
		"",
		dimStyle.Render("  Press any key to continue…"),
	)

	content := strings.Join(rows, "\n")

	// ── Bordered card ─────────────────────────────────────────────────────────
	cardWidth := 52
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.styles.Theme.Primary)).
		Padding(1, 3).
		Width(cardWidth).
		Render(content)

	// ── Centre on screen ──────────────────────────────────────────────────────
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

func (m *GreetingModel) setSize(w, h int) {
	m.width = w
	m.height = h
}

// ── helper: render a greeting line row ───────────────────────────────────────
func greetRow(label, value string, labelS, valueS lipgloss.Style) string {
	return fmt.Sprintf("%s%s", labelS.Render(label), valueS.Render(value))
}
