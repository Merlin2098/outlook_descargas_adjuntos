package jobs

import (
	"sync"
	"testing"
	"time"

	"outlook_descargas_adjuntos/internal/source"
)

// hookCounter cuenta Begin/end y registra si end corrió después del terminal.
type hookCounter struct {
	mu            sync.Mutex
	begins, ends  int
	terminalSeen  bool
	endAfterFinal bool
}

func (h *hookCounter) begin() func() {
	h.mu.Lock()
	h.begins++
	h.mu.Unlock()
	return func() {
		h.mu.Lock()
		h.ends++
		h.endAfterFinal = h.terminalSeen
		h.mu.Unlock()
	}
}

func (h *hookCounter) snapshot() (int, int, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.begins, h.ends, h.endAfterFinal
}

func runWithHook(t *testing.T, src source.MailSource, cancel bool) (*hookCounter, event) {
	t.Helper()
	h := &hookCounter{}
	em := newEmitter()
	em.onEvent = func(name string) {
		if name == EventDone || name == EventError || name == EventCancelled {
			h.mu.Lock()
			h.terminalSeen = true
			h.mu.Unlock()
		}
	}
	m := NewManagerWith(src, em, 10*time.Millisecond, quietLog(), Options{Begin: h.begin})
	id, err := m.Start(facturasReq(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if cancel {
		_ = m.Cancel(id)
	}
	e := em.waitTerminal(t)
	if !m.Wait(10 * time.Second) {
		t.Fatal("el job no terminó")
	}
	return h, e
}

func TestBeginEndOnce(t *testing.T) {
	h, e := runWithHook(t, demoSrc(), false)
	b, en, after := h.snapshot()
	if e.name != EventDone || b != 1 || en != 1 || !after {
		t.Fatalf("evento %s, begins %d, ends %d, end después del terminal %v", e.name, b, en, after)
	}
}

func TestBeginEndOnPanic(t *testing.T) {
	h, e := runWithHook(t, panicSource{demoSrc()}, false)
	if _, en, _ := h.snapshot(); e.name != EventError || en != 1 {
		t.Fatalf("evento %s, ends %d", e.name, en)
	}
}

func TestBeginEndOnCancel(t *testing.T) {
	h, e := runWithHook(t, demoSrc(), true)
	if _, en, _ := h.snapshot(); e.name != EventCancelled || en != 1 {
		t.Fatalf("evento %s, ends %d", e.name, en)
	}
}
