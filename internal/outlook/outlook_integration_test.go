//go:build windows && integration

// Tests contra el Outlook Classic real del equipo. Los corre el usuario:
//
//	go test -tags integration ./internal/outlook/ -v
//
// No imprimen asuntos, remitentes, nombres de carpeta ni de archivo: solo conteos.
package outlook

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"outlook_descargas_adjuntos/internal/domain"
	"outlook_descargas_adjuntos/internal/source"
)

func realSource(t *testing.T) *Source {
	t.Helper()
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if _, err := s.ListRoots(context.Background()); errors.Is(err, domain.ErrOutlookUnavailable) {
		t.Skip("Outlook Classic no está disponible en este equipo")
	} else if err != nil {
		t.Fatal(err)
	}
	return s
}

// inbox devuelve la Bandeja de entrada predeterminada (olFolderInbox = 6).
func inbox(t *testing.T, s *Source) domain.FolderRef {
	t.Helper()
	var ref domain.FolderRef
	err := s.sta.do(context.Background(), func() error {
		ns, err := s.namespace()
		if err != nil {
			return err
		}
		f, err := callObj(ns, "GetDefaultFolder", 6)
		if err != nil {
			return err
		}
		defer release(f)
		info, err := folderInfo(f)
		ref = info.Ref
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func TestIntegrationListRoots(t *testing.T) {
	s := realSource(t)
	roots, err := s.ListRoots(context.Background())
	if err != nil || len(roots) == 0 {
		t.Fatalf("ListRoots: %d raíces, %v", len(roots), err)
	}
	for _, r := range roots {
		if r.Ref.StoreID == "" || r.Ref.EntryID == "" {
			t.Fatal("una raíz no tiene StoreID o EntryID")
		}
	}
	kids, err := s.ListChildren(context.Background(), roots[0].Ref)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d raíces; la primera tiene %d subcarpetas", len(roots), len(kids))
}

func TestIntegrationScanFirst50(t *testing.T) {
	s := realSource(t)
	ref := inbox(t, s)
	n, mails, errs := 0, 0, 0
	var prev domain.Mail
	err := s.Scan(context.Background(), ref, func(it source.ScanItem) error {
		if it.Err != nil {
			errs++
		} else {
			if it.Mail.IsMail {
				mails++
			}
			if n > 0 && it.Mail.ReceivedAt.After(prev.ReceivedAt) {
				t.Errorf("orden no descendente en el item %d", n)
			}
			prev = it.Mail
		}
		n++
		if n == 50 {
			return source.ErrStopScan
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d items leídos (%d correos, %d con error de lectura)", n, mails, errs)
}

func TestIntegrationSaveFirstAttachment(t *testing.T) {
	s := realSource(t)
	ref := inbox(t, s)
	var mail domain.Mail
	var att domain.Attachment
	_ = s.Scan(context.Background(), ref, func(it source.ScanItem) error {
		if it.Err != nil || !it.Mail.IsMail {
			return nil
		}
		for _, a := range it.Mail.Attachments {
			if a.Kind == domain.KindFile {
				mail, att = it.Mail, a
				return source.ErrStopScan
			}
		}
		return nil
	})
	if mail.EntryID == "" {
		t.Skip("no hay adjuntos de archivo en la bandeja de entrada")
	}
	dest := filepath.Join(t.TempDir(), "adjunto.bin")
	if err := s.SaveAttachment(context.Background(), mail.EntryID, att.Index, dest); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(dest)
	if err != nil || st.Size() == 0 {
		t.Fatalf("archivo guardado vacío o inexistente: %v", err)
	}
	t.Logf("adjunto guardado: %d bytes", st.Size())
}
