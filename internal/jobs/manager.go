// Package jobs ejecuta una descarga a la vez y la comunica a la UI mediante eventos.
// Garantiza exactamente un evento terminal por job y limita la frecuencia del resto.
package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"outlook_descargas_adjuntos/internal/domain"
	"outlook_descargas_adjuntos/internal/extract"
	"outlook_descargas_adjuntos/internal/source"
)

// Nombres de los eventos que recibe la UI.
const (
	EventProgress  = "job:progress"
	EventActivity  = "job:activity"
	EventDone      = "job:done"
	EventError     = "job:error"
	EventCancelled = "job:cancelled"
)

// Emitter publica un evento hacia la UI (en producción, runtime.EventsEmit de Wails).
type Emitter interface {
	Emit(event string, payload any)
}

// Terminal es el payload de job:done, job:error y job:cancelled.
type Terminal struct {
	JobID      string         `json:"jobId"`
	Summary    domain.Summary `json:"summary"`
	ReportPath string         `json:"reportPath"`
	Message    string         `json:"message"`
}

//nolint:staticcheck // ST1005: mensajes visibles para el usuario.
var (
	ErrJobRunning = errors.New("Ya hay una descarga en curso")
	ErrUnknownJob = errors.New("Job desconocido")
)

// Manager ejecuta como máximo un job a la vez.
type Manager struct {
	src      source.MailSource
	em       Emitter
	interval time.Duration
	log      *slog.Logger

	mu     sync.Mutex
	id     string
	cancel context.CancelFunc
	done   chan struct{} // se cierra después del evento terminal del último job
}

// NewManager crea el gestor. interval es el mínimo entre eventos de progreso y de actividad.
func NewManager(src source.MailSource, em Emitter, interval time.Duration, log *slog.Logger) *Manager {
	return &Manager{src: src, em: em, interval: interval, log: log}
}

// Start valida la solicitud de forma síncrona, lanza el job en segundo plano y devuelve su id.
// Si la validación falla, el error vuelve aquí y no se emite ningún evento: así la UI nunca
// recibe un evento terminal de un job cuyo id todavía no conoce.
func (m *Manager) Start(req domain.JobRequest) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		return "", ErrJobRunning
	}
	if err := req.Validate(); err != nil {
		return "", err
	}
	id := newID()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	m.id, m.cancel, m.done = id, cancel, done
	go m.run(ctx, id, req, done)
	return id, nil
}

// Cancel pide cancelar el job indicado. El job termina con job:cancelled.
func (m *Manager) Cancel(jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel == nil || jobID != m.id {
		return ErrUnknownJob
	}
	m.cancel()
	return nil
}

// CancelActive cancela el job en curso, si lo hay (se usa al cerrar la app).
func (m *Manager) CancelActive() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
	}
}

// Running indica si hay un job en curso.
func (m *Manager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cancel != nil
}

func (m *Manager) finish() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
	}
	m.id, m.cancel = "", nil
}

func (m *Manager) run(ctx context.Context, id string, req domain.JobRequest, done chan struct{}) {
	defer close(done)
	sink := newThrottledSink(m.em, m.interval)
	log := m.log.With("job", id)
	log.Info("job iniciado", "carpeta", req.Folder.Path, "destino", req.DestDir)

	var res extract.Result
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("Error interno: %v", r)
			}
		}()
		res, err = extract.Run(ctx, m.src, req, sink, time.Now)
	}()

	sink.close()
	m.finish()

	term := Terminal{JobID: id, Summary: res.Summary, ReportPath: res.ReportPath}
	switch {
	case err == nil:
		log.Info("job terminado", "guardados", res.Summary.Saved, "errores", res.Summary.Errors)
		m.em.Emit(EventDone, term)
	case errors.Is(err, context.Canceled):
		log.Info("job cancelado", "guardados", res.Summary.Saved)
		m.em.Emit(EventCancelled, term)
	default:
		term.Message = err.Error()
		log.Error("job con error", "error", err)
		m.em.Emit(EventError, term)
	}
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// throttledSink acumula progreso y actividad y los publica como mucho una vez por intervalo.
type throttledSink struct {
	em     Emitter
	mu     sync.Mutex
	last   extract.Progress
	dirty  bool
	acts   []extract.Activity
	stop   chan struct{}
	done   chan struct{}
	ticker *time.Ticker
}

func newThrottledSink(em Emitter, interval time.Duration) *throttledSink {
	s := &throttledSink{em: em, stop: make(chan struct{}), done: make(chan struct{}), ticker: time.NewTicker(interval)}
	go func() {
		defer close(s.done)
		for {
			select {
			case <-s.ticker.C:
				s.flush()
			case <-s.stop:
				return
			}
		}
	}()
	return s
}

func (s *throttledSink) Progress(p extract.Progress) {
	s.mu.Lock()
	s.last, s.dirty = p, true
	s.mu.Unlock()
}

func (s *throttledSink) Activity(a extract.Activity) {
	s.mu.Lock()
	s.acts = append(s.acts, a)
	s.mu.Unlock()
}

func (s *throttledSink) flush() {
	s.mu.Lock()
	acts := s.acts
	s.acts = nil
	prog, dirty := s.last, s.dirty
	s.dirty = false
	s.mu.Unlock()
	if len(acts) > 0 {
		s.em.Emit(EventActivity, acts)
	}
	if dirty {
		s.em.Emit(EventProgress, prog)
	}
}

// close detiene el ticker y publica lo pendiente; después de close no se emite nada más.
func (s *throttledSink) close() {
	s.ticker.Stop()
	close(s.stop)
	<-s.done
	s.flush()
}

// Wait espera a que termine el job en curso (incluido su evento terminal y su reporte).
// Devuelve false si se agota timeout. Sin job en curso vuelve true de inmediato.
func (m *Manager) Wait(timeout time.Duration) bool {
	m.mu.Lock()
	done := m.done
	m.mu.Unlock()
	if done == nil {
		return true
	}
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}
