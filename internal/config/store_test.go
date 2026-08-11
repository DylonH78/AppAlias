package config

import (
	"path/filepath"
	"testing"

	"github.com/DylonH78/AppAlias/internal/model"
)

func TestStoreRoundTrip(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "data", "aliases.json"))
	want := model.Config{SchemaVersion: model.CurrentSchemaVersion, Aliases: map[string]model.Alias{
		"code": {Name: "code", Source: model.SourceManual, DisplayName: "VS Code"},
	}}
	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Aliases["code"].DisplayName != "VS Code" {
		t.Fatalf("unexpected config: %#v", got)
	}
}
