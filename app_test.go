package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"outlook_descargas_adjuntos/internal/config"
	"outlook_descargas_adjuntos/internal/domain"
	"outlook_descargas_adjuntos/internal/jobs"
	"outlook_descargas_adjuntos/internal/source/demo"
)

func testApp(t *testing.T) (*App, string) {
	t.Helper()
	src := demo.New(demo.DefaultFixture(time.Now(), 42), demo.Options{})
	dir := t.TempDir()
	return NewApp(src, slog.New(slog.NewTextHandler(io.Discard, nil)), dir, config.Default()), dir
}

func facturasDTO(t *testing.T, a *App) JobRequestDTO {
	t.Helper()
	roots, _ := a.src.ListRoots(context.Background())
	kids, _ := a.src.ListChildren(context.Background(), roots[0].Ref)
	for _, k := range kids {
		if k.Name == "Facturas proveedores" {
			today := time.Now().Format(dateLayout)
			return JobRequestDTO{Folder: k.Ref, From: time.Now().AddDate(0, 0, -30).Format(dateLayout), To: today,
				DestDir: t.TempDir(), Organize: "flat", SkipInline: false}
		}
	}
	t.Fatal("no se encontró la carpeta demo")
	return JobRequestDTO{}
}

func TestOpenTargets(t *testing.T) {
	dir := t.TempDir()
	report := filepath.Join(dir, "reporte.xlsx")
	if err := os.WriteFile(report, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		st      jobs.Status
		report  bool
		want    string
		wantErr error
	}{
		{"sin job, destino", jobs.Status{}, false, "", domain.ErrNothingToOpen},
		{"sin job, reporte", jobs.Status{}, true, "", domain.ErrNoReport},
		{"destino existente", jobs.Status{DestDir: dir}, false, dir, nil},
		{"destino inexistente", jobs.Status{DestDir: filepath.Join(dir, "no")}, false, "", domain.ErrNothingToOpen},
		{"reporte inexistente", jobs.Status{Last: &jobs.Terminal{ReportPath: filepath.Join(dir, "no.xlsx")}}, true, "", domain.ErrNoReport},
		{"reporte existente", jobs.Status{Last: &jobs.Terminal{ReportPath: report}}, true, report, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := openTarget(c.st, c.report)
			if !errors.Is(err, c.wantErr) || got != c.want {
				t.Fatalf("openTarget = %q, %v; want %q, %v", got, err, c.want, c.wantErr)
			}
		})
	}
}

func TestStartJobInvalidDoesNotSaveConfig(t *testing.T) {
	a, dir := testApp(t)
	req := facturasDTO(t, a)
	req.DestDir = ""
	if _, err := a.StartJob(req); err == nil {
		t.Fatal("StartJob con destino vacío no devolvió error")
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json")); !os.IsNotExist(err) {
		t.Fatalf("config.json existe tras un StartJob inválido (err = %v)", err)
	}
}

func TestStartJobSavesPreferences(t *testing.T) {
	a, dir := testApp(t)
	req := facturasDTO(t, a)
	if _, err := a.StartJob(req); err != nil {
		t.Fatal(err)
	}
	if !a.jobs.Wait(10 * time.Second) {
		t.Fatal("el job no terminó")
	}
	c, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.LastDestDir != req.DestDir || c.Organize != domain.OrganizeFlat || c.SkipInline {
		t.Fatalf("config guardada = %+v", c)
	}
}

func TestSourceInfoVersion(t *testing.T) {
	a, _ := testApp(t)
	if v := a.SourceInfo().Version; v != "0.2.0" {
		t.Fatalf("Version = %q", v)
	}
}

type fakeNotifier struct{ beeps, flashes int }

func (f *fakeNotifier) Beep()  { f.beeps++ }
func (f *fakeNotifier) Flash() { f.flashes++ }

func TestNotifyTerminal(t *testing.T) {
	cases := []struct {
		event string
		want  int
	}{
		{jobs.EventDone, 1}, {jobs.EventError, 1}, {jobs.EventCancelled, 1}, {jobs.EventProgress, 0}, {jobs.EventActivity, 0},
	}
	for _, c := range cases {
		n := &fakeNotifier{}
		notifyTerminal(c.event, n)
		if n.beeps != c.want || n.flashes != c.want {
			t.Errorf("%s: beeps %d flashes %d, want %d", c.event, n.beeps, n.flashes, c.want)
		}
	}
}

type recEmitter struct {
	mu     sync.Mutex
	events []string
}

func (r *recEmitter) Emit(name string, _ any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, name)
}

func (r *recEmitter) has(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.events {
		if e == name {
			return true
		}
	}
	return false
}

func TestBeforeCloseIdle(t *testing.T) {
	a, _ := testApp(t)
	rec := &recEmitter{}
	a.emit = rec
	if a.beforeClose(context.Background()) {
		t.Fatal("beforeClose sin job impidió cerrar")
	}
	if rec.has(EventCloseRequested) {
		t.Fatal("se emitió app:closeRequested sin job")
	}
}

func TestBeforeCloseRunning(t *testing.T) {
	src := demo.New(demo.DefaultFixture(time.Now(), 42), demo.Options{SaveDelay: 20 * time.Millisecond})
	a := NewApp(src, slog.New(slog.NewTextHandler(io.Discard, nil)), t.TempDir(), config.Default())
	rec := &recEmitter{}
	a.emit = rec
	if _, err := a.StartJob(facturasDTO(t, a)); err != nil {
		t.Fatal(err)
	}
	if !a.beforeClose(context.Background()) {
		t.Fatal("beforeClose con un job activo no impidió cerrar")
	}
	if !rec.has(EventCloseRequested) {
		t.Fatal("no se emitió app:closeRequested")
	}
	a.ConfirmClose()
	if a.beforeClose(context.Background()) {
		t.Fatal("tras ConfirmClose, beforeClose sigue impidiendo cerrar")
	}
	if !a.jobs.Wait(10 * time.Second) {
		t.Fatal("el job no terminó")
	}
	if last := a.jobs.Status().Last; last == nil || last.Outcome != jobs.OutcomeCancelled {
		t.Fatalf("Last = %+v, want cancelado", last)
	}
}
