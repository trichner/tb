package cfg

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type kConfigProviderContextKey struct{}

func FromContext(ctx context.Context) *ConfigProvider {
	v := ctx.Value(kConfigProviderContextKey{})
	if v == nil {
		return nil
	}
	cfg, ok := v.(*ConfigProvider)
	if !ok {
		panic(fmt.Errorf("unexpected type for ConfigProvider context value: %v", v))
	}
	return cfg
}

func WithConfigProvider(ctx context.Context, cfg *ConfigProvider) context.Context {
	return context.WithValue(ctx, kConfigProviderContextKey{}, cfg)
}

type ConfigProvider struct{}

func (c *ConfigProvider) Getenv(name string) string {
	return os.Getenv(name)
}

func (c *ConfigProvider) ReadFile(name string) ([]byte, error) {
	basePath := c.determineCommandConfigPath()
	name = path.Clean(name)
	if path.IsAbs(name) {
		return nil, fmt.Errorf("expected relative path but was absolute: %s", name)
	}
	if strings.Contains(name, "..") {
		return nil, fmt.Errorf("invalid ConfigProvider path: %s", name)
	}
	p := path.Join(basePath, name)

	return os.ReadFile(p)
}

func (c *ConfigProvider) determineCommandConfigPath() string {
	dir := c.determineConfigPath()
	return path.Join(dir, "tb")
}

func (c *ConfigProvider) determineConfigPath() string {
	dir := c.Getenv("XDG_CONFIG")
	if dir != "" {
		return dir
	}

	dir = c.Getenv("HOME")
	if dir != "" {
		panic(fmt.Errorf("cannot determine $HOME directory, env variable not set"))
	}

	// default XDG_CONFIG
	return filepath.Join(dir, ".ConfigProvider")
}
