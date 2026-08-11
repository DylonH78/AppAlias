//go:build windows

package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/DylonH78/AppAlias/internal/alias"
	"github.com/DylonH78/AppAlias/internal/model"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type linkInfo struct {
	Name             string `json:"Name"`
	TargetPath       string `json:"TargetPath"`
	Arguments        string `json:"Arguments"`
	WorkingDirectory string `json:"WorkingDirectory"`
}

type startApp struct {
	Name  string `json:"Name"`
	AppID string `json:"AppID"`
}

func Scan(ctx context.Context) model.ScanResult {
	result := model.ScanResult{}
	candidates := make(map[string]model.Candidate)
	add := func(displayName string, source model.Source, spec model.LaunchSpec) {
		candidate, ok := newCandidate(displayName, source, spec)
		if !ok {
			return
		}
		key := candidateKey(candidate)
		if _, found := candidates[key]; !found {
			candidates[key] = candidate
		}
	}

	for _, root := range startMenuRoots() {
		links, diagnostics := scanStartMenu(ctx, root)
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		for _, link := range links {
			add(link.name, model.SourceStartMenu, link.spec)
		}
	}
	appPathCandidates, diagnostics := scanAppPaths()
	result.Diagnostics = append(result.Diagnostics, diagnostics...)
	for _, candidate := range appPathCandidates {
		add(candidate.name, model.SourceAppPaths, candidate.spec)
	}
	uwpCandidates, diagnostics := scanStartApps(ctx)
	result.Diagnostics = append(result.Diagnostics, diagnostics...)
	for _, candidate := range uwpCandidates {
		add(candidate.name, model.SourceUWP, candidate.spec)
	}

	result.Candidates = make([]model.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		result.Candidates = append(result.Candidates, candidate)
	}
	sort.Slice(result.Candidates, func(i, j int) bool {
		return strings.ToLower(result.Candidates[i].DisplayName) < strings.ToLower(result.Candidates[j].DisplayName)
	})
	return result
}

type discovered struct {
	name string
	spec model.LaunchSpec
}

func startMenuRoots() []string {
	roots := []string{}
	if appData := os.Getenv("APPDATA"); appData != "" {
		roots = append(roots, filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs"))
	}
	if programData := os.Getenv("ProgramData"); programData != "" {
		roots = append(roots, filepath.Join(programData, "Microsoft", "Windows", "Start Menu", "Programs"))
	}
	return roots
}

func scanStartMenu(ctx context.Context, root string) ([]discovered, []string) {
	if _, err := os.Stat(root); err != nil {
		return nil, nil
	}
	batchContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	const script = "$ErrorActionPreference='Stop';[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new($false);$OutputEncoding=[Console]::OutputEncoding;$shell=New-Object -ComObject WScript.Shell;Get-ChildItem -LiteralPath $args[0] -Filter *.lnk -File -Recurse | ForEach-Object {$s=$shell.CreateShortcut($_.FullName);[pscustomobject]@{Name=$_.BaseName;TargetPath=$s.TargetPath;Arguments=$s.Arguments;WorkingDirectory=$s.WorkingDirectory}} | ConvertTo-Json -Compress"
	output, err := exec.CommandContext(batchContext, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script, root).Output()
	if err != nil {
		return nil, []string{fmt.Sprintf("scan Start Menu %s: %v", root, err)}
	}
	links, err := decodeLinks(output)
	if err != nil {
		return nil, []string{fmt.Sprintf("parse Start Menu %s: %v", root, err)}
	}
	found := make([]discovered, 0, len(links))
	var diagnostics []string
	for _, info := range links {
		if !safeExecutable(info.TargetPath) {
			continue
		}
		args, err := windows.CommandLineToArgv(info.Arguments)
		if err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("parse shortcut arguments for %s: %v", info.Name, err))
			continue
		}
		found = append(found, discovered{
			name: info.Name,
			spec: model.LaunchSpec{Kind: model.LaunchExecutable, Target: info.TargetPath, Arguments: args, WorkingDirectory: info.WorkingDirectory},
		})
	}
	return found, diagnostics
}

func decodeLinks(data []byte) ([]linkInfo, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "{") {
		var one linkInfo
		if err := json.Unmarshal(data, &one); err != nil {
			return nil, err
		}
		return []linkInfo{one}, nil
	}
	var many []linkInfo
	if err := json.Unmarshal(data, &many); err != nil {
		return nil, err
	}
	return many, nil
}

