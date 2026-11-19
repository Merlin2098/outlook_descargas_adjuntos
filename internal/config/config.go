// Package config guarda las preferencias del usuario en %APPDATA%\outlook_descargas_adjuntos.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"outlook_descargas_adjuntos/internal/domain"
)

const (
	appDir   = "outlook_descargas_adjuntos"
	fileName = "config.json"
)

// Config son las preferencias persistentes.
type Config struct {
	LastDestDir  string          `json:"lastDestDir"`
	Organize     domain.Organize `json:"organize"`
	SkipInline   bool            `json:"skipInline"`
	WindowWidth  int             `json:"windowWidth"`
	WindowHeight int             `json:"windowHeight"`
}

// Default devuelve la configuración de fábrica.
func Default() Config {
	return Config{Organize: domain.OrganizeByMonth, SkipInline: true, WindowWidth: 1280, WindowHeight: 720}
}

// Dir devuelve %APPDATA%\outlook_descargas_adjuntos.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appDir), nil
}

// Load lee la configuración. Si no existe devuelve Default(); si está corrupta, Default() y el error.
func Load(dir string) (Config, error) {
	b, err := os.ReadFile(filepath.Join(dir, fileName))
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), err
	}
	c := Default()
	if err := json.Unmarshal(b, &c); err != nil {
		return Default(), fmt.Errorf("config corrupta: %w", err)
	}
	return c, nil
}

// Save escribe la configuración de forma atómica (archivo temporal + rename).
func Save(dir string, c Config) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, fileName+".*.tmp")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, fileName))
}
