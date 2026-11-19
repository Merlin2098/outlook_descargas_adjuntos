//go:build windows

package outlook

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestCloseDoesNotHangWhenSTABusy(t *testing.T) {
	s, err := New(Options{ProgID: "OutlookDescargas.NoExiste"})
	if err != nil {
		t.Fatal(err)
	}
	orig := closeTimeout
	closeTimeout = 100 * time.Millisecond
	defer func() { closeTimeout = orig }()

	started, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	go func() {
		_ = s.sta.do(context.Background(), func() error { // simula Outlook colgado en una llamada COM
			close(started)
			<-release
			return nil
		})
	}()
	<-started
	t0 := time.Now()
	err = s.Close()
	if el := time.Since(t0); el > time.Second {
		t.Fatalf("Close tardó %v con el STA ocupado", el)
	}
	if err == nil {
		t.Fatal("Close con el STA ocupado debería informar que no se pudo liberar Outlook")
	}
}
