package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHumanSize(t *testing.T) {
	cases := map[int64]string{900: "900 B", 54_000: "53 KB", 19_713_229: "18,8 MB"}
	for n, want := range cases {
		if got := humanSize(n); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestScanDest(t *testing.T) {
	dir := t.TempDir()
	write := func(p string, n int) {
		t.Helper()
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, make([]byte, n), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("2026-11/b.pdf", 10)
	write("2026-10/a.pdf", 5)
	write("2026-10/a.xml", 7)
	write("reporte_2026-11-19_1500.xlsx", 1)
	write("reporte_2026-11-19_1600.xlsx", 1)

	tree, err := scanDest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if tree.Report != "reporte_2026-11-19_1600.xlsx" {
		t.Errorf("Report = %q, want el más reciente", tree.Report)
	}
	if len(tree.Folders) != 2 || tree.Folders[0].Name != "2026-10" || tree.Folders[0].Bytes != 12 || len(tree.Folders[0].Files) != 2 {
		t.Errorf("Folders = %+v", tree.Folders)
	}
}

func TestPickColumns(t *testing.T) {
	rows := [][]string{
		{"Fecha correo", "Remitente", "Asunto", "Archivo", "Ruta guardada", "Tamaño (KB)"},
		{"19/11/2026", "a@example.com", "Factura", "f.pdf", `C:\demo\f.pdf`, "91"},
		{"18/11/2026", "b@example.com"}, // fila corta: las celdas que faltan quedan vacías
	}
	got := pickColumns(rows, []string{"Archivo", "Fecha", "Tamaño"})
	want := [][]string{{"Archivo", "Fecha correo", "Tamaño (KB)"}, {"f.pdf", "19/11/2026", "91"}, {"", "18/11/2026", ""}}
	if len(got) != len(want) {
		t.Fatalf("filas = %d, want %d", len(got), len(want))
	}
	for i := range want {
		for j := range want[i] {
			if got[i][j] != want[i][j] {
				t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	}
}
