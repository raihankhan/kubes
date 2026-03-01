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

// ── Messages ─────────────────────────────────────────────────────────────────

// contextSelectedMsg is sent when the user presses Enter on a context
// (triggering the namespace view).
type contextSelectedMsg struct{ ctx kube.Context }

// contextSwitchedMsg is sent after activating a context without entering NS view.
type contextSwitchedMsg struct {
	name      string
	err       error
	envExport string
}

// ── List item ────────────────────────────────────────────────────────────────

type contextItem struct {
	ctx        kube.Context
	isHeader   bool
	headerName string
	isLast     bool // true if this is the last item in its branch
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

func (d contextDelegate) Height() int                             { return 4 }
func (d contextDelegate) Spacing() int                            { return 0 }
func (d contextDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d contextDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(contextItem)
	if !ok {
		return
	}

	if item.isHeader {
		header := d.styles.GroupHeader.Render(" " + item.headerName)
		fmt.Fprint(w, header) // Removed the arbitrary long divider line beneath headers
		return
	}

	ctx := item.ctx
	isSelected := index == m.Index()

	// Display name: strip alias prefix for readability.
	displayName := ctx.Name
	if ctx.IsExternal && strings.Contains(ctx.Name, "/") {
		parts := strings.SplitN(ctx.Name, "/", 2)
		displayName = parts[1]
	}

	// ── Icons & Selection ─────────────────────────────────────────────────────
	var icon string
	if ctx.IsExternal {
		icon = d.styles.ExternalIcon.Render("󰒋 ")
	} else {
		icon = d.styles.DimText.Render("󱃾 ")
	}

	// ── Status Badges ─────────────────────────────────────────────────────────
	var badge string
	if ctx.IsActive {
		badge = " " + d.styles.ActiveBadge.Render("active")
	}

	nsInfo := d.styles.DimText.Render(fmt.Sprintf(" (%s)", ctx.Namespace))

	// ── Text colors & Box layout ───────────────────────────────────────────────
	var boxStyle lipgloss.Style

	if isSelected {
		boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(d.styles.Theme.Primary)).
			Padding(0, 1).
			Width(60) // fixed width for a clean look
	} else {
		boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(d.styles.Theme.Surface)).
			Padding(0, 1).
			Width(60) // fixed width for a clean look
	}

	// ── Render lines ──────────────────────────────────────────────────────────
	// Line 1: icon + name + badge + namespace
	nameStyle := d.styles.NormalItem
	clusterStyle := d.styles.DimText
	if isSelected {
		nameStyle = d.styles.ActiveItem
	}

	line1 := nameStyle.Render(icon+displayName) + badge + nsInfo
	line2 := "  " + clusterStyle.Render(" cluster: "+ctx.Cluster)

	if !isSelected {
		line2 = d.styles.DimText.Render(line2) // Dim entire 2nd line if not selected
	}

	// Combine lines and wrap in the box
	content := line1 + "\n" + line2
	renderedBox := boxStyle.Render(content)

	fmt.Fprint(w, renderedBox)
}

// ── Model ────────────────────────────────────────────────────────────────────

// ContextsModel is the Bubbletea model for the context branched list.
type ContextsModel struct {
	list   list.Model
	styles Styles
}

func newContextsModel(contexts []kube.Context, styles Styles) ContextsModel {
	items := buildContextItems(contexts)

	delegate := &contextDelegate{styles: styles}

	l := list.New(items, delegate, 80, 30)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.FilterPrompt = styles.SearchBox
	l.Styles.FilterCursor = styles.ActiveItem

	return ContextsModel{list: l, styles: styles}
}

func (m ContextsModel) Init() tea.Cmd {
	return nil
}

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
			// Enter transitions to namespace view.
			return m, func() tea.Msg { return contextSelectedMsg{ctx: item.ctx} }

		case "s":
			// 's' = switch context immediately without entering namespace view.
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
					// No merge, just point KUBECONFIG to the external file
					exportCmd = fmt.Sprintf("export KUBECONFIG=\"%s\"", ctx.ConfigPath)
				} else {
					err = kube.SetCurrentContext(ctx.Name)
					if err == nil {
						exportCmd = "unset KUBECONFIG"
					}
				}
				return contextSwitchedMsg{name: ctx.Name, err: err, envExport: exportCmd}
			}
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

func (m ContextsModel) View() string {
	return m.list.View()
}

func (m *ContextsModel) setSize(w, h int) {
	m.list.SetWidth(w)
	m.list.SetHeight(h)
}

func (m *ContextsModel) applyStyles(styles Styles) {
	m.styles = styles
	m.list.Styles.FilterPrompt = styles.SearchBox
	m.list.Styles.FilterCursor = styles.ActiveItem
	m.list.SetDelegate(&contextDelegate{styles: styles})
}

func (m *ContextsModel) refreshItems(contexts []kube.Context, styles Styles) {
	items := buildContextItems(contexts)
	m.list.SetItems(items)
	m.applyStyles(styles)
}

// buildContextItems builds the flat list items including group headers.
func buildContextItems(contexts []kube.Context) []list.Item {
	var internal, external []kube.Context
	for _, c := range contexts {
		if c.IsExternal {
			external = append(external, c)
		} else {
			internal = append(internal, c)
		}
	}

	// Sort each group alphabetically, active first.
	sortContexts := func(ctxs []kube.Context) {
		sort.Slice(ctxs, func(i, j int) bool {
			if ctxs[i].IsActive != ctxs[j].IsActive {
				return ctxs[i].IsActive
			}
			return strings.ToLower(ctxs[i].Name) < strings.ToLower(ctxs[j].Name)
		})
	}
	sortContexts(internal)
	sortContexts(external)

	var items []list.Item

	// Internal group.
	items = append(items, contextItem{
		isHeader:   true,
		headerName: lipgloss.NewStyle().Render("󱃾  Internal Contexts"),
	})
	if len(internal) == 0 {
		items = append(items, contextItem{
			isHeader:   true,
			headerName: "  (none found)",
		})
	} else {
		for i, c := range internal {
			items = append(items, contextItem{
				ctx:    c,
				isLast: i == len(internal)-1,
			})
		}
	}

	// Margin between groups
	items = append(items, contextItem{isHeader: true, headerName: ""})

	// External group.
	items = append(items, contextItem{
		isHeader:   true,
		headerName: lipgloss.NewStyle().Render("󰒋  External Contexts"),
	})
	if len(external) == 0 {
		items = append(items, contextItem{
			isHeader:   true,
			headerName: "  (none — use 'kubes import <file> <alias>')",
		})
	} else {
		for i, c := range external {
			items = append(items, contextItem{
				ctx:    c,
				isLast: i == len(external)-1,
			})
		}
	}

	return items
}
