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

// Resultado de un job, igual al sufijo de su evento terminal.
const (
	OutcomeDone      = "done"
	OutcomeCancelled = "cancelled"
	OutcomeError     = "error"
)

// Terminal es el payload de job:done, job:error y job:cancelled.
// Outcome repite el resultado del evento para quien lo lea después con Status (p. ej. tras recargar la UI).
type Terminal struct {
	JobID      string         `json:"jobId"`
	Outcome    string         `json:"outcome"`
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

	// Estado consultable con Status (protegido por mu).
	statusID string
	dest     string
	prog     extract.Progress
	recent   []extract.Activity
	last     *Terminal
}

// RecentMax es cuántas actividades recientes conserva Status.
const RecentMax = 100

// Status es la foto del job que la UI consulta con GetJobStatus (por ejemplo, tras recargar).
type Status struct {
	JobID    string             `json:"jobId"` // job en curso, o el último terminado
	Running  bool               `json:"running"`
	DestDir  string             `json:"destDir"`  // destino del job en curso o del último
	Progress extract.Progress   `json:"progress"` // último progreso publicado
	Recent   []extract.Activity `json:"recent"`   // últimas RecentMax actividades, la más antigua primero
	Last     *Terminal          `json:"last"`     // terminal del último job terminado; nil si no hay
}

// Status devuelve una copia del estado actual.
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := Status{JobID: m.statusID, Running: m.cancel != nil, DestDir: m.dest, Progress: m.prog,
		Recent: append([]extract.Activity(nil), m.recent...)}
	if m.last != nil {
		t := *m.last
		st.Last = &t
	}
	return st
}

// published guarda lo que el sink acaba de publicar para el job id.
func (m *Manager) published(id string, prog *extract.Progress, acts []extract.Activity) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id != m.statusID {
		return
	}
	if prog != nil {
		m.prog = *prog
	}
	m.recent = append(m.recent, acts...)
	if n := len(m.recent); n > RecentMax {
		m.recent = append([]extract.Activity(nil), m.recent[n-RecentMax:]...)
	}
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
	m.statusID, m.dest, m.prog, m.recent = id, req.DestDir, extract.Progress{}, nil
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

// finish libera el gestor y guarda el terminal del job, antes de emitirlo.
func (m *Manager) finish(term Terminal) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
	}
	m.id, m.cancel = "", nil
	m.last = &term
}

func (m *Manager) run(ctx context.Context, id string, req domain.JobRequest, done chan struct{}) {
	defer close(done)
	sink := newThrottledSink(m.em, m.interval, func(p *extract.Progress, acts []extract.Activity) {
		m.published(id, p, acts)
	})
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

	term := Terminal{JobID: id, Summary: res.Summary, ReportPath: res.ReportPath}
	event := EventDone
	term.Outcome = OutcomeDone
	switch {
	case err == nil:
		log.Info("job terminado", "guardados", res.Summary.Saved, "errores", res.Summary.Errors)
	case errors.Is(err, context.Canceled):
		event, term.Outcome = EventCancelled, OutcomeCancelled
		log.Info("job cancelado", "guardados", res.Summary.Saved)
	default:
		event, term.Outcome = EventError, OutcomeError
		term.Message = err.Error()
		log.Error("job con error", "error", err)
	}
	m.finish(term)
	m.em.Emit(event, term)
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// throttledSink acumula progreso y actividad y los publica como mucho una vez por intervalo.
type throttledSink struct {
	em     Emitter
	onPub  func(*extract.Progress, []extract.Activity)
	mu     sync.Mutex
	last   extract.Progress
	dirty  bool
	acts   []extract.Activity
	stop   chan struct{}
	done   chan struct{}
	ticker *time.Ticker
}

// onPub (opcional) recibe lo publicado en cada flush; prog es nil si no hubo progreso nuevo.
func newThrottledSink(em Emitter, interval time.Duration, onPub func(*extract.Progress, []extract.Activity)) *throttledSink {
	s := &throttledSink{em: em, onPub: onPub, stop: make(chan struct{}), done: make(chan struct{}), ticker: time.NewTicker(interval)}
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
	if s.onPub != nil && (dirty || len(acts) > 0) {
		var p *extract.Progress
		if dirty {
			p = &prog
		}
		s.onPub(p, acts)
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
