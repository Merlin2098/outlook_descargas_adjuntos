package extract

import (
	"testing"
	"time"

	"outlook_descargas_adjuntos/internal/domain"
)

func at(y int, m time.Month, d, hh, mm, ss int) time.Time {
	return time.Date(y, m, d, hh, mm, ss, 0, time.Local)
}

func TestCheckRange(t *testing.T) {
	f := domain.Filters{From: at(2026, 8, 1, 0, 0, 0), To: at(2026, 10, 31, 0, 0, 0)}
	start, end := dayBounds(f)
	cases := []struct {
		t    time.Time
		want rangeCheck
	}{
		{at(2026, 8, 1, 0, 0, 0), inRange},
		{at(2026, 10, 31, 23, 59, 59), inRange},
		{at(2026, 11, 1, 0, 0, 0), afterRange},
		{at(2026, 7, 31, 23, 59, 59), beforeRange},
	}
	for _, c := range cases {
		if got := checkRange(c.t, start, end); got != c.want {
			t.Errorf("checkRange(%v) = %v, want %v", c.t, got, c.want)
		}
	}
}

func att(i int, name string, k domain.AttachmentKind) domain.Attachment {
	return domain.Attachment{Index: i, FileName: name, Size: 10, Kind: k}
}

func TestEvaluate(t *testing.T) {
	file := domain.KindFile
	mail := func(subject, sender string, atts ...domain.Attachment) domain.Mail {
		return domain.Mail{Subject: subject, Sender: sender, IsMail: true, Attachments: atts}
	}
	cases := []struct {
		name       string
		m          domain.Mail
		f          domain.Filters
		wantReason string
		wantNames  []string
	}{
		{"no es correo", domain.Mail{IsMail: false}, domain.Filters{}, domain.ReasonNotMail, nil},
		{"asunto", mail("Hola", "a@b", att(1, "f.pdf", file)), domain.Filters{SubjectContains: "factura"}, domain.ReasonSubject, nil},
		{"remitente", mail("Factura", "a@b", att(1, "f.pdf", file)), domain.Filters{SenderContains: "@prov"}, domain.ReasonSender, nil},
		{"sin adjuntos", mail("Factura", "a@b"), domain.Filters{}, domain.ReasonNoAttachments, nil},
		{"solo inline", mail("x", "a@b", att(1, "image001.png", domain.KindInline)), domain.Filters{SkipInline: true}, domain.ReasonOnlyInline, nil},
		{"tipo no soportado", mail("x", "a@b", att(1, "ole", domain.KindOther)), domain.Filters{}, domain.ReasonUnsupportedKind, nil},
		{"extensiones", mail("x", "a@b", att(1, "c.docx", file)), domain.Filters{Extensions: []string{".pdf"}}, domain.ReasonNoMatchingExt, nil},
		{"asunto sin mayúsculas", mail("FACTURA F001", "a@b", att(1, "f.pdf", file)), domain.Filters{SubjectContains: "factura"}, "", []string{"f.pdf"}},
		{"extensión en mayúsculas", mail("x", "a@b", att(1, "f.PDF", file)), domain.Filters{Extensions: []string{".pdf"}}, "", []string{"f.PDF"}},
		{"inline descartado", mail("x", "a@b", att(1, "image001.png", domain.KindInline), att(2, "f.pdf", file)), domain.Filters{SkipInline: true}, "", []string{"f.pdf"}},
		{"msg embebido", mail("x", "a@b", att(1, "reunion.msg", domain.KindEmbeddedMsg)), domain.Filters{Extensions: []string{".msg"}}, "", []string{"reunion.msg"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, reason := evaluate(c.m, c.f)
			if reason != c.wantReason {
				t.Fatalf("reason = %q, want %q", reason, c.wantReason)
			}
			var names []string
			for _, a := range got {
				names = append(names, a.FileName)
			}
			if len(names) != len(c.wantNames) {
				t.Fatalf("elegibles = %v, want %v", names, c.wantNames)
			}
			for i := range names {
				if names[i] != c.wantNames[i] {
					t.Fatalf("elegibles = %v, want %v", names, c.wantNames)
				}
			}
		})
	}
}
