package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAppVersionLdflags comprueba que la release puede inyectar la versión con -ldflags y que
// --version la imprime sin abrir la ventana.
func TestAppVersionLdflags(t *testing.T) {
	if testing.Short() {
		t.Skip("compila el binario completo")
	}
	exe := filepath.Join(t.TempDir(), "version-test.exe")
	build := exec.Command("go", "build", "-ldflags", "-X main.appVersion=9.9.9", "-o", exe, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, exe, "--version").Output()
	if err != nil {
		t.Fatalf("--version: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "9.9.9" {
		t.Fatalf("--version imprimió %q, want 9.9.9", got)
	}
}
