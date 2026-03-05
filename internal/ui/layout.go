package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raihankhan/kubes/internal/kube"
)

// viewState represents which view is currently shown.
type viewState int

const (
	stateGreeting   viewState = iota
	stateContexts
	stateNamespaces
	statePods
	stateImport
)

// AppModel is the root Bubbletea model managing the full TUI.
type AppModel struct {
	state        viewState
	styles       Styles
	greetingView GreetingModel
	contextView  ContextsModel
	nsView       NamespacesModel
	podsView     PodsModel
	importView   ImportModel
	statusMsg    string
	envExport    string
	width        int
	height       int
	themeIdx     int
	statusIsErr  bool
	showHelp     bool
}

// ── Shell exec messages ───────────────────────────────────────────────────────

type shellExecMsg struct{ ctx kube.Context }
type shellExitMsg struct{ err error }

func New() (AppModel, error) {
	styles := NewStyles(Themes[0])

	internalCtxs, err := kube.GetInternalContexts()
	if err != nil {
		internalCtxs = []kube.Context{}
	}
	externalCtxs, err := kube.GetExternalContexts()
	if err != nil {
		externalCtxs = []kube.Context{}
	}

	all := append(internalCtxs, externalCtxs...)
	ctxModel := newContextsModel(all, styles)
	greeting := newGreetingModel(styles)
	importModel := newImportModel(styles)

	activePath := kube.DefaultKubeConfigPath()
	envExport := ""
	if activePath != kube.StandardKubeConfigPath() {
		envExport = fmt.Sprintf("export KUBECONFIG=\"%s\"", activePath)
	}

	return AppModel{
		state:        stateGreeting,
		themeIdx:     0,
		styles:       styles,
		greetingView: greeting,
		contextView:  ctxModel,
		nsView:       newNamespacesModel(kube.Context{}, styles),
		podsView:     newPodsModel(kube.Context{}, "", styles),
		importView:   importModel,
		envExport:    envExport,
	}, nil
}

func (m AppModel) Init() tea.Cmd {
	return m.greetingView.Init()
}

