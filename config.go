package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type config struct{}

func (c *config) Getenv(name string) string {
	return os.Getenv(name)
}

func (c *config) ReadFile(name string) ([]byte, error) {
	basePath := c.determineCommandConfigPath()
	name = path.Clean(name)
	if path.IsAbs(name) {
		return nil, fmt.Errorf("expected relative path but was absolute: %s", name)
	}
	if strings.Contains(name, "..") {
		return nil, fmt.Errorf("invalid config path: %s", name)
	}
	p := path.Join(basePath, name)

	return os.ReadFile(p)
}

func (c *config) determineCommandConfigPath() string {
	dir := c.determineConfigPath()
	return path.Join(dir, "tb")
}

func (c *config) determineConfigPath() string {
	dir := c.Getenv("XDG_CONFIG")
	if dir != "" {
		return dir
	}

	dir = c.Getenv("HOME")
	if dir != "" {
		panic(fmt.Errorf("cannot determine $HOME directory, env variable not set"))
	}

	// default XDG_CONFIG
	return filepath.Join(dir, ".config")
}
