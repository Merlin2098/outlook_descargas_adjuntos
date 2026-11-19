package config

import (
	"os"
	"path/filepath"
	"testing"

	"outlook_descargas_adjuntos/internal/domain"
)

func TestLoadMissing(t *testing.T) {
	c, err := Load(t.TempDir())
	if err != nil || c != Default() {
		t.Fatalf("Load = %+v, %v", c, err)
	}
}

func TestDefault(t *testing.T) {
	d := Default()
	if d.Organize != domain.OrganizeByMonth || !d.SkipInline || d.WindowWidth != 1280 || d.WindowHeight != 720 {
		t.Fatalf("Default = %+v", d)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := Config{LastDestDir: `C:\x`, Organize: domain.OrganizeFlat, SkipInline: false, WindowWidth: 1400, WindowHeight: 800, Theme: "dark"}
	if err := Save(dir, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil || got != want {
		t.Fatalf("Load = %+v, %v", got, err)
	}
}

func TestLoadCorrupt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte("{no es json"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err == nil || got != Default() {
		t.Fatalf("Load = %+v, %v", got, err)
	}
}

func TestConfigTheme(t *testing.T) {
	if got := Default().Theme; got != "light" {
		t.Fatalf("Default().Theme = %q, want light", got)
	}
	dir := t.TempDir()
	c := Default()
	c.Theme = "dark"
	if err := Save(dir, c); err != nil {
		t.Fatal(err)
	}
	if got, err := Load(dir); err != nil || got.Theme != "dark" {
		t.Fatalf("Load tras guardar dark = %q, %v", got.Theme, err)
	}
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(`{"theme": "morado"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := Load(dir); err != nil || got.Theme != "light" {
		t.Fatalf("Load con theme inválido = %q, %v", got.Theme, err)
	}
}
