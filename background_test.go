package main

import (
	"testing"

	"outlook_descargas_adjuntos/internal/config"
)

func TestBackgroundFor(t *testing.T) {
	if c := backgroundFor(config.ThemeDark); c.R != 13 || c.G != 15 || c.B != 34 {
		t.Fatalf("oscuro = %+v", c)
	}
	if c := backgroundFor(config.ThemeLight); c.R != 248 || c.G != 250 || c.B != 252 {
		t.Fatalf("claro = %+v", c)
	}
}