// ── resize ────────────────────────────────────────────────────────────────────
// resize recomputes and sets the correct inner dimensions for both list panes.
// Must be called from Update() (pointer receiver context) whenever m.width,
// m.height, or m.showHelp changes.
func (m *AppModel) resize() {
	innerH := m.contentHeight() - 2
	if innerH < 1 {
		innerH = 1
	}
	leftW := m.width / 2
	innerLeftW := leftW - 2
	if innerLeftW < 1 {
		innerLeftW = 1
	}
	rightW := m.width - leftW - 1
	innerRightW := rightW - 2
	if innerRightW < 1 {
		innerRightW = 1
	}
	m.contextView.setSize(innerLeftW, innerH)
	m.nsView.setSize(innerRightW, innerH)
	m.podsView.setSize(innerRightW, innerH)
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.greetingView.setSize(msg.Width, msg.Height)
		m.importView.setSize(msg.Width, msg.Height)
		m.resize() // set correct pane sizes immediately
		return m, nil

	case tea.KeyMsg:
		if m.state == stateContexts || m.state == stateNamespaces || m.state == statePods {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit

			case "q":
				if m.state == statePods {
					m.state = stateNamespaces
					return m, nil
				}
				if m.state == stateNamespaces {
					m.state = stateContexts
					return m, nil
				}
				return m, tea.Quit

			case "esc":
				if m.state == statePods {
					m.state = stateNamespaces
					return m, nil
				}
				if m.state == stateNamespaces {
					m.state = stateContexts
					return m, nil
				}
				if m.state == stateContexts {
					m.greetingView = newGreetingModel(m.styles)
					m.greetingView.setSize(m.width, m.height)
					m.state = stateGreeting
					return m, nil
				}

			case "t":
				m.themeIdx = (m.themeIdx + 1) % len(Themes)
				m.styles = NewStyles(Themes[m.themeIdx])
				m.contextView.applyStyles(m.styles)
				m.nsView.applyStyles(m.styles)
				m.podsView.applyStyles(m.styles)
				m.importView.applyStyles(m.styles)
				return m, nil

			case "?":
				m.showHelp = !m.showHelp
				m.resize()
				return m, nil

			case "i":
				if m.state == stateContexts {
					im := newImportModel(m.styles)
					im.setSize(m.width, m.height)
					m.importView = im
					m.state = stateImport
					return m, im.Init()
				}
			}
		}

	case greetingDismissMsg:
		m.state = stateContexts
		return m, nil

	case contextSelectedMsg:
		var err error
		if msg.ctx.IsExternal {
			m.envExport = fmt.Sprintf("export KUBECONFIG=\"%s\"", msg.ctx.ConfigPath)
		} else {
			err = kube.SetCurrentContext(msg.ctx.Name)
			if err == nil {
				m.envExport = "unset KUBECONFIG"
			}
		}
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error switching context: %s", err)
			m.statusIsErr = true
		} else {
			m.statusMsg = fmt.Sprintf("✓ switched to context '%s'", msg.ctx.Name)
			m.statusIsErr = false
			kube.AddRecentContext(msg.ctx.Name)
		}
		m.refreshContexts()
		nsModel := newNamespacesModel(msg.ctx, m.styles)
		m.nsView = nsModel
		m.resize()
		m.state = stateNamespaces
		return m, nsModel.Init()

	case namespaceSelectedMsg:
		err := kube.SetContextNamespace(msg.ctx.ConfigPath, msg.ctx.InternalName, msg.namespace)
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error: %s", err)
			m.statusIsErr = true
		} else {
			m.statusMsg = fmt.Sprintf("✓ namespace '%s' set on context '%s'", msg.namespace, msg.ctx.Name)
			m.statusIsErr = false
		}
		m.state = stateContexts
		m.refreshContexts()
		return m, tea.Quit

	case contextSwitchedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %s", msg.err)
			m.statusIsErr = true
		} else {
			m.statusMsg = fmt.Sprintf("✓ switched to context '%s'", msg.name)
			m.statusIsErr = false
			m.envExport = msg.envExport
			kube.AddRecentContext(msg.name)
		}
		m.refreshContexts()
		return m, nil

	case podViewMsg:
		podsModel := newPodsModel(msg.ctx, msg.namespace, m.styles)
		m.podsView = podsModel
		m.resize()
		m.state = statePods
		return m, podsModel.Init()

	case shellExecMsg:
		return m, m.launchShell(msg.ctx)

	case shellExitMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Shell exited with error: %s", msg.err)
			m.statusIsErr = true
		} else {
			m.statusMsg = "Returned from shell session."
			m.statusIsErr = false
		}
		m.refreshContexts()
		return m, nil

	case importCancelMsg:
		m.state = stateContexts
		return m, nil

	case importDoneMsg:
		m.state = stateContexts
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Import failed: %s", msg.err)
			m.statusIsErr = true
		} else {
			m.statusMsg = fmt.Sprintf("✓ imported as '%s' — visible under External Contexts", msg.alias)
			m.statusIsErr = false
			m.refreshContexts()
		}
		return m, nil
	}

	var cmd tea.Cmd
	switch m.state {
	case stateGreeting:
		m.greetingView, cmd = m.greetingView.Update(msg)
	case stateContexts:
		m.contextView, cmd = m.contextView.Update(msg)
	case stateNamespaces:
		m.nsView, cmd = m.nsView.Update(msg)
	case statePods:
		m.podsView, cmd = m.podsView.Update(msg)
	case stateImport:
		m.importView, cmd = m.importView.Update(msg)
	}
	return m, cmd
}

// ── View ──────────────────────────────────────────────────────────────────────
//
// Screen layout (top → bottom):
//
//	[banner]      6 lines  — only when m.height >= minBannerHeight
//	subtitle      1 line   — descriptor + fill rule + theme + view label
//	panes         N lines  — left: contexts  |  right: namespaces or info
//	status        1 line
//	[help]        1 line   — only when showHelp
//
// All heights are computed exactly so content never overflows the terminal.

func (m AppModel) View() string {
	if m.width == 0 {
		return "Loading…"
	}
	if m.state == stateGreeting {
		return m.greetingView.View()
	}
	if m.state == stateImport {
		return m.importView.View()
	}

	var b strings.Builder

	// Banner is always at the top — 6 lines, no trailing \n.
	b.WriteString(m.renderBanner())
	b.WriteString("\n")
	b.WriteString(m.renderSubtitle()) // 1 line
	b.WriteString("\n")
	b.WriteString(m.renderPanes()) // contentHeight() lines
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar()) // 1 line

	if m.showHelp {
		b.WriteString("\n")
		b.WriteString(m.renderHelp())
	}

	return b.String()
}