func scanAppPaths() ([]discovered, []string) {
	const appPathsKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths`
	var found []discovered
	var diagnostics []string
	open := []struct {
		root registry.Key
		view uint32
		name string
	}{
		{registry.CURRENT_USER, registry.WOW64_64KEY, "HKCU 64-bit"},
		{registry.LOCAL_MACHINE, registry.WOW64_64KEY, "HKLM 64-bit"},
		{registry.LOCAL_MACHINE, registry.WOW64_32KEY, "HKLM 32-bit"},
	}
	for _, location := range open {
		key, err := registry.OpenKey(location.root, appPathsKey, registry.READ|location.view)
		if err == registry.ErrNotExist {
			continue
		}
		if err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("open %s App Paths: %v", location.name, err))
			continue
		}
		names, err := key.ReadSubKeyNames(-1)
		if err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("read %s App Paths: %v", location.name, err))
			key.Close()
			continue
		}
		for _, name := range names {
			child, err := registry.OpenKey(key, name, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			target, _, getErr := child.GetStringValue("")
			child.Close()
			if getErr != nil || !safeExecutable(target) {
				continue
			}
			found = append(found, discovered{name: strings.TrimSuffix(name, filepath.Ext(name)), spec: model.LaunchSpec{Kind: model.LaunchExecutable, Target: target}})
		}
		key.Close()
	}
	return found, diagnostics
}

func scanStartApps(parent context.Context) ([]discovered, []string) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	const script = "[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new($false);$OutputEncoding=[Console]::OutputEncoding;Get-StartApps | Select-Object Name,AppID | ConvertTo-Json -Compress"
	output, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return nil, []string{fmt.Sprintf("scan Store applications: %v", err)}
	}
	apps, err := decodeStartApps(output)
	if err != nil {
		return nil, []string{fmt.Sprintf("parse Store applications: %v", err)}
	}
	result := make([]discovered, 0, len(apps))
	for _, app := range apps {
		if strings.TrimSpace(app.Name) == "" || !strings.Contains(app.AppID, "!") {
			continue
		}
		result = append(result, discovered{name: app.Name, spec: model.LaunchSpec{Kind: model.LaunchAppsFolder, AppUserModelID: app.AppID}})
	}
	return result, nil
}

func decodeStartApps(data []byte) ([]startApp, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "{") {
		var one startApp
		if err := json.Unmarshal(data, &one); err != nil {
			return nil, err
		}
		return []startApp{one}, nil
	}
	var many []startApp
	if err := json.Unmarshal(data, &many); err != nil {
		return nil, err
	}
	return many, nil
}

func newCandidate(displayName string, source model.Source, spec model.LaunchSpec) (model.Candidate, bool) {
	if displayName = strings.TrimSpace(displayName); displayName == "" {
		return model.Candidate{}, false
	}
	executable := ""
	if spec.Kind == model.LaunchExecutable {
		if !safeExecutable(spec.Target) {
			return model.Candidate{}, false
		}
		executable = spec.Target
	}
	suggestions := alias.Suggestions(displayName, executable)
	candidate := model.Candidate{
		ID:           stableID(source, spec),
		DisplayName:  displayName,
		Source:       source,
		Launch:       spec,
		Suggestions:  suggestions,
	}
	if len(suggestions) == 0 {
		candidate.RequiresConfirmation = true
		candidate.ConfirmationReason = "no valid alias suggestion"
	} else {
		candidate.Recommended = suggestions[0]
	}
	return candidate, true
}

func safeExecutable(path string) bool {
	if !strings.EqualFold(filepath.Ext(strings.TrimSpace(path)), ".exe") {
		return false
	}
	base := strings.ToLower(filepath.Base(path))
	return base != "cmd.exe" && base != "powershell.exe" && base != "pwsh.exe"
}

func stableID(source model.Source, spec model.LaunchSpec) string {
	value := strings.ToLower(string(source) + "\x00" + spec.Target + "\x00" + strings.Join(spec.Arguments, "\x00") + "\x00" + spec.AppUserModelID)
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:12])
}

func candidateKey(candidate model.Candidate) string {
	return string(candidate.Launch.Kind) + "\x00" + strings.ToLower(candidate.Launch.Target) + "\x00" + strings.Join(candidate.Launch.Arguments, "\x00") + "\x00" + strings.ToLower(candidate.Launch.AppUserModelID)
}
