package jobs

import (
	"testing"
	"time"

	"outlook_descargas_adjuntos/internal/source/demo"
)

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timeout esperando: %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestStatusIdle(t *testing.T) {
	m := NewManager(demoSrc(), newEmitter(), 10*time.Millisecond, quietLog())
	st := m.Status()
	if st.Running || st.JobID != "" || st.Last != nil || len(st.Recent) != 0 || st.DestDir != "" {
		t.Fatalf("Status inicial = %+v", st)
	}
}

func TestStatusRunning(t *testing.T) {
	src := demo.New(demo.DefaultFixture(testNow, 42), demo.Options{SaveDelay: 20 * time.Millisecond})
	m := NewManager(src, newEmitter(), 10*time.Millisecond, quietLog())
	req := facturasReq(t, src)
	id, err := m.Start(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = m.Cancel(id); m.Wait(10 * time.Second) }()
	waitFor(t, "progreso publicado", func() bool { return m.Status().Progress.Scanned > 0 })
	st := m.Status()
	if !st.Running || st.JobID != id || st.DestDir != req.DestDir {
		t.Fatalf("Status en curso = running %v id %q dest %q", st.Running, st.JobID, st.DestDir)
	}
}

func TestStatusAfterDone(t *testing.T) {
	src := demoSrc()
	em := newEmitter()
	m := NewManager(src, em, 10*time.Millisecond, quietLog())
	id, err := m.Start(facturasReq(t, src))
	if err != nil {
		t.Fatal(err)
	}
	em.waitTerminal(t)
	m.Wait(10 * time.Second)
	st := m.Status()
	if st.Running {
		t.Fatal("Running después de terminar")
	}
	if st.Last == nil || st.Last.JobID != id || st.Last.ReportPath == "" {
		t.Fatalf("Last = %+v", st.Last)
	}
	if len(st.Recent) == 0 || len(st.Recent) > RecentMax {
		t.Fatalf("len(Recent) = %d", len(st.Recent))
	}
}

func TestStatusLastSetBeforeTerminal(t *testing.T) {
	src := demoSrc()
	em := newEmitter()
	m := NewManager(src, em, 10*time.Millisecond, quietLog())
	var lastAtTerminal *Terminal
	em.onEvent = func(name string) {
		if name == EventDone {
			lastAtTerminal = m.Status().Last
		}
	}
	if _, err := m.Start(facturasReq(t, src)); err != nil {
		t.Fatal(err)
	}
	em.waitTerminal(t)
	if lastAtTerminal == nil {
		t.Fatal("Last no estaba fijado cuando se emitió el terminal")
	}
}

func TestStatusNewJobResetsRecent(t *testing.T) {
	src := demoSrc()
	em := newEmitter()
	m := NewManager(src, em, 10*time.Millisecond, quietLog())
	first, err := m.Start(facturasReq(t, src))
	if err != nil {
		t.Fatal(err)
	}
	em.waitTerminal(t)
	m.Wait(10 * time.Second)

	second, err := m.Start(facturasReq(t, src))
	if err != nil {
		t.Fatal(err)
	}
	st := m.Status()
	if st.JobID != second || !st.Running || st.Progress.Scanned != 0 || len(st.Recent) != 0 {
		t.Fatalf("Status tras el segundo Start = id %q running %v scanned %d recent %d",
			st.JobID, st.Running, st.Progress.Scanned, len(st.Recent))
	}
	if st.Last == nil || st.Last.JobID != first {
		t.Fatalf("Last = %+v, want el del primer job", st.Last)
	}
	_ = m.Cancel(second)
	em.waitTerminal(t)
	m.Wait(10 * time.Second)
}

func TestTerminalOutcome(t *testing.T) {
	src := demoSrc()
	em := newEmitter()
	m := NewManager(src, em, 10*time.Millisecond, quietLog())
	if _, err := m.Start(facturasReq(t, src)); err != nil {
		t.Fatal(err)
	}
	e := em.waitTerminal(t)
	m.Wait(10 * time.Second)
	if got := e.payload.(Terminal).Outcome; got != OutcomeDone {
		t.Fatalf("Outcome del evento = %q, want %q", got, OutcomeDone)
	}
	if got := m.Status().Last.Outcome; got != OutcomeDone {
		t.Fatalf("Status().Last.Outcome = %q", got)
	}

	id, err := m.Start(facturasReq(t, src))
	if err != nil {
		t.Fatal(err)
	}
	_ = m.Cancel(id)
	e = em.waitTerminal(t)
	m.Wait(10 * time.Second)
	if e.name != EventCancelled || e.payload.(Terminal).Outcome != OutcomeCancelled || m.Status().Last.Outcome != OutcomeCancelled {
		t.Fatalf("cancelado: evento %s, outcome %q, last %q", e.name, e.payload.(Terminal).Outcome, m.Status().Last.Outcome)
	}
}
