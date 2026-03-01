package ui

import "github.com/charmbracelet/lipgloss"

// Theme holds all color values for a visual palette.
type Theme struct {
	Name      string
	BG        string
	Surface   string
	Primary   string
	Secondary string
	Text      string
	Subtle    string
	Active    string
	Error     string
}

// All available themes.
var (
	ThemeKubes = Theme{
		Name:      "Kubes",
		BG:        "#1a1b26",
		Surface:   "#24283b",
		Primary:   "#FF8700",
		Secondary: "#005FAF",
		Text:      "#c0caf5",
		Subtle:    "#565f89",
		Active:    "#FF8700",
		Error:     "#f7768e",
	}
	ThemeCatppuccin = Theme{
		Name:      "Catppuccin",
		BG:        "#24273a",
		Surface:   "#1e2030",
		Primary:   "#c6a0f6",
		Secondary: "#8aadf4",
		Text:      "#cad3f5",
		Subtle:    "#6e738d",
		Active:    "#a6da95",
		Error:     "#ed8796",
	}
	ThemeNord = Theme{
		Name:      "Nord",
		BG:        "#2E3440",
		Surface:   "#3B4252",
		Primary:   "#88C0D0",
		Secondary: "#81A1C1",
		Text:      "#ECEFF4",
		Subtle:    "#4C566A",
		Active:    "#A3BE8C",
		Error:     "#BF616A",
	}
	ThemeDracula = Theme{
		Name:      "Dracula",
		BG:        "#282A36",
		Surface:   "#343746",
		Primary:   "#FF79C6",
		Secondary: "#BD93F9",
		Text:      "#F8F8F2",
		Subtle:    "#6272A4",
		Active:    "#50FA7B",
		Error:     "#FF5555",
	}
	ThemeGruvbox = Theme{
		Name:      "Gruvbox Dark",
		BG:        "#282828",
		Surface:   "#3c3836",
		Primary:   "#FABD2F",
		Secondary: "#8EC07C",
		Text:      "#ebdbb2",
		Subtle:    "#928374",
		Active:    "#b8bb26",
		Error:     "#fb4934",
	}

	// Themes ordered for cycling.
	Themes = []Theme{
		ThemeKubes,
		ThemeCatppuccin,
		ThemeNord,
		ThemeDracula,
		ThemeGruvbox,
	}
)

// Styles holds all pre-computed Lipgloss styles for a given theme.
type Styles struct {
	Theme Theme

	// Base
	AppBG   lipgloss.Style
	Surface lipgloss.Style

	// Text
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Text      lipgloss.Style
	DimText   lipgloss.Style
	ErrorText lipgloss.Style

	// List items
	ActiveItem  lipgloss.Style
	NormalItem  lipgloss.Style
	GroupHeader lipgloss.Style

	// Badges
	ActiveBadge  lipgloss.Style
	ExternalIcon lipgloss.Style

	// Structural
	Border             lipgloss.Style
	PaneBorderActive   lipgloss.Style
	PaneBorderInactive lipgloss.Style
	StatusBar          lipgloss.Style
	HelpBar            lipgloss.Style
	SearchBox          lipgloss.Style
	Spinner            lipgloss.Style

	// Tree/Branch chars
	TreeBranch   lipgloss.Style
	TreeActive   lipgloss.Style
	TreeInactive lipgloss.Style

	// Divider
	Divider lipgloss.Style
}

// NewStyles builds a Styles struct from the given Theme.
func NewStyles(t Theme) Styles {
	s := Styles{Theme: t}

	s.AppBG = lipgloss.NewStyle().
		Background(lipgloss.Color(t.BG))

	s.Surface = lipgloss.NewStyle().
		Background(lipgloss.Color(t.Surface))

	s.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Primary)).
		Bold(true).
		PaddingLeft(1)

	s.Subtitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Secondary)).
		Italic(true).
		PaddingLeft(1)

	s.Text = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Text))

	s.DimText = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Subtle))

	s.ErrorText = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Error)).
		Bold(true)

	s.ActiveItem = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Primary)).
		Bold(true).
		PaddingLeft(2)

	s.NormalItem = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Text)).
		PaddingLeft(2)

	s.GroupHeader = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Secondary)).
		Bold(true).
		PaddingLeft(1).
		MarginTop(1)

	s.ActiveBadge = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.BG)).
		Background(lipgloss.Color(t.Active)).
		Bold(true).
		Padding(0, 1)

	s.ExternalIcon = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Secondary))

	s.Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Primary)).
		Padding(0, 1)

	s.PaneBorderActive = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Primary)).
		Padding(0, 1)

	s.PaneBorderInactive = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Subtle)).
		Padding(0, 1)

	s.StatusBar = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.BG)).
		Background(lipgloss.Color(t.Primary)).
		Bold(true).
		Padding(0, 1)

	s.HelpBar = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Subtle)).
		PaddingLeft(1)

	s.SearchBox = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Text)).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(t.Secondary)).
		Padding(0, 1)

	s.Spinner = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Primary))

	s.TreeBranch = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Subtle))

	s.TreeActive = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Primary))

	s.TreeInactive = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Subtle))

	s.Divider = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Subtle))

	return s
}
