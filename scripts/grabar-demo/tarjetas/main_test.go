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

func TestPickShowcase(t *testing.T) {
	tree := Tree{Folders: []Folder{
		{Name: "2026-10", Files: []File{{Name: "factura_F001-00090.pdf"}, {Name: "factura_F001-00090.xml"}}},
		{Name: "2026-11", Files: []File{
			{Name: "consulta_01.docx"},
			{Name: "estado_cuenta_mayo_1.xlsx"},
			{Name: "estado_cuenta_mayo.xlsx"},
			{Name: "factura_F001-00120_1.pdf"},
			{Name: "factura_F001-00120_1.xml"},
			{Name: "factura_F001-00121.pdf"},
			{Name: "factura_F001-00122.pdf"},
			{Name: "factura_F001-00122.xml"},
		}},
	}}
	s, err := pickShowcase(tree)
	if err != nil {
		t.Fatal(err)
	}
	want := Showcase{Folder: "2026-11", PDF: "factura_F001-00122.pdf", XML: "factura_F001-00122.xml", XLSX: "estado_cuenta_mayo.xlsx"}
	if s != want {
		t.Fatalf("pickShowcase = %+v, want %+v", s, want)
	}
	// Con el filtro pdf, xlsx no hay XML: la factura se muestra igual, sin su XML.
	solo := Tree{Folders: []Folder{{Name: "2026-11", Files: []File{{Name: "factura_F001-00130.pdf"}, {Name: "estado_cuenta_mayo.xlsx"}}}}}
	if s, err := pickShowcase(solo); err != nil || s.PDF != "factura_F001-00130.pdf" || s.XML != "" {
		t.Fatalf("sin XML: %+v, %v", s, err)
	}
	if _, err := pickShowcase(Tree{Folders: []Folder{{Name: "x"}}}); err == nil {
		t.Fatal("sin factura ni estado de cuenta debería fallar")
	}
}

func TestSpanishNumber(t *testing.T) {
	cases := map[string]string{"1,980.52": "1.980,52", "175.10": "175,10", "10,394.82": "10.394,82", "F001-00263": "F001-00263", "09/10/2026": "09/10/2026"}
	for in, want := range cases {
		if got := spanishNumber(in); got != want {
			t.Errorf("spanishNumber(%q) = %q, want %q", in, got, want)
		}
	}
}
