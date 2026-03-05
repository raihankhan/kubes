package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raihankhan/kubes/internal/kube"
	"github.com/raihankhan/kubes/internal/ui"
)

var (
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555")).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b")).Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89"))
)

func main() {
	args := os.Args[1:]

	// ── CLI mode: kubes import <path> <alias> ────────────────────────────────
	if len(args) >= 1 && args[0] == "import" {
		runImport(args[1:])
		return
	}

	// ── TUI mode ─────────────────────────────────────────────────────────────
	model, err := ui.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render("Error initializing kubes: "+err.Error()))
		os.Exit(1)
	}

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	m, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render("Fatal error: "+err.Error()))
		os.Exit(1)
	}

	appModel := m.(ui.AppModel)
	if envOut := os.Getenv("KUBES_ENV_FILE"); envOut != "" {
		if exportCmd := appModel.GetEnvExport(); exportCmd != "" {
			err = os.WriteFile(envOut, []byte(exportCmd+"\n"), 0644)
			if err != nil {
				fmt.Fprintln(os.Stderr, errorStyle.Render("Failed to write to KUBES_ENV_FILE: "+err.Error()))
			}
		}
	}
}

// runImport handles 'kubes import <path> <alias>'
func runImport(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, errorStyle.Render("Usage: kubes import <path> <alias>"))
		os.Exit(1)
	}

	srcPath := args[0]
	alias := args[1]

	fmt.Printf("%s Importing kubeconfig %s as %s…\n",
		dimStyle.Render("󰒋"),
		dimStyle.Render(srcPath),
		dimStyle.Render(alias),
	)

	if err := kube.ImportConfig(srcPath, alias); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render("Error: "+err.Error()))
		os.Exit(1)
	}

	dir, _ := kube.ExternalContextDir()
	fmt.Println(successStyle.Render("✓ Imported successfully!"))
	fmt.Printf("%s Saved to %s/%s\n",
		dimStyle.Render("  "),
		dimStyle.Render(dir),
		dimStyle.Render(alias),
	)
	fmt.Println(dimStyle.Render("  Run 'kubes' to switch to this context."))
}
