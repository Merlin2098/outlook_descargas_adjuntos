//go:build windows

package outlook

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"outlook_descargas_adjuntos/internal/domain"
)

func TestOleDateToLocal(t *testing.T) {
	cases := []struct {
		days float64
		want time.Time
	}{
		{0, time.Date(1899, 12, 30, 0, 0, 0, 0, time.Local)},
		{45000.5, time.Date(2023, 3, 15, 12, 0, 0, 0, time.Local)},
		{45000.25, time.Date(2023, 3, 15, 6, 0, 0, 0, time.Local)},
		{46338.99999884259, time.Date(2026, 11, 12, 23, 59, 59, 900_000_000, time.Local)},
	}
	for _, c := range cases {
		got := oleDateToLocal(c.days)
		if got.Location() != time.Local {
			t.Errorf("%v: Location = %v", c.days, got.Location())
		}
		if d := got.Sub(c.want); d < -time.Millisecond || d > time.Millisecond {
			t.Errorf("oleDateToLocal(%v) = %v, want %v", c.days, got, c.want)
		}
	}
}

func TestClassifyAttachment(t *testing.T) {
	cases := []struct {
		attType   int
		hidden    bool
		contentID string
		htmlCID   bool
		want      domain.AttachmentKind
	}{
		{olByValue, false, "", false, domain.KindFile},
		{olByValue, true, "", false, domain.KindInline},
		{olByValue, false, "img@x", true, domain.KindInline},
		{olByValue, false, "img@x", false, domain.KindFile},
		{olEmbeddedItem, false, "", false, domain.KindEmbeddedMsg},
		{olOLE, false, "", false, domain.KindOther},
		{olByReference, false, "", false, domain.KindOther},
	}
	for _, c := range cases {
		if got := classifyAttachment(c.attType, c.hidden, c.contentID, c.htmlCID); got != c.want {
			t.Errorf("classifyAttachment(%d,%v,%q,%v) = %v, want %v", c.attType, c.hidden, c.contentID, c.htmlCID, got, c.want)
		}
	}
}

func TestAttachmentFileName(t *testing.T) {
	cases := []struct {
		name  string
		kind  domain.AttachmentKind
		index int
		want  string
	}{
		{"Acta", domain.KindEmbeddedMsg, 2, "Acta.msg"},
		{"Acta.msg", domain.KindEmbeddedMsg, 2, "Acta.msg"},
		{"Acta.MSG", domain.KindEmbeddedMsg, 2, "Acta.MSG"},
		{"", domain.KindFile, 3, "adjunto_3"},
		{"  ", domain.KindEmbeddedMsg, 1, "adjunto_1.msg"},
		{"informe.pdf", domain.KindFile, 1, "informe.pdf"},
	}
	for _, c := range cases {
		if got := attachmentFileName(c.name, c.kind, c.index); got != c.want {
			t.Errorf("attachmentFileName(%q,%v,%d) = %q, want %q", c.name, c.kind, c.index, got, c.want)
		}
	}
}

func TestSenderText(t *testing.T) {
	cases := [][3]string{
		{"Ana", "ana@x.example", "Ana <ana@x.example>"},
		{"", "ana@x.example", "ana@x.example"},
		{"Ana", "", "Ana"},
		{"ana@x.example", "ana@x.example", "ana@x.example"},
		{"", "", ""},
	}
	for _, c := range cases {
		if got := senderText(c[0], c[1]); got != c[2] {
			t.Errorf("senderText(%q,%q) = %q, want %q", c[0], c[1], got, c[2])
		}
	}
}

func TestFolderPath(t *testing.T) {
	if got := folderPath(`\\demo@example.com\Bandeja de entrada`); got != `demo@example.com\Bandeja de entrada` {
		t.Fatalf("folderPath = %q", got)
	}
}

func TestConnectErrorWraps(t *testing.T) {
	err := connectError(io.EOF)
	if !errors.Is(err, domain.ErrOutlookUnavailable) || !strings.Contains(err.Error(), io.EOF.Error()) {
		t.Fatalf("connectError = %v", err)
	}
}

func TestIsMailClass(t *testing.T) {
	for class, want := range map[int]bool{43: true, 26: false, 46: false, 53: false} {
		if isMailClass(class) != want {
			t.Errorf("isMailClass(%d) != %v", class, want)
		}
	}
}
