```markdown
# Project: Kubes CLI
**Description:** A modern, high-performance Terminal User Interface (TUI) for Kubernetes context and namespace switching. It acts as a visual, themed, and extensible successor to `kubectx` and `kubens`.
**Vibe:** Sleek, fast, highly polished, with smooth view transitions and beautiful NerdFont typography.

## 1. Visual Design System
### Theme Engine
The UI supports 5 dynamic color palettes toggleable via a hotkey (e.g., `[t]`). Styles must be reactive.

| Theme Name | Background | Primary (Accent) | Secondary |
| :--- | :--- | :--- | :--- |
| **Kubes (Default)** | `#1a1b26` | `#FF8700` (Orange) | `#005FAF` (Blue) |
| **Catppuccin** | `#24273a` | `#c6a0f6` (Lavender) | `#8aadf4` (Sapphire) |
| **Nord** | `#2E3440` | `#88C0D0` (Frost Blue) | `#81A1C1` (Glacier) |
| **Dracula** | `#282A36` | `#FF79C6` (Pink) | `#BD93F9` (Purple) |
| **Gruvbox Dark** | `#282828` | `#FABD2F` (Yellow) | `#8EC07C` (Aqua) |

### TUI Components & Animations
- **Typography/Icons:** Use NerdFonts heavily. `󱃾` for Kubernetes/Contexts, `󰋘` for Namespaces, `󰒋` for External files.
- **Layout:** A unified, centered dashboard or a split-pane design.
- **Transitions:** Use state machines in the main Bubbletea `Update` function to switch cleanly between the "Context View" and the "Namespace View".

## 2. Core Features & Logic

### Feature 1: The Context Switcher (Branched Pane)
- **Data Sources:** - *Internal Contexts:* Parsed from the default `~/.kube/config`.
  - *External Contexts:* Parsed from `~/.kubes/context/config/`. The filename acts as the context alias.
- **The UI (Branched Pane):** A single, unified tree-like list where "Internal" and "External" are collapsible or clearly distinct parent branches. 
- **The Action:** - *Internal:* Updates the `current-context` in `~/.kube/config`.
  - *External:* Safely points the active KUBECONFIG environment or swaps the session to use the external file without polluting the default config.

### Feature 2: The Namespace Switcher
- **The Data:** Fetches all available namespaces for the *currently active context* (whether internal or external).
- **The UI:** A searchable `bubbles/list`.
- **The Action:** Modifies the current context's `namespace` field inside the active kubeconfig file. This ensures all subsequent `kubectl` commands default to the selected namespace without the `-n` flag.

### Feature 3: External Config Import (CLI)
- **The Flow:** Handled purely via CLI arguments, not the TUI.
- **Command:** `kubes import <path-to-kubeconfig> <alias>`
- **The Action:** Copies the target file into `~/.kubes/context/config/<alias>` and sanitizes it if necessary.

## 3. Project Architecture

### Tech Stack
- **Language:** Golang (1.21+)
- **TUI Framework:** `github.com/charmbracelet/bubbletea`, `bubbles`, `lipgloss`
- **K8s Interaction:** `k8s.io/client-go/tools/clientcmd` (Crucial for safely reading, modifying, and writing kubeconfig files).

### Directory Structure
```text
kubes/
├── cmd/
│   └── kubes/
│       └── main.go              # Entry point. Handles 'import' flag or launches TUI.
├── internal/
│   ├── kube/
│   │   ├── config.go            # Logic for reading/writing ~/.kube/config
│   │   ├── external.go          # Logic for CLI import and reading ~/.kubes/context/config
│   │   └── namespace.go         # Logic to fetch namespaces via client-go
│   └── ui/
│       ├── theme.go             # Palette definitions & Lipgloss style generators
│       ├── layout.go            # Main state machine (swapping Context/Namespace views)
│       ├── contexts.go          # Bubbletea model for Branched Context tree
│       └── namespaces.go        # Bubbletea model for Namespace selection

```

## 4. Implementation Roadmap

### Phase 1: Kubeconfig Data Layer & CLI Commands

1. Implement `internal/kube/config.go` to parse `~/.kube/config`. Create functions to `GetContexts()` and `SetCurrentContext()`.
2. Implement the `kubes import <file> <alias>` command logic in `cmd/kubes/main.go` and `internal/kube/external.go`.
3. Create the reading logic to scan `~/.kubes/context/config` and map filenames as aliases.

### Phase 2: Theme Engine & UI Shell

1. Create `internal/ui/theme.go` with the 5 predefined palettes.
2. Build `internal/ui/layout.go`. Implement a basic Bubbletea router that can swap between the `contexts.go` model and the `namespaces.go` model.

### Phase 3: Branched Contexts & Namespace Views

1. Build the Contexts view. Use a grouped list or tree structure to display Internal vs. External branches. Bind the `Enter` key to activate the selected context.
2. Build the Namespaces view. On initialization, it must query the K8s API using the newly active context to list namespaces. Bind `Enter` to update the namespace field in the relevant kubeconfig.

### Phase 4: Polish

1. Add error handling for unreachable clusters when fetching namespaces.
2. Ensure the active item in the list clearly indicates if it's currently selected in the background kubeconfig.

```