// ── Height accounting ─────────────────────────────────────────────────────────
//
// Fixed lines above/below panes (always):
//   banner(6) + \n(1) + subtitle(1) + \n(1) + \n(1) + status(1) = 11
//   + help(\n + 1 line = 2) when showHelp

func (m AppModel) contentHeight() int {
	reserved := 11 // banner(6) + newline(1) + subtitle(1) + newline(1) + newline(1) + status(1)
	if m.showHelp {
		reserved += 2
	}
	h := m.height - reserved
	if h < 2 {
		h = 2
	}
	return h
}

// ── Banner ────────────────────────────────────────────────────────────────────

// bannerText is the original "kubes" lean ASCII art (6 lines, no leading newline).
// Backslashes are real backslashes inside this raw string literal.
const bannerText = `  _  __       _
 | |/ /      | |
 | ' / _   _ | |__    ___  ___
 |  < | | | || '_ \  / _ \/ __|
 | . \| |_| || |_) ||  __/\__ \
 |_|\_\\__,_||_.__/  \___||___/ `

func (m AppModel) renderBanner() string {
	lines := strings.Split(bannerText, "\n")
	grad := bannerGradient(m.styles.Theme)

	rendered := make([]string, len(lines))
	for i, line := range lines {
		idx := i
		if idx >= len(grad) {
			idx = len(grad) - 1
		}
		rendered[i] = lipgloss.NewStyle().
			Foreground(grad[idx]).
			Bold(true).
			Render(line)
	}
	return strings.Join(rendered, "\n")
}

// renderSubtitle returns the one-line row beneath the banner.
// It stretches a dim fill rule across the full terminal width, with
// the theme name and current view label right-aligned.
func (m AppModel) renderSubtitle() string {
	gem := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.Primary)).
		Bold(true).
		Render(" ◆")
	desc := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.Subtle)).
		Render("  kubernetes context & namespace manager")

	themeTag := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.Subtle)).
		Render("[" + m.styles.Theme.Name + "]")

	viewTag := ""
	sec := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.Secondary)).Bold(true)
	switch m.state {
	case stateContexts:
		viewTag = sec.Render("  Contexts")
	case stateNamespaces:
		viewTag = sec.Render("  Namespaces  ›  " + m.nsView.contextName)
	case statePods:
		viewTag = sec.Render("  Pods  ›  " + m.nsView.contextName + "  ›  " + m.podsView.namespace)
	}

	left := gem + desc
	right := themeTag + viewTag

	fillW := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if fillW < 1 {
		// Terminal too narrow to fit both sides — drop the right side.
		return left
	}
	fill := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.Surface)).
		Render(" " + strings.Repeat("─", fillW-1))

	return left + fill + right
}

// bannerGradient returns a 6-stop top-lit color ramp for the given theme.
// Colors flow: bright highlight (top) → primary → deep shadow (bottom).
func bannerGradient(t Theme) []lipgloss.Color {
	switch t.Name {
	case "Kubes":
		return []lipgloss.Color{"#FFE0A0", "#FFD060", "#FFAB40", "#FF9000", "#FF8700", "#E65100"}
	case "Catppuccin":
		return []lipgloss.Color{"#f5c2e7", "#e0bfff", "#cba6f7", "#c6a0f6", "#b4befe", "#89b4fa"}
	case "Nord":
		return []lipgloss.Color{"#ECEFF4", "#D8DEE9", "#88C0D0", "#81A1C1", "#5E81AC", "#4C566A"}
	case "Dracula":
		return []lipgloss.Color{"#ffffff", "#ffb8d1", "#ff79c6", "#ff79c6", "#bd93f9", "#6272a4"}
	case "Gruvbox Dark":
		return []lipgloss.Color{"#FFF8E1", "#FFECB3", "#FABD2F", "#F9A825", "#FF8F00", "#E65100"}
	default:
		return []lipgloss.Color{
			lipgloss.Color(t.Text), lipgloss.Color(t.Text),
			lipgloss.Color(t.Primary), lipgloss.Color(t.Primary),
			lipgloss.Color(t.Secondary), lipgloss.Color(t.Secondary),
		}
	}
}

