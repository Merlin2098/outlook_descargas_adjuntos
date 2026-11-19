package extract

import (
	"path/filepath"
	"strings"
	"time"

	"outlook_descargas_adjuntos/internal/domain"
)

type rangeCheck int

const (
	inRange rangeCheck = iota
	beforeRange
	afterRange
)

func (r rangeCheck) String() string {
	return [...]string{"inRange", "beforeRange", "afterRange"}[r]
}

// dayBounds devuelve [From 00:00:00, To 23:59:59.999999999] en time.Local.
func dayBounds(f domain.Filters) (start, end time.Time) {
	from := f.From.In(time.Local)
	to := f.To.In(time.Local)
	start = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.Local)
	end = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1).Add(-time.Nanosecond)
	return start, end
}

func checkRange(t, start, end time.Time) rangeCheck {
	switch {
	case t.Before(start):
		return beforeRange
	case t.After(end):
		return afterRange
	default:
		return inRange
	}
}

func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(strings.TrimSpace(sub)))
}

func extOf(a domain.Attachment) string {
	ext := strings.ToLower(filepath.Ext(a.FileName))
	if ext == "" && a.Kind == domain.KindEmbeddedMsg {
		return ".msg"
	}
	return ext
}

// evaluate decide qué adjuntos de un correo son elegibles. Si no queda ninguno, devuelve el motivo.
// El orden de las comprobaciones fija qué motivo se informa.
func evaluate(m domain.Mail, f domain.Filters) ([]domain.Attachment, string) {
	if !m.IsMail {
		return nil, domain.ReasonNotMail
	}
	if strings.TrimSpace(f.SubjectContains) != "" && !containsFold(m.Subject, f.SubjectContains) {
		return nil, domain.ReasonSubject
	}
	if strings.TrimSpace(f.SenderContains) != "" && !containsFold(m.Sender, f.SenderContains) {
		return nil, domain.ReasonSender
	}
	if len(m.Attachments) == 0 {
		return nil, domain.ReasonNoAttachments
	}

	var kept []domain.Attachment
	allInline := true
	for _, a := range m.Attachments {
		if a.Kind != domain.KindInline {
			allInline = false
		}
		if a.Kind == domain.KindOther || (f.SkipInline && a.Kind == domain.KindInline) {
			continue
		}
		kept = append(kept, a)
	}
	if len(kept) == 0 {
		if allInline {
			return nil, domain.ReasonOnlyInline
		}
		return nil, domain.ReasonUnsupportedKind
	}

	if len(f.Extensions) == 0 {
		return kept, ""
	}
	allowed := map[string]bool{}
	for _, e := range domain.NormalizeExtensions(f.Extensions) {
		allowed[e] = true
	}
	var out []domain.Attachment
	for _, a := range kept {
		if allowed[extOf(a)] {
			out = append(out, a)
		}
	}
	if len(out) == 0 {
		return nil, domain.ReasonNoMatchingExt
	}
	return out, ""
}
