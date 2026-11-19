package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"outlook_descargas_adjuntos/internal/config"
	"outlook_descargas_adjuntos/internal/domain"
	"outlook_descargas_adjuntos/internal/jobs"
	"outlook_descargas_adjuntos/internal/source"
	"outlook_descargas_adjuntos/internal/winapi"
)

// eventInterval limita los eventos de progreso y actividad a unos 10 por segundo.
const eventInterval = 100 * time.Millisecond

// App expone a la UI los métodos que Wails convierte en bindings de TypeScript.
// Solo traduce DTOs y delega; la lógica vive en internal/.
type App struct {
	ctx       context.Context
	src       source.MailSource
	jobs      *jobs.Manager
	log       *slog.Logger
	configDir string

	emit jobs.Emitter // eventos de la app hacia la UI (fuera de los del job)

	mu      sync.Mutex // protege cfg y closeOK: Wails llama a los bindings desde varios goroutines
	cfg     config.Config
	closeOK bool
}

// EventCloseRequested pide a la UI confirmar el cierre porque hay una descarga en curso.
const EventCloseRequested = "app:closeRequested"

// wailsEmitter publica los eventos del gestor de jobs en el runtime de Wails.
type wailsEmitter struct{ app *App }

func (e wailsEmitter) Emit(event string, payload any) {
	if e.app.ctx != nil {
		runtime.EventsEmit(e.app.ctx, event, payload)
		notifyTerminal(event, windowNotifier{})
	}
}

// appTitle es el título de la ventana; FindOwnWindow lo usa para hacerla parpadear.
const appTitle = "outlook_descargas_adjuntos"

// notifier avisa al usuario que un job terminó.
type notifier interface {
	Beep()
	Flash()
}

// windowNotifier suena siempre y hace parpadear la ventana solo si no está en primer plano.
type windowNotifier struct{}

func (windowNotifier) Beep() { winapi.Beep() }

func (windowNotifier) Flash() {
	if h := winapi.FindOwnWindow(appTitle); h != 0 && !winapi.IsForeground(h) {
		winapi.FlashWindow(h)
	}
}

// notifyTerminal avisa (beep y parpadeo) cuando llega un evento terminal.
func notifyTerminal(event string, n notifier) {
	switch event {
	case jobs.EventDone, jobs.EventError, jobs.EventCancelled:
		n.Beep()
		n.Flash()
	}
}

// NewApp crea la app. src puede ser nil si no hay fuente disponible.
func NewApp(src source.MailSource, log *slog.Logger, configDir string, cfg config.Config) *App {
	a := &App{src: src, log: log, configDir: configDir, cfg: cfg}
	a.emit = wailsEmitter{a}
	if src != nil {
		a.jobs = jobs.NewManagerWith(src, wailsEmitter{a}, eventInterval, log, jobs.Options{Begin: winapi.KeepAwake})
	}
	return a
}

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// beforeClose (OnBeforeClose de Wails) impide cerrar con una descarga en curso y pide a la UI
// que lo confirme. Cubre Alt+F4 y el cierre desde la barra de tareas.
func (a *App) beforeClose(context.Context) (prevent bool) {
	a.mu.Lock()
	ok := a.closeOK
	a.mu.Unlock()
	if ok || a.jobs == nil || !a.jobs.Running() {
		return false
	}
	a.emit.Emit(EventCloseRequested, nil)
	return true
}

