package jobs

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"outlook_descargas_adjuntos/internal/domain"
	"outlook_descargas_adjuntos/internal/extract"
	"outlook_descargas_adjuntos/internal/source"
	"outlook_descargas_adjuntos/internal/source/demo"
)

var testNow = time.Date(2026, 11, 12, 10, 0, 0, 0, time.Local)

type event struct {
	name    string
	payload any
}

type fakeEmitter struct {
	mu       sync.Mutex
	events   []event
	terminal chan event
	onEvent  func(string)
}

func newEmitter() *fakeEmitter { return &fakeEmitter{terminal: make(chan event, 4)} }

func (f *fakeEmitter) Emit(name string, payload any) {
	f.mu.Lock()
	f.events = append(f.events, event{name, payload})
	cb := f.onEvent
	f.mu.Unlock()
	if cb != nil {
		cb(name)
	}
	switch name {
	case EventDone, EventError, EventCancelled:
		f.terminal <- event{name, payload}
	}
}

func (f *fakeEmitter) snapshot() []event {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]event(nil), f.events...)
}

func (f *fakeEmitter) waitTerminal(t *testing.T) event {
	t.Helper()
	select {
	case e := <-f.terminal:
		return e
	case <-time.After(10 * time.Second):
		t.Fatal("no llegó ningún evento terminal")
		return event{}
	}
}

func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func demoSrc() *demo.Source {
	return demo.New(demo.DefaultFixture(testNow, 42), demo.Options{SaveDelay: 2 * time.Millisecond})
}

func facturasReq(t *testing.T, src source.MailSource) domain.JobRequest {
	t.Helper()
	roots, _ := src.ListRoots(context.Background())
	kids, _ := src.ListChildren(context.Background(), roots[0].Ref)
	var ref domain.FolderRef
	for _, k := range kids {
		if k.Name == "Facturas proveedores" {
			ref = k.Ref
		}
	}
	return domain.JobRequest{
		Folder:  ref,
		Filters: domain.Filters{From: testNow.AddDate(0, 0, -121), To: testNow, SkipInline: true},
		DestDir: t.TempDir(),
	}
}

func countTerminals(evs []event) int {
	n := 0
	for _, e := range evs {
		if e.name == EventDone || e.name == EventError || e.name == EventCancelled {
			n++
		}
	}
	return n
}

func TestStartDone(t *testing.T) {
	src := demoSrc()
	em := newEmitter()
	m := NewManager(src, em, 20*time.Millisecond, quietLog())
	if _, err := m.Start(facturasReq(t, src)); err != nil {
		t.Fatal(err)
	}
	term := em.waitTerminal(t)
	if term.name != EventDone {
		t.Fatalf("terminal = %s (%+v)", term.name, term.payload)
	}
	evs := em.snapshot()
	if countTerminals(evs) != 1 || evs[len(evs)-1].name != EventDone {
		t.Fatalf("el terminal no es único o no es el último")
	}
	var lastProg *extract.Progress
	for _, e := range evs {
		if e.name == EventProgress {
			p := e.payload.(extract.Progress)
			lastProg = &p
		}
	}
	if lastProg == nil || lastProg.Phase != extract.PhaseReporting {
		t.Fatalf("último progress = %+v", lastProg)
	}
	if m.Running() {
		t.Fatal("Running() sigue en true")
	}
}

func TestThrottle(t *testing.T) {
	src := demoSrc()
	em := newEmitter()
	m := NewManager(src, em, 50*time.Millisecond, quietLog())
	start := time.Now()
	if _, err := m.Start(facturasReq(t, src)); err != nil {
		t.Fatal(err)
	}
	em.waitTerminal(t)
	dur := time.Since(start)
	n := 0
	for _, e := range em.snapshot() {
		if e.name == EventProgress {
			n++
		}
	}
	if limit := 1 + int(dur/(50*time.Millisecond)) + 2; n >= limit {
		t.Fatalf("%d eventos de progreso en %v (límite %d)", n, dur, limit)
	}
}

