package kube

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const maxRecent = 5

func recentFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kubes", "recent.json")
}

// LoadRecentContexts returns the list of recently used context names, newest first.
func LoadRecentContexts() []string {
	data, err := os.ReadFile(recentFilePath())
	if err != nil {
		return nil
	}
	var names []string
	_ = json.Unmarshal(data, &names)
	return names
}

// AddRecentContext records name as the most recently used context.
// It keeps at most maxRecent entries and deduplicates.
func AddRecentContext(name string) {
	existing := LoadRecentContexts()
	filtered := make([]string, 0, len(existing))
	for _, n := range existing {
		if n != name {
			filtered = append(filtered, n)
		}
	}
	recent := append([]string{name}, filtered...)
	if len(recent) > maxRecent {
		recent = recent[:maxRecent]
	}
	path := recentFilePath()
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := json.MarshalIndent(recent, "", "  ")
	_ = os.WriteFile(path, data, 0644)
}