// ConfirmClose lo llama la UI cuando el usuario confirma salir: cancela la descarga y cierra.
// shutdown espera a que el job escriba su reporte parcial.
func (a *App) ConfirmClose() {
	a.mu.Lock()
	a.closeOK = true
	a.mu.Unlock()
	if a.jobs != nil {
		a.jobs.CancelActive()
	}
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

// shutdownWait es cuánto se espera, al cerrar, a que un job cancelado escriba su reporte parcial.
const shutdownWait = 5 * time.Second

func (a *App) shutdown(context.Context) {
	if a.jobs == nil {
		return
	}
	a.jobs.CancelActive()
	if !a.jobs.Wait(shutdownWait) {
		a.log.Warn("el job no terminó a tiempo al cerrar la app")
	}
	if c, ok := a.src.(io.Closer); ok {
		if err := c.Close(); err != nil {
			a.log.Warn("no se pudo cerrar la fuente de correos", "error", err)
		}
	}
}

var errNoSource = errors.New("No hay una fuente de correos disponible") //nolint:staticcheck // texto visible

// SourceInfo indica qué fuente está activa ("demo", "outlook" o "ninguna").
func (a *App) SourceInfo() SourceInfo {
	if a.src == nil {
		return SourceInfo{Name: "ninguna", Version: appVersion}
	}
	return SourceInfo{Name: a.src.Name(), Demo: a.src.Name() == "demo", Version: appVersion}
}

// ListRoots devuelve las cuentas o stores de la fuente.
func (a *App) ListRoots() ([]domain.Folder, error) {
	if a.src == nil {
		return nil, errNoSource
	}
	return a.src.ListRoots(a.ctx)
}

// ListChildren devuelve las subcarpetas de una carpeta.
func (a *App) ListChildren(ref domain.FolderRef) ([]domain.Folder, error) {
	if a.src == nil {
		return nil, errNoSource
	}
	return a.src.ListChildren(a.ctx, ref)
}

// PickDirectory abre el diálogo nativo para elegir la carpeta de destino.
func (a *App) PickDirectory() (string, error) {
	a.mu.Lock()
	last := a.cfg.LastDestDir
	a.mu.Unlock()
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Carpeta de destino",
		DefaultDirectory: last,
	})
}

// StartJob valida y lanza la descarga; el progreso llega por eventos job:*.
func (a *App) StartJob(req JobRequestDTO) (string, error) {
	if a.jobs == nil {
		return "", errNoSource
	}
	r, err := req.ToDomain()
	if err != nil {
		return "", err
	}
	id, err := a.jobs.Start(r)
	if err != nil {
		return "", err
	}
	a.mu.Lock()
	a.cfg.LastDestDir, a.cfg.Organize, a.cfg.SkipInline = r.DestDir, r.Organize, r.Filters.SkipInline
	cfg := a.cfg
	a.mu.Unlock()
	if err := config.Save(a.configDir, cfg); err != nil {
		a.log.Warn("no se pudo guardar la config", "error", err)
	}
	return id, nil
}

// GetJobStatus devuelve el estado del job en curso o del último (la UI lo usa al recargar).
func (a *App) GetJobStatus() jobs.Status {
	if a.jobs == nil {
		return jobs.Status{}
	}
	return a.jobs.Status()
}

// CancelJob cancela la descarga en curso.
func (a *App) CancelJob(jobID string) error {
	if a.jobs == nil {
		return errNoSource
	}
	return a.jobs.Cancel(jobID)
}

// openTarget elige qué ruta del último job abrir: el reporte o la carpeta de destino.
// Solo se abren rutas que produjo un job y que existen en disco.
func openTarget(st jobs.Status, report bool) (string, error) {
	path, errNone := st.DestDir, domain.ErrNothingToOpen
	if report {
		path, errNone = "", domain.ErrNoReport
		if st.Last != nil {
			path = st.Last.ReportPath
		}
	}
	if path == "" {
		return "", errNone
	}
	if _, err := os.Stat(path); err != nil {
		return "", errNone
	}
	return path, nil
}

func (a *App) open(report bool) error {
	path, err := openTarget(a.GetJobStatus(), report)
	if err != nil {
		return err
	}
	return winapi.OpenPath(path)
}

// OpenDestination abre en el Explorador la carpeta de destino del último job.
func (a *App) OpenDestination() error { return a.open(false) }

// OpenReport abre el reporte Excel del último job.
func (a *App) OpenReport() error { return a.open(true) }

// GetConfig devuelve las preferencias actuales.
func (a *App) GetConfig() config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg
}

// SaveConfig guarda las preferencias.
func (a *App) SaveConfig(c config.Config) error {
	a.mu.Lock()
	a.cfg = c
	a.mu.Unlock()
	return config.Save(a.configDir, c)
}
