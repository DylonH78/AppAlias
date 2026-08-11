package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DylonH78/AppAlias/internal/model"
)

type Store struct {
	path string
}

func New(path string) Store {
	return Store{path: path}
}

func (s Store) Path() string { return s.path }

func (s Store) Load() (model.Config, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return model.Config{SchemaVersion: model.CurrentSchemaVersion, Aliases: map[string]model.Alias{}}, nil
	}
	if err != nil {
		return model.Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg model.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return model.Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.SchemaVersion != model.CurrentSchemaVersion {
		return model.Config{}, fmt.Errorf("unsupported config schema version %d", cfg.SchemaVersion)
	}
	if cfg.Aliases == nil {
		cfg.Aliases = map[string]model.Alias{}
	}
	return cfg, nil
}

func (s Store) Save(cfg model.Config) error {
	if cfg.SchemaVersion == 0 {
		cfg.SchemaVersion = model.CurrentSchemaVersion
	}
	if cfg.SchemaVersion != model.CurrentSchemaVersion {
		return fmt.Errorf("cannot write unsupported config schema version %d", cfg.SchemaVersion)
	}
	if cfg.Aliases == nil {
		cfg.Aliases = map[string]model.Alias{}
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".aliases-*.json")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("replace config atomically: %w", err)
	}
	return nil
}
