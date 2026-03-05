package ui

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raihankhan/kubes/internal/kube"
)

// ── Messages ──────────────────────────────────────────────────────────────────

type contextSelectedMsg struct{ ctx kube.Context }

type contextSwitchedMsg struct {
	name      string
	err       error
	envExport string
}

// ── List item ─────────────────────────────────────────────────────────────────

type contextItem struct {
	ctx             kube.Context
	isHeader        bool
	headerName      string
	isExternalGroup bool // true when this header belongs to the external group
	isLast          bool
}

func (i contextItem) FilterValue() string {
	if i.isHeader {
		return ""
	}
	return i.ctx.Name
}

func (i contextItem) Title() string       { return i.ctx.Name }
func (i contextItem) Description() string { return "" }

// ── Item Delegate ─────────────────────────────────────────────────────────────

type contextDelegate struct {
	styles Styles
}

func (d contextDelegate) Height() int                             { return 2 }
func (d contextDelegate) Spacing() int                            { return 1 }
func (d contextDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d contextDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(contextItem)
	if !ok {
		return
	}

	width := m.Width()
	if width <= 0 {
		width = 60
	}

	// ── Section headers ───────────────────────────────────────────────────────
	if item.isHeader {
		if item.headerName == "" {
			fmt.Fprint(w, "\n") // blank spacer (2 lines: this + the spacing row)
			return
		}
		// Color: primary for internal, secondary for external.
		var hdrColor lipgloss.Color
		if item.isExternalGroup {
			hdrColor = lipgloss.Color(d.styles.Theme.Secondary)
		} else {
			hdrColor = lipgloss.Color(d.styles.Theme.Primary)
		}
		label := lipgloss.NewStyle().
			Foreground(hdrColor).
			Bold(true).
			Render(item.headerName)
		ruleLen := width - lipgloss.Width(label) - 2
		if ruleLen < 0 {
			ruleLen = 0
		}
		rule := lipgloss.NewStyle().
			Foreground(lipgloss.Color(d.styles.Theme.Subtle)).
			Render(strings.Repeat("─", ruleLen))
		fmt.Fprintf(w, "\n%s %s", label, rule)
		return
	}

	ctx := item.ctx
	isSelected := index == m.Index()

	displayName := ctx.Name
	if ctx.IsExternal && strings.Contains(ctx.Name, "/") {
		parts := strings.SplitN(ctx.Name, "/", 2)
		displayName = parts[1]
	}

	// ── Per-group accent color ─────────────────────────────────────────────
	// Internal → Primary palette.  External → Secondary palette.
	var accentColor lipgloss.Color
	if ctx.IsExternal {
		accentColor = lipgloss.Color(d.styles.Theme.Secondary)
	} else {
		accentColor = lipgloss.Color(d.styles.Theme.Primary)
	}

	// ── Selection marker ──────────────────────────────────────────────────────
	var marker string
	if isSelected {
		marker = lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("▌")
	} else {
		marker = lipgloss.NewStyle().Foreground(lipgloss.Color(d.styles.Theme.Subtle)).Render(" ")
	}

	// ── Icon ──────────────────────────────────────────────────────────────────
	var iconRune string
	if ctx.IsExternal {
		iconRune = "󰒋 "
	} else {
		iconRune = "󱃾 "
	}
	var icon string
	if isSelected {
		icon = lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render(" " + iconRune)
	} else {
		if ctx.IsExternal {
			icon = lipgloss.NewStyle().Foreground(lipgloss.Color(d.styles.Theme.Secondary)).Render(" " + iconRune)
		} else {
			icon = lipgloss.NewStyle().Foreground(lipgloss.Color(d.styles.Theme.Subtle)).Render(" " + iconRune)
		}
	}

	// ── Name ──────────────────────────────────────────────────────────────────
	var name string
	if isSelected {
		name = lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render(displayName)
	} else {
		if ctx.IsExternal {
			name = lipgloss.NewStyle().Foreground(lipgloss.Color(d.styles.Theme.Secondary)).Render(displayName)
		} else {
			name = lipgloss.NewStyle().Foreground(lipgloss.Color(d.styles.Theme.Text)).Render(displayName)
		}
	}

	// ── Active badge (right-aligned on line 1) ────────────────────────────────
	activeBadge := ""
	if ctx.IsActive {
		activeBadge = d.styles.ActiveBadge.Render("active")
	}

	line1Left := marker + icon + name
	var line1 string
	if activeBadge != "" {
		gap := width - lipgloss.Width(line1Left) - lipgloss.Width(activeBadge)
		if gap < 1 {
			gap = 1
		}
		line1 = line1Left + strings.Repeat(" ", gap) + activeBadge
	} else {
		line1 = line1Left
	}

	// ── Line 2: active namespace only ────────────────────────────────────────────
	nsVal := lipgloss.NewStyle().
		Foreground(accentColor).
		Render("ns: " + ctx.Namespace)

	line2 := "   " + nsVal

	fmt.Fprintf(w, "%s\n%s", line1, line2)
}

// ── Model ─────────────────────────────────────────────────────────────────────

type ContextsModel struct {
	list        list.Model
	styles      Styles
	recentNames []string
}