// ── Two-column pane layout ────────────────────────────────────────────────────

// renderPanes renders the two side-by-side bordered columns.
// The list sizes have already been set correctly via resize() in Update(),
// so no setSize calls are made here (View has a value receiver and any
// mutations would be ephemeral and discard the caller's state).
func (m AppModel) renderPanes() string {
	paneH := m.contentHeight()
	innerH := paneH - 2 // top + bottom border
	if innerH < 1 {
		innerH = 1
	}

	leftW := m.width / 2
	rightW := m.width - leftW - 1
	innerLeftW := leftW - 2
	if innerLeftW < 1 {
		innerLeftW = 1
	}
	innerRightW := rightW - 2
	if innerRightW < 1 {
		innerRightW = 1
	}

	// ── Left pane: Contexts ───────────────────────────────────────────────────
	leftBorder := m.styles.PaneBorderActive
	if m.state != stateContexts {
		leftBorder = m.styles.PaneBorderInactive
	}
	leftPane := leftBorder.
		Width(innerLeftW).
		Height(innerH).
		Render(m.contextView.View())

	// ── Right pane: Namespaces or context info ────────────────────────────────
	rightBorder := m.styles.PaneBorderInactive
	var rightContent string

	switch m.state {
	case stateContexts:
		ctx := m.contextView.SelectedContext()
		if ctx != nil {
			rightContent = m.renderContextInfoPanel(ctx, innerRightW)
		} else {
			rightContent = m.renderGettingStartedPanel()
		}
	case stateNamespaces:
		rightBorder = m.styles.PaneBorderActive
		rightContent = m.nsView.View()
	case statePods:
		rightBorder = m.styles.PaneBorderActive
		rightContent = m.podsView.View()
	}

	rightPane := rightBorder.
		Width(innerRightW).
		Height(innerH).
		Render(rightContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)
}

// ── Status & help ─────────────────────────────────────────────────────────────

func (m AppModel) renderStatusBar() string {
	if m.statusMsg != "" {
		if m.statusIsErr {
			return m.styles.ErrorText.Render("  " + m.statusMsg)
		}
		return m.styles.DimText.Render("  " + m.statusMsg)
	}

	key := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.Primary)).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.Subtle))

	var hints []string
	switch m.state {
	case stateContexts:
		hints = []string{
			key.Render("↑↓") + dim.Render(" navigate"),
			key.Render("↵") + dim.Render(" namespaces"),
			key.Render("s") + dim.Render(" switch"),
			key.Render("x") + dim.Render(" shell"),
			key.Render("i") + dim.Render(" import"),
			key.Render("t") + dim.Render(" theme"),
			key.Render("?") + dim.Render(" help"),
			key.Render("q") + dim.Render(" quit"),
		}
	case stateNamespaces:
		hints = []string{
			key.Render("↑↓") + dim.Render(" navigate"),
			key.Render("↵") + dim.Render(" set namespace"),
			key.Render("p") + dim.Render(" pods"),
			key.Render("esc") + dim.Render(" back"),
			key.Render("q") + dim.Render(" quit"),
		}
	case statePods:
		hints = []string{
			key.Render("↑↓") + dim.Render(" navigate"),
			key.Render("/") + dim.Render(" filter"),
			key.Render("esc") + dim.Render(" back"),
			key.Render("q") + dim.Render(" quit"),
		}
	default:
		return m.styles.HelpBar.Render("  ?=help  t=theme  i=import  q=quit")
	}

	sep := dim.Render("  ·  ")
	return "  " + strings.Join(hints, sep)
}

func (m AppModel) renderHelp() string {
	entries := []string{
		"↑/↓ k/j  navigate",
		"enter    select/drill-down",
		"s        switch ctx",
		"x        open shell",
		"p        view pods",
		"i        import config",
		"t        next theme",
		"esc      back",
		"q        quit",
		"?        close help",
	}
	return m.styles.HelpBar.Render("  " + strings.Join(entries, "   "))
}

// ── Context info panel (right pane in stateContexts) ─────────────────────────

