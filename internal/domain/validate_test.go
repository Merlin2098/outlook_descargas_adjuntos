package domain

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func validReq(t *testing.T) JobRequest {
	t.Helper()
	day := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)
	return JobRequest{
		Folder:  FolderRef{StoreID: "s", EntryID: "e", Path: `cuenta\carpeta`},
		Filters: Filters{From: day, To: day},
		DestDir: t.TempDir(),
	}
}

func TestValidate(t *testing.T) {
	fileDest := filepath.Join(t.TempDir(), "archivo.txt")
	if err := os.WriteFile(fileDest, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		mutate func(*JobRequest)
		want   error
	}{
		{"sin carpeta", func(r *JobRequest) { r.Folder.EntryID = "" }, ErrNoFolder},
		{"sin destino", func(r *JobRequest) { r.DestDir = "" }, ErrNoDest},
		{"destino es archivo", func(r *JobRequest) { r.DestDir = fileDest }, ErrDestIsFile},
		{"rango invertido", func(r *JobRequest) { r.Filters.From = r.Filters.To.AddDate(0, 0, 1) }, ErrBadRange},
		{"mismo día", func(r *JobRequest) {}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := validReq(t)
			c.mutate(&r)
			if err := r.Validate(); !errors.Is(err, c.want) {
				t.Fatalf("Validate() = %v, want %v", err, c.want)
			}
		})
	}
}

func TestValidateCreatesMissingDest(t *testing.T) {
	r := validReq(t)
	r.DestDir = filepath.Join(t.TempDir(), "nueva", "sub")
	if err := r.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
	if st, err := os.Stat(r.DestDir); err != nil || !st.IsDir() {
		t.Fatalf("destino no creado: %v", err)
	}
	entries, _ := os.ReadDir(r.DestDir)
	if len(entries) != 0 {
		t.Fatalf("quedaron archivos de prueba: %v", entries)
	}
}

func TestNormalizeExtensions(t *testing.T) {
	got := NormalizeExtensions([]string{" PDF", "xlsx", ".Xml", "", "pdf"})
	want := []string{".pdf", ".xlsx", ".xml"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
