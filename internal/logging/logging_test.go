package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSetupRetention(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.Local)
	mk := func(daysAgo int) {
		ts := now.AddDate(0, 0, -daysAgo)
		p := filepath.Join(dir, fmt.Sprintf("sesion_%s.log", ts.Format("20060102_150405")))
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, ts, ts); err != nil {
			t.Fatal(err)
		}
	}
	for d := 1; d <= 25; d++ {
		mk(d)
	}
	for d := 40; d < 43; d++ {
		mk(d)
	}
	log, closer, err := Setup(dir, now)
	if err != nil {
		t.Fatal(err)
	}
	log.Info("hola")
	closer.Close()
	files, _ := filepath.Glob(filepath.Join(dir, "sesion_*.log"))
	if len(files) != 20 {
		t.Fatalf("quedaron %d logs, want 20", len(files))
	}
	newest := filepath.Join(dir, "sesion_20261112_100000.log")
	if b, err := os.ReadFile(newest); err != nil || len(b) == 0 {
		t.Fatalf("log nuevo vacío o inexistente: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, fmt.Sprintf("sesion_%s.log", now.AddDate(0, 0, -20).Format("20060102_150405")))); err == nil {
		t.Fatal("no se borró el log de hace 20 días (sobra uno de los 20 más nuevos)")
	}
}
