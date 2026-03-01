package kube

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// DefaultKubeConfigPath returns the current active kubeconfig path (respects KUBECONFIG env).
func DefaultKubeConfigPath() string {
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env
	}
	return StandardKubeConfigPath()
}

// StandardKubeConfigPath returns the hardcoded default path ~/.kube/config.
func StandardKubeConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}

// Context represents a kubeconfig context entry.
type Context struct {
	Name          string
	Cluster       string
	Namespace     string
	AuthInfo      string
	IsActive      bool
	IsExternal    bool
	ExternalAlias string
	ConfigPath    string
	InternalName  string
}

// GetInternalContexts parses ~/.kube/config and returns all context entries.
func GetInternalContexts() ([]Context, error) {
	path := StandardKubeConfigPath()
	cfg, err := loadRawConfig(path)
	if err != nil {
		return nil, fmt.Errorf("loading internal kubeconfig: %w", err)
	}

	activePath := DefaultKubeConfigPath()
	activeCfg, _ := loadRawConfig(activePath)
	current := ""
	if activePath == path && activeCfg != nil {
		current = activeCfg.CurrentContext
	}
	var contexts []Context
	for name, ctx := range cfg.Contexts {
		ns := ctx.Namespace
		if ns == "" {
			ns = "default"
		}
		cluster := ""
		if ctx.Cluster != "" {
			cluster = ctx.Cluster
		}
		contexts = append(contexts, Context{
			Name:         name,
			Cluster:      cluster,
			Namespace:    ns,
			AuthInfo:     ctx.AuthInfo,
			IsActive:     name == current,
			IsExternal:   false,
			ConfigPath:   path,
			InternalName: name,
		})
	}
	return contexts, nil
}

// SetCurrentContext updates the current-context field in ~/.kube/config.
func SetCurrentContext(name string) error {
	path := StandardKubeConfigPath()
	cfg, err := loadRawConfig(path)
	if err != nil {
		return fmt.Errorf("loading internal kubeconfig: %w", err)
	}

	if _, ok := cfg.Contexts[name]; !ok {
		return fmt.Errorf("context %q not found in kubeconfig", name)
	}

	cfg.CurrentContext = name
	return writeRawConfig(path, cfg)
}

// SetCurrentContextFromExternal sets the active kubeconfig to an external file.
// It merges the external file's context into the default kubeconfig.
func SetCurrentContextFromExternal(alias string) error {
	extPath, err := ExternalContextPath(alias)
	if err != nil {
		return err
	}

	extCfg, err := loadRawConfig(extPath)
	if err != nil {
		return fmt.Errorf("loading external kubeconfig: %w", err)
	}

	defaultPath := DefaultKubeConfigPath()
	defaultCfg, err := loadRawConfig(defaultPath)
	if err != nil {
		// If default doesn't exist, start fresh.
		defaultCfg = clientcmdapi.NewConfig()
	}

	// Merge external entries (prefixed with alias) into default config.
	for ctxName, ctx := range extCfg.Contexts {
		mergedName := alias + "/" + ctxName
		defaultCfg.Contexts[mergedName] = ctx
	}
	for clusterName, cluster := range extCfg.Clusters {
		defaultCfg.Clusters[clusterName] = cluster
	}
	for authName, info := range extCfg.AuthInfos {
		defaultCfg.AuthInfos[authName] = info
	}

	// Find the first context from the external file to set as current.
	for ctxName := range extCfg.Contexts {
		defaultCfg.CurrentContext = alias + "/" + ctxName
		break
	}

	return writeRawConfig(defaultPath, defaultCfg)
}

// SetContextNamespace updates the namespace for a specific context in a given kubeconfig file.
func SetContextNamespace(path, contextName, namespace string) error {
	if path == "" {
		path = DefaultKubeConfigPath()
	}
	cfg, err := loadRawConfig(path)
	if err != nil {
		return fmt.Errorf("loading kubeconfig: %w", err)
	}

	ctx, ok := cfg.Contexts[contextName]
	if !ok {
		return fmt.Errorf("context %q not found in kubeconfig", contextName)
	}

	ctx.Namespace = namespace
	cfg.Contexts[contextName] = ctx
	return writeRawConfig(path, cfg)
}

// GetCurrentContext returns the name of the currently active context.
func GetCurrentContext() (string, error) {
	path := DefaultKubeConfigPath()
	cfg, err := loadRawConfig(path)
	if err != nil {
		return "", err
	}
	return cfg.CurrentContext, nil
}

// GetCurrentNamespace returns the namespace of the currently active context.
func GetCurrentNamespace() (string, error) {
	path := DefaultKubeConfigPath()
	cfg, err := loadRawConfig(path)
	if err != nil {
		return "", err
	}
	ctx, ok := cfg.Contexts[cfg.CurrentContext]
	if !ok {
		return "default", nil
	}
	if ctx.Namespace == "" {
		return "default", nil
	}
	return ctx.Namespace, nil
}

// loadRawConfig loads the raw kubeconfig from a file path.
func loadRawConfig(path string) (*clientcmdapi.Config, error) {
	cfg, err := clientcmd.LoadFromFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return clientcmdapi.NewConfig(), nil
		}
		return nil, err
	}
	return cfg, nil
}

// writeRawConfig writes the raw kubeconfig to a file path.
func writeRawConfig(path string, cfg *clientcmdapi.Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return clientcmd.WriteToFile(*cfg, path)
}
