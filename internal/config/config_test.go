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
	want := Config{LastDestDir: `C:\x`, Organize: domain.OrganizeFlat, SkipInline: false, WindowWidth: 1400, WindowHeight: 800}
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
