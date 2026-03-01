package kube

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ExternalContextDir returns the directory where external kubeconfigs are stored.
func ExternalContextDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".kubes", "context", "config"), nil
}

// ExternalContextPath returns the full path to an external kubeconfig by alias.
func ExternalContextPath(alias string) (string, error) {
	dir, err := ExternalContextDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, alias), nil
}

// GetExternalContexts scans the external context directory and returns contexts.
// The filename is used as the alias/display name.
func GetExternalContexts() ([]Context, error) {
	dir, err := ExternalContextDir()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating external context dir: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading external context dir: %w", err)
	}

	activePath := DefaultKubeConfigPath()
	currentCtx := ""
	if activeCfg, _ := loadRawConfig(activePath); activeCfg != nil {
		currentCtx = activeCfg.CurrentContext
	}

	var contexts []Context
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		alias := entry.Name()
		path := filepath.Join(dir, alias)

		// Try to load and inspect the kubeconfig to get cluster info.
		cfg, err := loadRawConfig(path)
		if err != nil {
			// If we can't load it, still show it with the alias.
			contexts = append(contexts, Context{
				Name:          alias,
				ExternalAlias: alias,
				IsExternal:    true,
				Namespace:     "default",
				ConfigPath:    path,
			})
			continue
		}

		// Use the internal context name from the file, display alias as label.
		for ctxName, ctx := range cfg.Contexts {
			ns := ctx.Namespace
			if ns == "" {
				ns = "default"
			}
			mergedName := alias + "/" + ctxName
			isActive := false
			if activePath == path && ctxName == currentCtx {
				isActive = true
			}

			contexts = append(contexts, Context{
				Name:          mergedName,
				Cluster:       ctx.Cluster,
				Namespace:     ns,
				AuthInfo:      ctx.AuthInfo,
				IsActive:      isActive,
				IsExternal:    true,
				ExternalAlias: alias,
				ConfigPath:    path,
				InternalName:  ctxName,
			})
			break // Only first context per external file
		}
	}
	return contexts, nil
}

// ImportConfig copies a kubeconfig file into the external context directory
// with the given alias as the filename.
func ImportConfig(srcPath, alias string) error {
	if alias == "" {
		return fmt.Errorf("alias cannot be empty")
	}

	// Validate the source is a readable kubeconfig.
	if _, err := loadRawConfig(srcPath); err != nil {
		return fmt.Errorf("invalid kubeconfig at %s: %w", srcPath, err)
	}

	destDir, err := ExternalContextDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating external context dir: %w", err)
	}

	destPath := filepath.Join(destDir, alias)

	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copying kubeconfig: %w", err)
	}

	return nil
}