func TestCancel(t *testing.T) {
	src := demoSrc()
	em := newEmitter()
	m := NewManager(src, em, 20*time.Millisecond, quietLog())
	var once sync.Once
	var id string
	var idMu sync.Mutex
	em.onEvent = func(name string) {
		if name == EventActivity {
			once.Do(func() {
				idMu.Lock()
				defer idMu.Unlock()
				go m.Cancel(id)
			})
		}
	}
	idMu.Lock()
	got, err := m.Start(facturasReq(t, src))
	id = got
	idMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	term := em.waitTerminal(t)
	if term.name != EventCancelled {
		t.Fatalf("terminal = %s", term.name)
	}
	tp := term.payload.(Terminal)
	if _, err := os.Stat(tp.ReportPath); err != nil {
		t.Fatalf("reporte parcial: %v", err)
	}
	if countTerminals(em.snapshot()) != 1 {
		t.Fatal("más de un terminal")
	}
	em.mu.Lock()
	em.onEvent = nil
	em.mu.Unlock()
	if _, err := m.Start(facturasReq(t, src)); err != nil {
		t.Fatalf("no acepta un job nuevo: %v", err)
	}
	em.waitTerminal(t)
}

func TestSecondStart(t *testing.T) {
	src := demoSrc()
	em := newEmitter()
	m := NewManager(src, em, 20*time.Millisecond, quietLog())
	if _, err := m.Start(facturasReq(t, src)); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Start(facturasReq(t, src)); !errors.Is(err, ErrJobRunning) {
		t.Fatalf("err = %v, want ErrJobRunning", err)
	}
	em.waitTerminal(t)
}

func TestInvalidRequest(t *testing.T) {
	src := demoSrc()
	em := newEmitter()
	m := NewManager(src, em, 20*time.Millisecond, quietLog())
	req := facturasReq(t, src)
	req.DestDir = ""
	// La validación es síncrona: el error vuelve en Start y no se emite ningún evento,
	// así la UI nunca recibe un terminal antes de conocer el jobID.
	if _, err := m.Start(req); !errors.Is(err, domain.ErrNoDest) {
		t.Fatalf("Start = %v, want ErrNoDest", err)
	}
	if len(em.snapshot()) != 0 || m.Running() {
		t.Fatalf("eventos = %d, Running = %v", len(em.snapshot()), m.Running())
	}
}

func TestWaitAfterCancel(t *testing.T) {
	src := demoSrc()
	em := newEmitter()
	m := NewManager(src, em, 20*time.Millisecond, quietLog())
	if !m.Wait(time.Second) {
		t.Fatal("Wait sin job debería volver true de inmediato")
	}
	if _, err := m.Start(facturasReq(t, src)); err != nil {
		t.Fatal(err)
	}
	m.CancelActive()
	if !m.Wait(5 * time.Second) {
		t.Fatal("Wait no esperó al job cancelado")
	}
	evs := em.snapshot()
	if countTerminals(evs) != 1 || evs[len(evs)-1].name != EventCancelled {
		t.Fatal("al volver Wait, el terminal job:cancelled ya debe haberse emitido")
	}
	tp := evs[len(evs)-1].payload.(Terminal)
	if _, err := os.Stat(tp.ReportPath); err != nil {
		t.Fatalf("reporte parcial no escrito al volver Wait: %v", err)
	}
}

type panicSource struct{ *demo.Source }

func (panicSource) Scan(context.Context, domain.FolderRef, func(source.ScanItem) error) error {
	panic("boom")
}

func TestPanic(t *testing.T) {
	src := panicSource{demoSrc()}
	em := newEmitter()
	m := NewManager(src, em, 20*time.Millisecond, quietLog())
	if _, err := m.Start(facturasReq(t, src)); err != nil {
		t.Fatal(err)
	}
	term := em.waitTerminal(t)
	if term.name != EventError || m.Running() {
		t.Fatalf("terminal = %s, Running = %v", term.name, m.Running())
	}
	if countTerminals(em.snapshot()) != 1 {
		t.Fatal("más de un terminal")
	}
}

func TestCancelUnknown(t *testing.T) {
	m := NewManager(demoSrc(), newEmitter(), 20*time.Millisecond, quietLog())
	if err := m.Cancel("x"); !errors.Is(err, ErrUnknownJob) {
		t.Fatalf("err = %v", err)
	}
}
