//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DylonH78/AppAlias/internal/alias"
	"github.com/DylonH78/AppAlias/internal/config"
	"github.com/DylonH78/AppAlias/internal/launch"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "AppAlias launcher:", err)
		os.Exit(1)
	}
}

func run() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate launcher: %w", err)
	}
	name := strings.TrimSuffix(filepath.Base(executable), filepath.Ext(executable))
	root := filepath.Dir(filepath.Dir(executable))
	store := config.New(filepath.Join(root, "data", "aliases.json"))
	cfg, err := store.Load()
	if err != nil {
		return err
	}
	item, found := cfg.Aliases[alias.Key(name)]
	if !found {
		return fmt.Errorf("no configured alias named %q", name)
	}
	return launch.Start(item.Launch)
}
