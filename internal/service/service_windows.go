//go:build windows

package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/DylonH78/AppAlias/internal/alias"
	"github.com/DylonH78/AppAlias/internal/config"
	"github.com/DylonH78/AppAlias/internal/launch"
	"github.com/DylonH78/AppAlias/internal/layout"
	"github.com/DylonH78/AppAlias/internal/model"
	"github.com/DylonH78/AppAlias/internal/pathenv"
	"github.com/DylonH78/AppAlias/internal/scanner"
	"github.com/DylonH78/AppAlias/internal/shim"
)

type Service struct {
	layout layout.Layout
	store  config.Store
	shims  shim.Manager
}

type ApplyReport struct {
	Applied []string          `json:"applied"`
	Skipped map[string]string `json:"skipped"`
}

type DoctorReport struct {
	Root            string   `json:"root"`
	BinDirectory    string   `json:"binDirectory"`
	PathInSession   bool     `json:"pathInCurrentSession"`
	LauncherPresent bool     `json:"launcherPresent"`
	Issues          []string `json:"issues,omitempty"`
}

func New(rootOverride string, portable bool) (*Service, error) {
	l, err := layout.Resolve(rootOverride, portable)
	if err != nil {
		return nil, err
	}
	return &Service{layout: l, store: config.New(l.ConfigPath), shims: shim.New(l)}, nil
}

func (s *Service) Layout() layout.Layout { return s.layout }

func (s *Service) Init() (bool, error) {
	if err := s.layout.EnsureDirectories(); err != nil {
		return false, fmt.Errorf("create AppAlias directories: %w", err)
	}
	if _, err := os.Stat(s.layout.LauncherPath); err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("launcher.exe is missing from %s; use the installer or keep it beside appalias.exe in portable mode", s.layout.Root)
		}
		return false, err
	}
	if _, err := s.store.Load(); err != nil {
		return false, err
	}
	return pathenv.Ensure(s.layout.BinDir)
}

func (s *Service) Scan(ctx context.Context) model.ScanResult {
	return scanner.Scan(ctx)
}

func (s *Service) Apply(candidates []model.Candidate) (ApplyReport, error) {
	report := ApplyReport{Skipped: map[string]string{}}
	cfg, err := s.store.Load()
	if err != nil {
		return report, err
	}
	counts := make(map[string]int)
	for _, candidate := range candidates {
		if candidate.Recommended != "" {
			counts[alias.Key(candidate.Recommended)]++
		}
	}
	created := make([]string, 0)
	for _, candidate := range candidates {
		name := candidate.Recommended
		if name == "" {
			report.Skipped[candidate.DisplayName] = "no safe recommended alias"
			continue
		}
		if counts[alias.Key(name)] != 1 {
			report.Skipped[candidate.DisplayName] = "candidate alias conflicts with another discovered application"
			continue
		}
		if reason := s.available(cfg, name, ""); reason != "" {
			report.Skipped[candidate.DisplayName] = reason
			continue
		}
		if err := validateLaunch(candidate.Launch); err != nil {
			report.Skipped[candidate.DisplayName] = err.Error()
			continue
		}
		if err := s.shims.Ensure(name); err != nil {
			for _, rollback := range created {
				_ = s.shims.Remove(rollback)
			}
			return report, err
		}
		cfg.Aliases[alias.Key(name)] = toAlias(name, candidate.DisplayName, candidate.Source, candidate.ID, candidate.Launch)
		created = append(created, name)
		report.Applied = append(report.Applied, name)
	}
	if err := s.store.Save(cfg); err != nil {
		for _, rollback := range created {
			_ = s.shims.Remove(rollback)
		}
		return ApplyReport{Skipped: report.Skipped}, err
	}
	return report, nil
}

func (s *Service) Add(name, displayName string, launchSpec model.LaunchSpec) error {
	if strings.TrimSpace(displayName) == "" {
		displayName = name
	}
	if err := validateLaunch(launchSpec); err != nil {
		return err
	}
	cfg, err := s.store.Load()
	if err != nil {
		return err
	}
	if reason := s.available(cfg, name, ""); reason != "" {
		return fmt.Errorf("cannot use alias %q: %s", name, reason)
	}
	if err := s.shims.Ensure(name); err != nil {
		return err
	}
	cfg.Aliases[alias.Key(name)] = toAlias(name, displayName, model.SourceManual, "", launchSpec)
	if err := s.store.Save(cfg); err != nil {
		_ = s.shims.Remove(name)
		return err
	}
	return nil
}

