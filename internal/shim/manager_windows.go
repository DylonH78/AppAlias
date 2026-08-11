//go:build windows

package shim

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/DylonH78/AppAlias/internal/alias"
	"github.com/DylonH78/AppAlias/internal/layout"
)

type Manager struct {
	layout layout.Layout
}

func New(l layout.Layout) Manager { return Manager{layout: l} }

func (m Manager) Path(name string) string {
	return filepath.Join(m.layout.BinDir, name+".exe")
}

func (m Manager) Ensure(name string) error {
	if err := alias.Validate(name); err != nil {
		return err
	}
	if _, err := os.Stat(m.layout.LauncherPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("launcher.exe is missing from %s; reinstall AppAlias", m.layout.Root)
		}
		return fmt.Errorf("check launcher: %w", err)
	}
	if err := os.MkdirAll(m.layout.BinDir, 0o755); err != nil {
		return fmt.Errorf("create shim directory: %w", err)
	}
	target := m.Path(name)
	if _, err := os.Stat(target); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check shim: %w", err)
	}
	if err := os.Link(m.layout.LauncherPath, target); err == nil {
		return nil
	}
	// FAT/exFAT and some enterprise profiles do not support hard links. A copied
	// launcher has identical behavior and is repaired on each upgrade.
	if err := copyFile(m.layout.LauncherPath, target); err != nil {
		return fmt.Errorf("create launcher shim: %w", err)
	}
	return nil
}

func (m Manager) Replace(name string) error {
	path := m.Path(name)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove old shim %q: %w", name, err)
	}
	return m.Ensure(name)
}

func (m Manager) Remove(name string) error {
	err := os.Remove(m.Path(name))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("remove shim %q: %w", name, err)
	}
	return nil
}

// HasExternalPathCollision protects existing programs before AppAlias writes
// an executable into its own PATH directory.
func (m Manager) HasExternalPathCollision(name string) bool {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" || samePath(dir, m.layout.BinDir) {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, name+".exe")); err == nil {
			return true
		}
	}
	return false
}

func copyFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func samePath(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return strings.EqualFold(a, b)
	}
	return strings.EqualFold(aa, bb)
}
