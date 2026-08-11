//go:build windows

package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DylonH78/AppAlias/internal/model"
)

func TestValidateLaunchRejectsScripts(t *testing.T) {
	if err := validateLaunch(model.LaunchSpec{Kind: model.LaunchExecutable, Target: `C:\temp\run.ps1`}); err == nil {
		t.Fatal("expected non-executable target to be rejected")
	}
}

func TestValidateLaunchAcceptsExistingExecutable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.exe")
	if err := os.WriteFile(path, []byte("not executed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateLaunch(model.LaunchSpec{Kind: model.LaunchExecutable, Target: path}); err != nil {
		t.Fatalf("expected existing .exe to be valid: %v", err)
	}
}
