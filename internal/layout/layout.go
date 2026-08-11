package layout

import (
	"fmt"
	"os"
	"path/filepath"
)

type Layout struct {
	Root         string
	DataDir      string
	BinDir       string
	ConfigPath   string
	LauncherPath string
}

func Resolve(override string, portable bool) (Layout, error) {
	var root string
	if override != "" {
		root = override
	} else if portable {
		exe, err := os.Executable()
		if err != nil {
			return Layout{}, fmt.Errorf("locate executable: %w", err)
		}
		root = filepath.Dir(exe)
	} else if executableRoot, ok := existingApplicationRoot(); ok {
		// Once a portable copy has been initialized, subsequent invocations can
		// discover its colocated data without requiring --portable every time.
		root = executableRoot
	} else if local := os.Getenv("LOCALAPPDATA"); local != "" {
		root = filepath.Join(local, "AppAlias")
	} else {
		return Layout{}, fmt.Errorf("LOCALAPPDATA is not available")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return Layout{}, fmt.Errorf("resolve application root: %w", err)
	}
	return FromRoot(root), nil
}

func existingApplicationRoot() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	root := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(root, "launcher.exe")); err != nil {
		return "", false
	}
	if _, err := os.Stat(filepath.Join(root, "data")); err != nil {
		return "", false
	}
	return root, true
}

func FromRoot(root string) Layout {
	return Layout{
		Root:         root,
		DataDir:      filepath.Join(root, "data"),
		BinDir:       filepath.Join(root, "bin"),
		ConfigPath:   filepath.Join(root, "data", "aliases.json"),
		LauncherPath: filepath.Join(root, "launcher.exe"),
	}
}

func (l Layout) EnsureDirectories() error {
	if err := os.MkdirAll(l.DataDir, 0o755); err != nil {
		return err
	}
	return os.MkdirAll(l.BinDir, 0o755)
}
