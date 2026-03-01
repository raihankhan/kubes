# kubes

```text
  _  __       _                  
 | |/ /      | |                 
 | ' / _   _ | |__    ___  ___  
 |  < | | | || '_ \  / _ \/ __| 
 | . \| |_| || |_) ||  __/\__ \ 
 |_|\_\\__,_||_.__/  \___||___/ 
      Modern K8s Context Tool
```

[![Release](https://img.shields.io/github/v/release/raihankhan/kubes?style=flat-square)](https://github.com/raihankhan/kubes/releases/latest)
[![Build](https://img.shields.io/github/actions/workflow/status/raihankhan/kubes/release.yml?style=flat-square&label=build)](https://github.com/raihankhan/kubes/actions)
[![License](https://img.shields.io/github/license/raihankhan/kubes?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/raihankhan/kubes?style=flat-square)](go.mod)

A modern, high-performance Terminal User Interface (TUI) for Kubernetes context and namespace switching — a polished, themed successor to `kubectx` and `kubens`.

## Features

- **5 Visual Themes** — Kubes (default), Catppuccin, Nord, Dracula, Gruvbox Dark — cycle with `t`
- **Interactive Import** — Import external kubeconfigs directly within the TUI (`i` key)
- **Startup Greeting** — Instant overview of your active context and cluster on launch
- **Branched Context View** — Internal (`~/.kube/config`) and External (`~/.kubes/context/config/`) in one list
- **Namespace Switcher** — Searchable live namespace list fetched from the active cluster
- **NerdFont Typography** — `󱃾` contexts · `󰋘` namespaces · `󰒋` external files

---

## � Local Build & Development

If you want to build and run **Kubes** locally from source:

```bash
# 1. Clone the repository
git clone https://github.com/raihankhan/kubes.git
cd kubes

# 2. Install dependencies
go mod tidy

# 3. Build the binary
go build -o kubes ./cmd/kubes/

# 4. Run locally
./kubes
```

---

## 📦 Installation

For production use, choose the method that fits your workflow:

### 1. Homebrew (macOS & Linux) — recommended

```bash
brew install raihankhan/tap/kubes
```

### 2. Shell Script (instant, no package manager needed)

```bash
curl -sL https://raw.githubusercontent.com/raihankhan/kubes/main/install.sh | sh
```

### 3. Windows (Scoop)

```bash
scoop bucket add kubes https://github.com/raihankhan/scoop-bucket.git
scoop install kubes
```

### 4. From source (Go install)

```bash
go install github.com/raihankhan/kubes/cmd/kubes@latest
```

---

## 🚀 Getting Started

1. **Launch**: Just run `kubes`. You'll be greeted with a card showing your current context and namespace.
2. **Navigate**: Use arrows or `j`/`k` to move through contexts.
3. **Switch Context**: Press `Enter` on a context to see its namespaces, or `s` to switch immediately.
4. **Import**: Press `i` to interactively import a new kubeconfig. It will ask for the file path and an alias.

---

## 🐚 Shell Integration (Important!)

If you want Kubes to actively change the `KUBECONFIG` environment variable in your current terminal when switching to External Contexts, you must add a shell wrapper function.

Add the following to your `~/.zshrc` or `~/.bashrc`:

```bash
kubes() {
    export KUBES_ENV_FILE=$(mktemp)
    command kubes "$@"
    local exit_code=$?

    if [ -f "$KUBES_ENV_FILE" ]; then
        source "$KUBES_ENV_FILE"
        rm -f "$KUBES_ENV_FILE"
    fi

    return $exit_code
}
```

Restart your terminal, and `kubes` will seamlessly set and unset `KUBECONFIG` for you.

---

## ⌨️ Keybindings

| Key | Action |
|---|---|
| `↑` / `↓`  or  `k` / `j` | Navigate list |
| `Enter` | Select context → open namespace view |
| `s` | Switch to highlighted context immediately |
| `i` | **Import external kubeconfig interactively** |
| `esc` | Back to context view / Show greeting |
| `t` | Cycle through themes |
| `/` | Filter / search |
| `?` | Toggle help bar |
| `q` / `Ctrl+C` | Quit |

---

## 📝 License

[Apache 2.0](LICENSE)