func (s *Service) Rename(oldName, newName string) error {
	cfg, err := s.store.Load()
	if err != nil {
		return err
	}
	oldKey := alias.Key(oldName)
	item, found := cfg.Aliases[oldKey]
	if !found {
		return fmt.Errorf("alias %q does not exist", oldName)
	}
	if reason := s.available(cfg, newName, oldKey); reason != "" {
		return fmt.Errorf("cannot use alias %q: %s", newName, reason)
	}
	if alias.Key(oldName) == alias.Key(newName) {
		item.Name = newName
		cfg.Aliases[oldKey] = item
		return s.store.Save(cfg)
	}
	if err := s.shims.Ensure(newName); err != nil {
		return err
	}
	delete(cfg.Aliases, oldKey)
	item.Name = newName
	cfg.Aliases[alias.Key(newName)] = item
	if err := s.store.Save(cfg); err != nil {
		_ = s.shims.Remove(newName)
		return err
	}
	if err := s.shims.Remove(oldName); err != nil {
		return err
	}
	return nil
}

func (s *Service) RemoveFromUserPath() (bool, error) {
	return pathenv.Remove(s.layout.BinDir)
}

func (s *Service) Remove(name string) error {
	cfg, err := s.store.Load()
	if err != nil {
		return err
	}
	key := alias.Key(name)
	item, found := cfg.Aliases[key]
	if !found {
		return fmt.Errorf("alias %q does not exist", name)
	}
	if err := s.shims.Remove(item.Name); err != nil {
		return err
	}
	delete(cfg.Aliases, key)
	if err := s.store.Save(cfg); err != nil {
		_ = s.shims.Ensure(item.Name)
		return err
	}
	return nil
}

func (s *Service) List() ([]model.Alias, error) {
	cfg, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	items := make([]model.Alias, 0, len(cfg.Aliases))
	for _, item := range cfg.Aliases {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name) })
	return items, nil
}

func (s *Service) Launch(name string) error {
	cfg, err := s.store.Load()
	if err != nil {
		return err
	}
	item, found := cfg.Aliases[alias.Key(name)]
	if !found {
		return fmt.Errorf("alias %q does not exist", name)
	}
	return launch.Start(item.Launch)
}

func (s *Service) Repair() ([]string, error) {
	items, err := s.List()
	if err != nil {
		return nil, err
	}
	repaired := make([]string, 0, len(items))
	for _, item := range items {
		if err := s.shims.Replace(item.Name); err != nil {
			return repaired, err
		}
		repaired = append(repaired, item.Name)
	}
	return repaired, nil
}

func (s *Service) Doctor() (DoctorReport, error) {
	report := DoctorReport{
		Root:          s.layout.Root,
		BinDirectory:  s.layout.BinDir,
		PathInSession: pathenv.Contains(s.layout.BinDir),
	}
	if _, err := os.Stat(s.layout.LauncherPath); err == nil {
		report.LauncherPresent = true
	} else {
		report.Issues = append(report.Issues, "launcher.exe is missing")
	}
	if !report.PathInSession {
		report.Issues = append(report.Issues, "bin directory is not visible to this PowerShell session; open a new terminal after init")
	}
	items, err := s.List()
	if err != nil {
		return report, err
	}
	for _, item := range items {
		if _, err := os.Stat(s.shims.Path(item.Name)); err != nil {
			report.Issues = append(report.Issues, fmt.Sprintf("missing shim for %s", item.Name))
		}
		if item.Launch.Kind == model.LaunchExecutable {
			if _, err := os.Stat(item.Launch.Target); err != nil {
				report.Issues = append(report.Issues, fmt.Sprintf("missing target for %s: %s", item.Name, item.Launch.Target))
			}
		}
	}
	return report, nil
}

func (s *Service) available(cfg model.Config, name, ignoredKey string) string {
	if err := alias.Validate(name); err != nil {
		return err.Error()
	}
	key := alias.Key(name)
	if key != ignoredKey {
		if _, found := cfg.Aliases[key]; found {
			return "already managed by AppAlias"
		}
	}
	if s.shims.HasExternalPathCollision(name) {
		return "an executable with this name already exists on PATH"
	}
	return ""
}

func validateLaunch(spec model.LaunchSpec) error {
	switch spec.Kind {
	case model.LaunchExecutable:
		if !strings.EqualFold(filepath.Ext(spec.Target), ".exe") {
			return fmt.Errorf("only .exe targets are supported")
		}
		base := strings.ToLower(filepath.Base(spec.Target))
		if base == "cmd.exe" || base == "powershell.exe" || base == "pwsh.exe" {
			return fmt.Errorf("command-shell wrappers are not supported")
		}
		if _, err := os.Stat(spec.Target); err != nil {
			return fmt.Errorf("target executable is unavailable: %w", err)
		}
		return nil
	case model.LaunchAppsFolder:
		if strings.TrimSpace(spec.AppUserModelID) == "" {
			return fmt.Errorf("AppUserModelId is required")
		}
		return nil
	default:
		return fmt.Errorf("unsupported launch kind %q", spec.Kind)
	}
}

func toAlias(name, displayName string, source model.Source, candidateID string, launchSpec model.LaunchSpec) model.Alias {
	return model.Alias{
		Name:        name,
		CandidateID: candidateID,
		Source:      source,
		DisplayName: displayName,
		Launch:      launchSpec,
		CreatedAt:   time.Now().UTC(),
	}
}
