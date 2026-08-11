//go:build windows

package shim

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DylonH78/AppAlias/internal/layout"
)

func TestEnsureAndReplaceShim(t *testing.T) {
	root := t.TempDir()
	l := layout.FromRoot(root)
	if err := l.EnsureDirectories(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(l.LauncherPath, []byte("launcher"), 0o600); err != nil {
		t.Fatal(err)
	}
	manager := New(l)
	if err := manager.Ensure("demo"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(l.BinDir, "demo.exe")); err != nil {
		t.Fatalf("shim was not created: %v", err)
	}
	if err := manager.Replace("demo"); err != nil {
		t.Fatal(err)
	}
}
