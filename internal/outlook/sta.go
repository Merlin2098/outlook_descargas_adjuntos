//go:build windows

package outlook

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"

	ole "github.com/go-ole/go-ole"

	"outlook_descargas_adjuntos/internal/source"
)

var errSTAClosed = errors.New("la conexión con Outlook está cerrada")

// sta es el único goroutine que toca COM. Vive en un hilo del SO fijo (LockOSThread) inicializado
// como single-threaded apartment; el resto del programa le pasa funciones con do. Así ningún
// IDispatch cruza de hilo y nunca hay dos llamadas COM a la vez.
type sta struct {
	calls  chan staCall
	quit   chan struct{}
	closed chan struct{}
	once   sync.Once
}

type staCall struct {
	fn   func() error
	done chan error
}

// newSTA arranca el goroutine e inicializa COM en su hilo.
func newSTA() (*sta, error) {
	s := &sta{calls: make(chan staCall), quit: make(chan struct{}), closed: make(chan struct{})}
	ready := make(chan error, 1)
	go s.loop(ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return s, nil
}

func (s *sta) loop(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(s.closed)
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		// S_FALSE (1): el hilo ya tenía COM inicializado; sigue siendo válido.
		var oe *ole.OleError
		if !errors.As(err, &oe) || oe.Code() != 1 {
			ready <- fmt.Errorf("CoInitializeEx: %w", err)
			return
		}
	}
	defer ole.CoUninitialize()
	ready <- nil
	for {
		select {
		case c := <-s.calls:
			c.done <- runSafe(c.fn)
		case <-s.quit:
			return
		}
	}
}

// runSafe convierte un panic dentro de fn en error para que el STA siga vivo.
func runSafe(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error interno en COM: %v", r)
		}
	}()
	return fn()
}

// do corre fn en el STA y espera su resultado. Si ctx ya terminó antes de empezar, no la corre.
// Una vez empezada, espera a que termine aunque ctx se cancele: fn debe revisar ctx si es larga.
func (s *sta) do(ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c := staCall{fn: fn, done: make(chan error, 1)}
	select {
	case s.calls <- c:
		return <-c.done
	case <-s.closed:
		return errSTAClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

// close termina el goroutine y libera COM en su hilo. Es idempotente.
func (s *sta) close() {
	s.once.Do(func() { close(s.quit) })
	<-s.closed
}

// stream corre produce en el STA y entrega cada item al goroutine que llama, que corre fn.
// fn nunca corre dentro del STA, así puede hacer otras llamadas a la fuente sin bloquearse.
// yield devuelve false cuando hay que parar: fn devolvió source.ErrStopScan u otro error, o ctx terminó.
func stream[T any](ctx context.Context, s *sta, produce func(yield func(T) bool) error, fn func(T) error) error {
	items := make(chan T)
	stopped := make(chan struct{})
	var stopOnce sync.Once
	stop := func() { stopOnce.Do(func() { close(stopped) }) }
	produced := make(chan error, 1)

	go func() {
		produced <- s.do(ctx, func() error {
			defer close(items)
			return produce(func(v T) bool {
				select {
				case items <- v:
					return true
				case <-stopped:
					return false
				case <-ctx.Done():
					return false
				}
			})
		})
	}()

	var result error
	byUs := false // paramos nosotros: lo que devuelva produce ya no importa
loop:
	for {
		select {
		case v, ok := <-items:
			if !ok {
				break loop
			}
			if err := fn(v); err != nil {
				byUs = true
				if !errors.Is(err, source.ErrStopScan) {
					result = err
				}
				break loop
			}
			if err := ctx.Err(); err != nil {
				byUs, result = true, err
				break loop
			}
		case <-ctx.Done():
			byUs, result = true, ctx.Err()
			break loop
		}
	}
	stop()
	perr := <-produced // espera a que el STA quede libre
	if !byUs {
		result = perr
	}
	return result
}
