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

type podViewMsg struct {
	ctx       kube.Context
	namespace string
}

type podsLoadedMsg struct {
	pods []kube.Pod
	err  error
}

// ── List item ────────────────────────────────────────────────────────────────

type podItem struct{ pod kube.Pod }

func (i podItem) FilterValue() string { return i.pod.Name }
func (i podItem) Title() string       { return i.pod.Name }
func (i podItem) Description() string { return i.pod.Status }

// ── Item Delegate ─────────────────────────────────────────────────────────────

type podDelegate struct{ styles Styles }

func (d podDelegate) Height() int                             { return 1 }
func (d podDelegate) Spacing() int                            { return 1 }
func (d podDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d podDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(podItem)
	if !ok {
		return
	}

	width := m.Width()
	if width <= 0 {
		width = 60
	}

	isSelected := index == m.Index()
	pod := item.pod
	t := d.styles.Theme

	// ── Selection marker ──────────────────────────────────────────────────────
	var marker string
	if isSelected {
		marker = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true).Render("▌")
	} else {
		marker = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Subtle)).Render(" ")
	}

	// ── Status-driven color ───────────────────────────────────────────────────
	statusColor := podStatusColor(pod.Status, t)

	// ── Icon ──────────────────────────────────────────────────────────────────
	iconStyle := lipgloss.NewStyle().Foreground(statusColor)
	if isSelected {
		iconStyle = iconStyle.Bold(true)
	}
	icon := iconStyle.Render(" 󰻗 ")

	// ── Name ──────────────────────────────────────────────────────────────────
	var nameS string
	if isSelected {
		nameS = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true).Render(pod.Name)
	} else {
		nameS = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Text)).Render(pod.Name)
	}

	left := marker + icon + nameS

	// ── Right: ready · status · age · restarts ───────────────────────────────
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Subtle))
	statusS := lipgloss.NewStyle().Foreground(statusColor).Bold(isSelected).Render(pod.Status)
	meta := dim.Render(pod.Ready) + "  " + statusS + "  " + dim.Render(pod.Age)
	if pod.Restarts > 0 {
		meta += "  " + lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Error)).
			Render(fmt.Sprintf("↺%d", pod.Restarts))
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(meta)
	if gap < 1 {
		gap = 1
	}
	fmt.Fprint(w, left+strings.Repeat(" ", gap)+meta)
}

func podStatusColor(status string, t Theme) lipgloss.Color {
	switch strings.ToLower(status) {
	case "running":
		return lipgloss.Color(t.Active)
	case "completed", "succeeded":
		return lipgloss.Color(t.Subtle)
	case "pending", "containercreating", "podscheduled":
		return lipgloss.Color(t.Secondary)
	default: // Error, CrashLoopBackOff, OOMKilled, etc.
		return lipgloss.Color(t.Error)
	}
}

// ── Load state ────────────────────────────────────────────────────────────────

type podLoadState int

const (
	podLoading podLoadState = iota
	podLoaded
	podError
)

// ── Model ─────────────────────────────────────────────────────────────────────

// PodsModel is the Bubbletea model for the pod quick-view pane.
type PodsModel struct {
	list      list.Model
	styles    Styles
	ctx       kube.Context
	namespace string
	loadState podLoadState
	errMsg    string
	spinner   spinner.Model
}

func newPodsModel(ctx kube.Context, namespace string, styles Styles) PodsModel {
	l := list.New([]list.Item{}, podDelegate{styles: styles}, 80, 30)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.FilterPrompt = styles.SearchBox
	l.Styles.FilterCursor = styles.ActiveItem

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styles.Spinner

	return PodsModel{
		list:      l,
		styles:    styles,
		ctx:       ctx,
		namespace: namespace,
		loadState: podLoading,
		spinner:   sp,
	}
}

func (m PodsModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchPods())
}

func (m PodsModel) fetchPods() tea.Cmd {
	ctx := m.ctx
	ns := m.namespace
	return func() tea.Msg {
		// Use InternalName so it matches the context name inside the kubeconfig file.
		pods, err := kube.GetPodsForNamespace(ctx.InternalName, ns, ctx.ConfigPath)
		return podsLoadedMsg{pods: pods, err: err}
	}
}

func (m PodsModel) Update(msg tea.Msg) (PodsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case podsLoadedMsg:
		if msg.err != nil {
			m.loadState = podError
			m.errMsg = msg.err.Error()
			return m, nil
		}
		items := make([]list.Item, len(msg.pods))
		for i, p := range msg.pods {
			items[i] = podItem{pod: p}
		}
		m.list.SetItems(items)
		m.loadState = podLoaded
		return m, nil

	case spinner.TickMsg:
		if m.loadState == podLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m PodsModel) View() string {
	switch m.loadState {
	case podLoading:
		return m.styles.DimText.PaddingLeft(2).Render(
			m.spinner.View() + " Fetching pods…",
		)
	case podError:
		return m.styles.ErrorText.PaddingLeft(2).Render(
			"⚠ Could not reach cluster\n\n" + m.errMsg,
		)
	default:
		return m.list.View()
	}
}

func (m *PodsModel) setSize(w, h int) {
	m.list.SetWidth(w)
	m.list.SetHeight(h)
}

func (m *PodsModel) applyStyles(styles Styles) {
	m.styles = styles
	m.list.Styles.FilterPrompt = styles.SearchBox
	m.list.Styles.FilterCursor = styles.ActiveItem
	m.list.SetDelegate(podDelegate{styles: styles})
	m.spinner.Style = styles.Spinner
}
