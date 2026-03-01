package ui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raihankhan/kubes/internal/kube"
)

// viewState represents which view is currently shown.
type viewState int

const (
	stateGreeting   viewState = iota // startup modal
	stateContexts                    // context list
	stateNamespaces                  // namespace list
	stateImport                      // interactive import form
)

// AppModel is the root Bubbletea model managing the full TUI.
type AppModel struct {
	state        viewState
	styles       Styles
	greetingView GreetingModel
	contextView  ContextsModel
	nsView       NamespacesModel
	importView   ImportModel
	statusMsg    string
	envExport    string
	width        int
	height       int
	themeIdx     int
	statusIsErr  bool
	showHelp     bool
}

// New creates and initialises the root AppModel.
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

	// Determine initial envExport based on active context
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
		importView:   importModel,
		envExport:    envExport,
	}, nil
}

// Init is the Bubbletea Init hook.
func (m AppModel) Init() tea.Cmd {
	return m.greetingView.Init()
}

// Update handles all messages routed through the state machine.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.greetingView.setSize(msg.Width, msg.Height)
		m.contextView.setSize(msg.Width, m.contentHeight())
		m.nsView.setSize(msg.Width, m.contentHeight())
		m.importView.setSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		// Global hotkeys that work from any view (except greeting/import which
		// handle their own keys).
		if m.state == stateContexts || m.state == stateNamespaces {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit

			case "q":
				if m.state == stateNamespaces {
					m.state = stateContexts
					return m, nil
				}
				return m, tea.Quit

			case "esc":
				if m.state == stateNamespaces {
					m.state = stateContexts
					return m, nil
				}
				if m.state == stateContexts {
					// Re-show greeting when esc is pressed from context list.
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
				m.importView.applyStyles(m.styles)
				return m, nil

			case "?":
				m.showHelp = !m.showHelp
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

	// ── Greeting dismissed ────────────────────────────────────────────────────
	case greetingDismissMsg:
		m.state = stateContexts
		return m, nil

	// ── Context selected → make current and show namespace view ───────────────
	case contextSelectedMsg:
		var err error
		if msg.ctx.IsExternal {
			// For external contexts, we don't merge into ~/.kube/config anymore.
			// We just set the KUBECONFIG env var to the external file.
			m.envExport = fmt.Sprintf("export KUBECONFIG=\"%s\"", msg.ctx.ConfigPath)
		} else {
			err = kube.SetCurrentContext(msg.ctx.Name)
			if err == nil {
				// Clear KUBECONFIG to use default ~/.kube/config
				m.envExport = "unset KUBECONFIG"
			}
		}

		if err != nil {
			m.statusMsg = fmt.Sprintf("Error switching context: %s", err)
			m.statusIsErr = true
		} else {
			m.statusMsg = fmt.Sprintf("✓ switched to context '%s'", msg.ctx.Name)
			m.statusIsErr = false
		}

		// Refresh context items so the active badge moves to this context immediately
		m.refreshContexts()

		// Prepare namespace view
		nsModel := newNamespacesModel(msg.ctx, m.styles)
		m.nsView = nsModel
		m.state = stateNamespaces
		return m, nsModel.Init()

	// ── Namespace selected → update kubeconfig ────────────────────────────────
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
		return m, tea.Quit // Drop the user back into the shell immediately so they can run commands.

	// ── Context switched ───────────────────────────────────────────────────────
	case contextSwitchedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %s", msg.err)
			m.statusIsErr = true
		} else {
			m.statusMsg = fmt.Sprintf("✓ switched to context '%s'", msg.name)
			m.statusIsErr = false
			m.envExport = msg.envExport
		}
		m.refreshContexts()
		return m, nil

	// ── Import cancelled ───────────────────────────────────────────────────────
	case importCancelMsg:
		m.state = stateContexts
		return m, nil

	// ── Import done ────────────────────────────────────────────────────────────
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

	// Delegate to active sub-model.
	var cmd tea.Cmd
	switch m.state {
	case stateGreeting:
		m.greetingView, cmd = m.greetingView.Update(msg)
	case stateContexts:
		m.contextView, cmd = m.contextView.Update(msg)
	case stateNamespaces:
		m.nsView, cmd = m.nsView.Update(msg)
	case stateImport:
		m.importView, cmd = m.importView.Update(msg)
	}
	return m, cmd
}

// View renders the full TUI screen.
func (m AppModel) View() string {
	if m.width == 0 {
		return "Loading…"
	}

	// Greeting and Import are full-screen overlays.
	if m.state == stateGreeting {
		return m.greetingView.View()
	}
	if m.state == stateImport {
		return m.importView.View()
	}

	var b strings.Builder

	b.WriteString(m.renderHeader())
	b.WriteString("\n") // newline after header = 2 lines used so far

	if m.state == stateContexts || m.state == stateNamespaces {
		b.WriteString(m.renderMainLayout())
	} else {
		switch m.state {
		case stateContexts:
			b.WriteString(m.contextView.View())
		case stateNamespaces:
			b.WriteString(m.nsView.View())
		}
	}

	b.WriteString("\n") // newline after panels
	b.WriteString(m.renderStatusBar())

	if m.showHelp {
		b.WriteString("\n") // newline before help
		b.WriteString(m.renderHelp())
	}

	return b.String()
}

