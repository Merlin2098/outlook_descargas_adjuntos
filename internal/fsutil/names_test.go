package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeFileName(t *testing.T) {
	cases := []struct{ in, want string }{
		{`a<b>:c"d/e\f|g?h*.pdf`, "a_b__c_d_e_f_g_h_.pdf"},
		{"CON.pdf", "_CON.pdf"},
		{"lpt1.txt", "_lpt1.txt"},
		{"", "adjunto"},
		{"   ", "adjunto"},
		{"informe. ", "informe"},
		{"fac\x01tura.pdf", "fac_tura.pdf"},
		{"año_2026.xlsx", "año_2026.xlsx"},
	}
	for _, c := range cases {
		if got := SanitizeFileName(c.in); got != c.want {
			t.Errorf("SanitizeFileName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFitPath(t *testing.T) {
	dir := `C:\` + strings.Repeat("d", 47) // 50 caracteres
	long := strings.Repeat("a", 300) + ".pdf"
	got, err := FitPath(dir, long, MaxPath)
	if err != nil {
		t.Fatalf("FitPath error: %v", err)
	}
	if n := len([]rune(filepath.Join(dir, got))); n > MaxPath {
		t.Fatalf("ruta de %d caracteres, máximo %d", n, MaxPath)
	}
	if !strings.HasSuffix(got, ".pdf") {
		t.Fatalf("perdió la extensión: %q", got)
	}

	short, err := FitPath(dir, "f.pdf", MaxPath)
	if err != nil || short != "f.pdf" {
		t.Fatalf("FitPath corto = %q, %v", short, err)
	}

	tooLong := `C:\` + strings.Repeat("d", 227) // 230 caracteres
	if _, err := FitPath(tooLong, "f.pdf", MaxPath); !errors.Is(err, ErrPathTooLong) {
		t.Fatalf("err = %v, want ErrPathTooLong", err)
	}
}

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestUniquePath(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "f.pdf"))
	touch(t, filepath.Join(dir, "f_1.pdf"))
	if got, err := UniquePath(dir, "f.pdf"); err != nil || got != filepath.Join(dir, "f_2.pdf") {
		t.Fatalf("UniquePath = %q, %v", got, err)
	}

	empty := t.TempDir()
	if got, err := UniquePath(empty, "f.pdf"); err != nil || got != filepath.Join(empty, "f.pdf") {
		t.Fatalf("UniquePath vacío = %q, %v", got, err)
	}

	touch(t, filepath.Join(dir, "LEEME"))
	if got, err := UniquePath(dir, "LEEME"); err != nil || got != filepath.Join(dir, "LEEME_1") {
		t.Fatalf("UniquePath sin extensión = %q, %v", got, err)
	}
}
