package domain

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Validate comprueba la solicitud. Crea el destino si no existe y verifica que se pueda escribir en él.
func (r JobRequest) Validate() error {
	if r.Folder.EntryID == "" {
		return ErrNoFolder
	}
	if strings.TrimSpace(r.DestDir) == "" {
		return ErrNoDest
	}
	if dateOnly(r.Filters.From).After(dateOnly(r.Filters.To)) {
		return ErrBadRange
	}
	if st, err := os.Stat(r.DestDir); err == nil && !st.IsDir() {
		return ErrDestIsFile
	}
	if err := os.MkdirAll(r.DestDir, 0o755); err != nil {
		return fmt.Errorf("%w: %v", ErrDestNotWritable, err)
	}
	f, err := os.CreateTemp(r.DestDir, ".write-test-*")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDestNotWritable, err)
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return nil
}

func dateOnly(t time.Time) time.Time {
	l := t.In(time.Local)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.Local)
}

// NormalizeExtensions deja las extensiones en minúsculas, con punto, sin vacíos ni duplicados.
func NormalizeExtensions(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, e := range in {
		e = strings.ToLower(strings.TrimSpace(e))
		e = strings.TrimLeft(e, ".")
		if e == "" {
			continue
		}
		e = "." + e
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	return out
}
