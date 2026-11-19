//go:build windows

package outlook

import (
	"context"
	"errors"
	"testing"

	"outlook_descargas_adjuntos/internal/domain"
)

func TestConnectError(t *testing.T) {
	s, err := New(Options{ProgID: "OutlookDescargas.NoExiste"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Name() != "outlook" {
		t.Fatalf("Name = %q", s.Name())
	}
	_, err = s.ListRoots(context.Background())
	if !errors.Is(err, domain.ErrOutlookUnavailable) {
		t.Fatalf("ListRoots sin Outlook = %v, want ErrOutlookUnavailable", err)
	}
	// Reintenta en la siguiente llamada (no cachea el fallo) y sigue respondiendo.
	if _, err := s.ListChildren(context.Background(), domain.FolderRef{EntryID: "x", StoreID: "y"}); !errors.Is(err, domain.ErrOutlookUnavailable) {
		t.Fatalf("ListChildren sin Outlook = %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("segundo Close = %v", err)
	}
}