func (m AppModel) renderContextInfoPanel(ctx *kube.Context, width int) string {
	t := m.styles.Theme

	// Accent: primary for internal, secondary for external.
	accentColor := lipgloss.Color(t.Primary)
	if ctx.IsExternal {
		accentColor = lipgloss.Color(t.Secondary)
	}

	displayName := ctx.Name
	if ctx.IsExternal && strings.Contains(ctx.Name, "/") {
		parts := strings.SplitN(ctx.Name, "/", 2)
		displayName = parts[1]
	}

	// ── Name ──────────────────────────────────────────────────────────────────
	nameRow := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Render("  " + displayName)

	// ── Active badge inline with name ─────────────────────────────────────────
	if ctx.IsActive {
		badge := lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.BG)).
			Background(lipgloss.Color(t.Active)).
			Bold(true).
			Padding(0, 1).
			Render("active")
		nameRow = nameRow + "  " + badge
	}

	// ── Info rows: namespace ──────────────────────────────────────────────────
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Subtle))
	nsValueStyle := lipgloss.NewStyle().Foreground(accentColor).Bold(true)

	nsRow := "  " + labelStyle.Render("Namespace ") + nsValueStyle.Render(ctx.Namespace)

	// ── Divider ───────────────────────────────────────────────────────────────
	ruleLen := width - 4
	if ruleLen < 1 {
		ruleLen = 1
	}
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Subtle)).
		Render("  " + strings.Repeat("╌", ruleLen))

	// ── Hints ─────────────────────────────────────────────────────────────────
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Subtle))
	keyStyle := lipgloss.NewStyle().Foreground(accentColor).Bold(true)

	hintEnter := keyStyle.Render("↵") + hint.Render("  browse namespaces")
	hintS := keyStyle.Render("s") + hint.Render("  switch context directly")

	rows := []string{
		"",
		nameRow,
		"",
		nsRow,
		"",
		divider,
		"",
		"  " + hintEnter,
		"  " + hintS,
	}
	return strings.Join(rows, "\n")
}

// ── Getting started panel (right pane when no context is highlighted) ─────────

func (m AppModel) renderGettingStartedPanel() string {
	t := m.styles.Theme
	title := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true)
	key := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Subtle))
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Secondary))

	step := func(n, k, desc string) string {
		return "  " + accent.Render(n) + "  " + key.Render(k) + "  " + dim.Render(desc)
	}

	rows := []string{
		"",
		"  " + title.Render("Quick Start"),
		"",
		step("1", "↑ ↓", "navigate contexts"),
		step("2", "↵", "browse namespaces"),
		step("3", "↵", "set active namespace"),
		"",
		"  " + dim.Render("— or —"),
		"",
		step("", "s", "switch context directly"),
		step("", "i", "import external kubeconfig"),
		"",
		"  " + dim.Render("Press ") + key.Render("?") + dim.Render(" for full keybindings"),
	}
	return strings.Join(rows, "\n")
}

// ── Shell passthrough ─────────────────────────────────────────────────────────

func (m *AppModel) launchShell(ctx kube.Context) tea.Cmd {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	c := exec.Command(shell)
	env := os.Environ()

	if ctx.IsExternal {
		// Point KUBECONFIG at the external file so kubectl uses it directly.
		env = filterEnv(env, "KUBECONFIG")
		env = append(env, "KUBECONFIG="+ctx.ConfigPath)
	} else {
		// Ensure the target context is active in the standard kubeconfig.
		_ = kube.SetCurrentContext(ctx.Name)
		env = filterEnv(env, "KUBECONFIG")
	}

	// Expose the context name so users can reference it in PS1 / scripts.
	env = filterEnv(env, "KUBES_CONTEXT")
	env = append(env, "KUBES_CONTEXT="+ctx.Name)

	c.Env = env
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return shellExitMsg{err: err}
	})
}

// filterEnv returns a copy of env with all entries for key removed.
func filterEnv(env []string, key string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			out = append(out, e)
		}
	}
	return out
}

// ── Misc ──────────────────────────────────────────────────────────────────────

func (m *AppModel) refreshContexts() {
	internalCtxs, _ := kube.GetInternalContexts()
	externalCtxs, _ := kube.GetExternalContexts()
	all := append(internalCtxs, externalCtxs...)
	m.contextView.refreshItems(all, m.styles)
}

func (m AppModel) GetEnvExport() string {
	return m.envExport
}