// ── Private helpers ──────────────────────────────────────────────────────────

func (m AppModel) renderHeader() string {
	icon := "󱃾 "
	title := m.styles.Title.Render(icon + "kubes")

	themeLabel := m.styles.DimText.Render(
		fmt.Sprintf(" [%s]", m.styles.Theme.Name),
	)

	viewLabel := ""
	switch m.state {
	case stateContexts:
		viewLabel = m.styles.Subtitle.Render("  Contexts")
	case stateNamespaces:
		viewLabel = m.styles.Subtitle.Render("  Namespaces » " + m.nsView.contextName)
	}

	right := lipgloss.JoinHorizontal(lipgloss.Top, themeLabel, "  ", viewLabel)
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return title + strings.Repeat(" ", gap) + right
}

func (m AppModel) renderStatusBar() string {
	if m.statusMsg == "" {
		return m.styles.HelpBar.Render("?=help  t=theme  i=import  q=quit")
	}
	if m.statusIsErr {
		return m.styles.ErrorText.Render("  " + m.statusMsg)
	}
	return m.styles.DimText.Render("  " + m.statusMsg)
}

func (m AppModel) renderHelp() string {
	entries := []string{
		"↑/↓ k/j  navigate",
		"enter    select",
		"s        switch ctx",
		"i        import config",
		"t        next theme",
		"esc      back/greeting",
		"q        quit",
		"?        close help",
	}
	return m.styles.HelpBar.Render("  " + strings.Join(entries, "   "))
}

func (m AppModel) contentHeight() int {
	// Header(1) + newline(1) + newline(1) + status(1) = 4 lines
	reserved := 4
	if m.showHelp {
		reserved += 2
	}
	h := m.height - reserved
	if h < 5 {
		h = 5
	}
	return h
}

func (m AppModel) renderMainLayout() string {
	paneHeight := m.contentHeight()

	// Title / Banner at the top
	bannerText := `
  _  __       _                  
 | |/ /      | |                 
 | ' / _   _ | |__    ___  ___  
 |  < | | | || '_ \  / _ \/ __| 
 | . \| |_| || |_) ||  __/\__ \ 
 |_|\_\\__,_||_.__/  \___||___/ `

	banner := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.styles.Theme.Primary)).
		Render(bannerText)
	desc := m.styles.Subtitle.Render("  The Modern Kubernetes TUI")

	// Calculate remaining height for lists
	bannerHeight := lipgloss.Height(banner) + 1 // +1 for the description
	if bannerHeight > paneHeight {
		bannerHeight = paneHeight
	}

	listAreaHeight := paneHeight - bannerHeight - 1 // -1 for padding gap
	if listAreaHeight < 5 {
		listAreaHeight = 5 // minimum fallback
	}

	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth - 1 // 1 char space between panes

	listWidthInnerLeft := leftWidth - 2
	if listWidthInnerLeft < 1 {
		listWidthInnerLeft = 1
	}

	rightWidthInner := rightWidth - 2
	if rightWidthInner < 1 {
		rightWidthInner = 1
	}

	// ── Left Pane (Contexts) ──
	m.contextView.setSize(listWidthInnerLeft, listAreaHeight-2)
	leftBorder := m.styles.PaneBorderActive
	if m.state != stateContexts {
		leftBorder = m.styles.PaneBorderInactive
	}
	leftPane := leftBorder.
		Width(listWidthInnerLeft).
		Height(listAreaHeight - 2).
		Render(m.contextView.View())

	// ── Right Pane (Namespaces) ──
	rightBorder := m.styles.PaneBorderInactive
	var rightContent string

	if m.state == stateContexts {
		ctx := m.contextView.SelectedContext()
		if ctx != nil {
			rightContent = m.styles.DimText.Render(fmt.Sprintf("\n  Selected Context: %s\n  Current Namespace: %s\n\n  Press Enter to load and switch namespaces.", ctx.Name, ctx.Namespace))
		} else {
			rightContent = m.styles.DimText.Render("\n  No context selected.")
		}
	} else if m.state == stateNamespaces {
		rightBorder = m.styles.PaneBorderActive
		m.nsView.setSize(rightWidthInner, listAreaHeight-2)
		rightContent = m.nsView.View()
	}

	rightPane := rightBorder.
		Width(rightWidthInner).
		Height(listAreaHeight - 2).
		Render(rightContent)

	listsRow := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)

	return lipgloss.JoinVertical(lipgloss.Left, banner, desc, "", listsRow)
}

func (m *AppModel) refreshContexts() {
	internalCtxs, _ := kube.GetInternalContexts()
	externalCtxs, _ := kube.GetExternalContexts()
	all := append(internalCtxs, externalCtxs...)
	m.contextView.refreshItems(all, m.styles)
}

func userHome() string {
	home, _ := os.UserHomeDir()
	return home
}

// GetEnvExport returns the constructed bash string for setting KUBECONFIG
func (m AppModel) GetEnvExport() string {
	return m.envExport
}
