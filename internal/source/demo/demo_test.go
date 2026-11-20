package demo

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"outlook_descargas_adjuntos/internal/domain"
	"outlook_descargas_adjuntos/internal/source"
)

var testNow = time.Date(2026, 11, 12, 10, 0, 0, 0, time.Local)

func newDefault() *Source { return New(DefaultFixture(testNow, 42), Options{}) }

func findChild(t *testing.T, s *Source, parent domain.FolderRef, name string) domain.Folder {
	t.Helper()
	kids, err := s.ListChildren(context.Background(), parent)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range kids {
		if k.Name == name {
			return k
		}
	}
	t.Fatalf("no existe la carpeta %q", name)
	return domain.Folder{}
}

func facturas(t *testing.T, s *Source) domain.Folder {
	t.Helper()
	roots, _ := s.ListRoots(context.Background())
	return findChild(t, s, roots[0].Ref, "Facturas proveedores")
}

func TestDefaultFixtureDeterministic(t *testing.T) {
	if !reflect.DeepEqual(DefaultFixture(testNow, 42), DefaultFixture(testNow, 42)) {
		t.Fatal("DefaultFixture no es determinista")
	}
}

func TestListRootsAndChildren(t *testing.T) {
	s := newDefault()
	roots, err := s.ListRoots(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, r := range roots {
		names = append(names, r.Name)
	}
	if want := []string{"demo@example.com", "Archivo local (PST)"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("raíces = %v, want %v", names, want)
	}
	if f := facturas(t, s); f.ItemCount != 312 {
		t.Fatalf("Facturas proveedores ItemCount = %d", f.ItemCount)
	}
	if p := findChild(t, s, roots[0].Ref, "Proyectos"); !p.HasChildren {
		t.Fatal("Proyectos debería tener hijos")
	}
}

func TestScanOrderAndStop(t *testing.T) {
	s := newDefault()
	ref := facturas(t, s).Ref
	var prev time.Time
	n := 0
	err := s.Scan(context.Background(), ref, func(it source.ScanItem) error {
		if n > 0 && it.Err == nil && it.Mail.ReceivedAt.After(prev) {
			t.Fatalf("orden no descendente en el elemento %d", n)
		}
		if it.Err == nil {
			prev = it.Mail.ReceivedAt
		}
		n++
		return nil
	})
	if err != nil || n != 312 {
		t.Fatalf("Scan = %v, %d elementos", err, n)
	}

	calls := 0
	err = s.Scan(context.Background(), ref, func(source.ScanItem) error {
		calls++
		if calls == 10 {
			return source.ErrStopScan
		}
		return nil
	})
	if err != nil || calls != 10 {
		t.Fatalf("Scan con stop = %v, %d llamadas", err, calls)
	}
}

func TestScanReadErr(t *testing.T) {
	s := newDefault()
	errs := 0
	_ = s.Scan(context.Background(), facturas(t, s).Ref, func(it source.ScanItem) error {
		if it.Err != nil {
			errs++
		}
		return nil
	})
	if errs != 1 {
		t.Fatalf("elementos con error de lectura = %d, want 1", errs)
	}
}

func TestSaveAttachment(t *testing.T) {
	s := newDefault()
	var ok, failing domain.Mail
	var failIdx int
	_ = s.Scan(context.Background(), facturas(t, s).Ref, func(it source.ScanItem) error {
		if it.Err != nil || !it.Mail.IsMail || len(it.Mail.Attachments) == 0 {
			return nil
		}
		if ok.EntryID == "" && it.Mail.Attachments[0].Kind == domain.KindFile {
			ok = it.Mail
		}
		for _, a := range it.Mail.Attachments {
			if a.FileName == "herramienta.exe" {
				failing, failIdx = it.Mail, a.Index
			}
		}
		return nil
	})
	dir := t.TempDir()

	dest := filepath.Join(dir, "ok.bin")
	if err := s.SaveAttachment(context.Background(), ok.EntryID, 1, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	want, err := fileContent(ok, FixtureAttachment{FileName: ok.Attachments[0].FileName, Kind: domain.KindFile})
	if err != nil || len(want) == 0 || !bytes.Equal(got, want) {
		t.Fatalf("SaveAttachment no escribió el contenido generado (%d bytes, want %d): %v", len(got), len(want), err)
	}

	err = s.SaveAttachment(context.Background(), failing.EntryID, failIdx, filepath.Join(dir, "x.exe"))
	if err == nil || !strings.Contains(err.Error(), "Adjunto bloqueado por Outlook") {
		t.Fatalf("err = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.SaveAttachment(ctx, ok.EntryID, 1, filepath.Join(dir, "c.bin")); !errors.Is(err, context.Canceled) {
		t.Fatalf("err con ctx cancelado = %v", err)
	}
}