func newContextsModel(contexts []kube.Context, styles Styles) ContextsModel {
	recentNames := kube.LoadRecentContexts()
	items := buildContextItems(contexts, recentNames)
	delegate := &contextDelegate{styles: styles}

	l := list.New(items, delegate, 80, 30)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.FilterPrompt = styles.SearchBox
	l.Styles.FilterCursor = styles.ActiveItem

	return ContextsModel{list: l, styles: styles, recentNames: recentNames}
}

func (m ContextsModel) Init() tea.Cmd { return nil }

func (m ContextsModel) Update(msg tea.Msg) (ContextsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "right", "l", " ":
			selected := m.list.SelectedItem()
			if selected == nil {
				return m, nil
			}
			item, ok := selected.(contextItem)
			if !ok || item.isHeader {
				return m, nil
			}
			return m, func() tea.Msg { return contextSelectedMsg{ctx: item.ctx} }

		case "s":
			selected := m.list.SelectedItem()
			if selected == nil {
				return m, nil
			}
			item, ok := selected.(contextItem)
			if !ok || item.isHeader {
				return m, nil
			}
			ctx := item.ctx
			return m, func() tea.Msg {
				var err error
				var exportCmd string
				if ctx.IsExternal {
					exportCmd = fmt.Sprintf("export KUBECONFIG=\"%s\"", ctx.ConfigPath)
				} else {
					err = kube.SetCurrentContext(ctx.Name)
					if err == nil {
						exportCmd = "unset KUBECONFIG"
					}
				}
				return contextSwitchedMsg{name: ctx.Name, err: err, envExport: exportCmd}
			}

		case "x":
			selected := m.list.SelectedItem()
			if selected == nil {
				return m, nil
			}
			item, ok := selected.(contextItem)
			if !ok || item.isHeader {
				return m, nil
			}
			return m, func() tea.Msg { return shellExecMsg{ctx: item.ctx} }
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ContextsModel) SelectedContext() *kube.Context {
	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}
	ci, ok := item.(contextItem)
	if !ok || ci.isHeader {
		return nil
	}
	return &ci.ctx
}

func (m ContextsModel) View() string         { return m.list.View() }
func (m *ContextsModel) setSize(w, h int)    { m.list.SetWidth(w); m.list.SetHeight(h) }

func (m *ContextsModel) applyStyles(styles Styles) {
	m.styles = styles
	m.list.Styles.FilterPrompt = styles.SearchBox
	m.list.Styles.FilterCursor = styles.ActiveItem
	m.list.SetDelegate(&contextDelegate{styles: styles})
}

func (m *ContextsModel) refreshItems(contexts []kube.Context, styles Styles) {
	m.recentNames = kube.LoadRecentContexts()
	m.list.SetItems(buildContextItems(contexts, m.recentNames))
	m.applyStyles(styles)
}

// ── buildContextItems ─────────────────────────────────────────────────────────

func buildContextItems(contexts []kube.Context, recentNames []string) []list.Item {
	// Index all contexts by name for the recent lookup.
	byName := make(map[string]kube.Context, len(contexts))
	for _, c := range contexts {
		byName[c.Name] = c
	}

	var internal, external []kube.Context
	for _, c := range contexts {
		if c.IsExternal {
			external = append(external, c)
		} else {
			internal = append(internal, c)
		}
	}

	sortCtxs := func(ctxs []kube.Context) {
		sort.Slice(ctxs, func(i, j int) bool {
			if ctxs[i].IsActive != ctxs[j].IsActive {
				return ctxs[i].IsActive
			}
			return strings.ToLower(ctxs[i].Name) < strings.ToLower(ctxs[j].Name)
		})
	}
	sortCtxs(internal)
	sortCtxs(external)

	var items []list.Item

	// ── Recent group (if any) ─────────────────────────────────────────────────
	var recentCtxs []kube.Context
	for _, name := range recentNames {
		if c, ok := byName[name]; ok {
			recentCtxs = append(recentCtxs, c)
		}
	}
	if len(recentCtxs) > 0 {
		items = append(items, contextItem{
			isHeader:   true,
			headerName: "◷  Recent",
		})
		for i, c := range recentCtxs {
			items = append(items, contextItem{ctx: c, isLast: i == len(recentCtxs)-1})
		}
		items = append(items, contextItem{isHeader: true, headerName: ""}) // spacer
	}

	// Internal group — header uses primary color (isExternalGroup=false)
	items = append(items, contextItem{
		isHeader:        true,
		headerName:      "󱃾  Internal Contexts",
		isExternalGroup: false,
	})
	if len(internal) == 0 {
		items = append(items, contextItem{isHeader: true, headerName: "  (none found)"})
	} else {
		for i, c := range internal {
			items = append(items, contextItem{ctx: c, isLast: i == len(internal)-1})
		}
	}

	// Spacer between groups
	items = append(items, contextItem{isHeader: true, headerName: ""})

	// External group — header uses secondary color (isExternalGroup=true)
	items = append(items, contextItem{
		isHeader:        true,
		headerName:      "󰒋  External Contexts",
		isExternalGroup: true,
	})
	if len(external) == 0 {
		items = append(items, contextItem{
			isHeader:        true,
			headerName:      "  (none — use 'kubes import <file> <alias>')",
			isExternalGroup: true,
		})
	} else {
		for i, c := range external {
			items = append(items, contextItem{ctx: c, isLast: i == len(external)-1})
		}
	}

	return items
}
