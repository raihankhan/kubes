package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raihankhan/kubes/internal/kube"
)

// ── Messages ─────────────────────────────────────────────────────────────────

type namespaceSelectedMsg struct {
	ctx       kube.Context
	namespace string
}

type namespacesLoadedMsg struct {
	namespaces []string
	err        error
}

// ── List item ────────────────────────────────────────────────────────────────

type nsItem struct {
	name     string
	isActive bool
}

func (i nsItem) FilterValue() string { return i.name }
func (i nsItem) Title() string       { return i.name }
func (i nsItem) Description() string { return "" }

// ── Item Delegate ─────────────────────────────────────────────────────────────

type nsDelegate struct {
	styles    Styles
	currentNS string
}

func (d nsDelegate) Height() int                             { return 1 }
func (d nsDelegate) Spacing() int                            { return 1 }
func (d nsDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d nsDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(nsItem)
	if !ok {
		return
	}

	width := m.Width()
	if width <= 0 {
		width = 60
	}

	isSelected := index == m.Index()

	// ── Selection marker ──────────────────────────────────────────────────────
	var marker string
	if isSelected {
		marker = lipgloss.NewStyle().
			Foreground(lipgloss.Color(d.styles.Theme.Primary)).
			Bold(true).
			Render("▌")
	} else {
		marker = lipgloss.NewStyle().
			Foreground(lipgloss.Color(d.styles.Theme.Subtle)).
			Render(" ")
	}

	// ── Icon ──────────────────────────────────────────────────────────────────
	var icon string
	if isSelected {
		icon = lipgloss.NewStyle().
			Foreground(lipgloss.Color(d.styles.Theme.Primary)).
			Bold(true).
			Render(" 󰋘 ")
	} else {
		icon = lipgloss.NewStyle().
			Foreground(lipgloss.Color(d.styles.Theme.Subtle)).
			Render(" 󰋘 ")
	}

	// ── Name ──────────────────────────────────────────────────────────────────
	var name string
	if isSelected {
		name = lipgloss.NewStyle().
			Foreground(lipgloss.Color(d.styles.Theme.Primary)).
			Bold(true).
			Render(item.name)
	} else {
		name = lipgloss.NewStyle().
			Foreground(lipgloss.Color(d.styles.Theme.Text)).
			Render(item.name)
	}

	// ── Active badge (right-aligned) ──────────────────────────────────────────
	badge := ""
	if item.isActive {
		badge = d.styles.ActiveBadge.Render("active")
	}

	left := marker + icon + name
	if badge == "" {
		fmt.Fprint(w, left)
		return
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(badge)
	if gap < 1 {
		gap = 1
	}
	fmt.Fprint(w, left+strings.Repeat(" ", gap)+badge)
}

// ── Model ────────────────────────────────────────────────────────────────────

// nsLoadState tracks the loading state for the namespace list.
type nsLoadState int

const (
	nsLoading nsLoadState = iota
	nsLoaded
	nsError
)

// NamespacesModel is the Bubbletea model for the namespace selection view.
type NamespacesModel struct {
	list        list.Model
	styles      Styles
	contextName string
	contextObj  kube.Context
	loadState   nsLoadState
	errMsg      string
	spinner     spinner.Model
	active      string
}

func newNamespacesModel(ctx kube.Context, styles Styles) NamespacesModel {
	l := list.New([]list.Item{}, nsDelegate{styles: styles}, 80, 30)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.FilterPrompt = styles.SearchBox
	l.Styles.FilterCursor = styles.ActiveItem

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styles.Spinner

	// Display name for the title bar.
	displayName := ctx.Name
	if ctx.IsExternal && strings.Contains(ctx.Name, "/") {
		parts := strings.SplitN(ctx.Name, "/", 2)
		displayName = parts[1]
	}

	return NamespacesModel{
		list:        l,
		styles:      styles,
		contextName: displayName,
		contextObj:  ctx,
		loadState:   nsLoading,
		spinner:     sp,
		active:      ctx.Namespace,
	}
}

func (m NamespacesModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchNamespaces(),
	)
}

func (m NamespacesModel) fetchNamespaces() tea.Cmd {
	ctx := m.contextObj
	return func() tea.Msg {
		var namespaces []string
		var err error
		namespaces, err = kube.GetNamespacesForContext(ctx.Name, ctx.ConfigPath)

		if len(namespaces) > 0 {
			// Find and extract "default"
			defaultIdx := -1
			for i, ns := range namespaces {
				if ns == "default" {
					defaultIdx = i
					break
				}
			}

			// Move "default" to the front if it exists
			if defaultIdx > 0 {
				defaultNS := namespaces[defaultIdx]
				namespaces = append(namespaces[:defaultIdx], namespaces[defaultIdx+1:]...)
				namespaces = append([]string{defaultNS}, namespaces...)
			}
		}

		return namespacesLoadedMsg{namespaces: namespaces, err: err}
	}
}

func (m NamespacesModel) Update(msg tea.Msg) (NamespacesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case namespacesLoadedMsg:
		if msg.err != nil {
			m.loadState = nsError
			m.errMsg = msg.err.Error()
			return m, nil
		}
		items := make([]list.Item, len(msg.namespaces))
		for i, ns := range msg.namespaces {
			items[i] = nsItem{name: ns, isActive: ns == m.active}
		}
		m.list.SetItems(items)
		m.loadState = nsLoaded
		return m, nil

	case spinner.TickMsg:
		if m.loadState == nsLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		if m.loadState != nsLoaded {
			return m, nil
		}
		switch msg.String() {
		case "enter":
			selected := m.list.SelectedItem()
			if selected == nil {
				return m, nil
			}
			item := selected.(nsItem)
			return m, func() tea.Msg {
				return namespaceSelectedMsg{
					ctx:       m.contextObj,
					namespace: item.name,
				}
			}
		case "p":
			selected := m.list.SelectedItem()
			if selected == nil {
				return m, nil
			}
			item := selected.(nsItem)
			ctx := m.contextObj
			ns := item.name
			return m, func() tea.Msg {
				return podViewMsg{ctx: ctx, namespace: ns}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m NamespacesModel) View() string {
	switch m.loadState {
	case nsLoading:
		return m.styles.DimText.PaddingLeft(2).Render(
			m.spinner.View() + " Fetching namespaces…",
		)
	case nsError:
		return m.styles.ErrorText.PaddingLeft(2).Render(
			"⚠ Could not reach cluster\n\n" + m.errMsg,
		)
	default:
		return m.list.View()
	}
}

func (m *NamespacesModel) setSize(w, h int) {
	m.list.SetWidth(w)
	m.list.SetHeight(h)
}

func (m *NamespacesModel) applyStyles(styles Styles) {
	m.styles = styles
	m.list.Styles.FilterPrompt = styles.SearchBox
	m.list.Styles.FilterCursor = styles.ActiveItem
	m.list.SetDelegate(nsDelegate{styles: styles, currentNS: m.active})
	m.spinner.Style = styles.Spinner
}
