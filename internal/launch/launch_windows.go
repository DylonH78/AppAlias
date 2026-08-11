//go:build windows

package launch

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/DylonH78/AppAlias/internal/model"
)

func Start(spec model.LaunchSpec) error {
	switch spec.Kind {
	case model.LaunchExecutable:
		if !strings.EqualFold(filepathExtension(spec.Target), ".exe") {
			return fmt.Errorf("only .exe launch targets are supported")
		}
		cmd := exec.Command(spec.Target, spec.Arguments...)
		if spec.WorkingDirectory != "" {
			cmd.Dir = spec.WorkingDirectory
		}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start %s: %w", spec.Target, err)
		}
		return nil
	case model.LaunchAppsFolder:
		if strings.TrimSpace(spec.AppUserModelID) == "" {
			return fmt.Errorf("missing AppUserModelId")
		}
		if err := exec.Command("explorer.exe", "shell:AppsFolder\\"+spec.AppUserModelID).Start(); err != nil {
			return fmt.Errorf("start Store application: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported launch kind %q", spec.Kind)
	}
}

func filepathExtension(path string) string {
	index := strings.LastIndex(path, ".")
	if index < 0 {
		return ""
	}
	return path[index:]
}
