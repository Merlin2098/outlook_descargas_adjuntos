//go:build windows

package outlook

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"

	"outlook_descargas_adjuntos/internal/source"
)

func newTestSTA(t *testing.T) *sta {
	t.Helper()
	s, err := newSTA()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.close)
	return s
}

func TestSTAThreadAffinity(t *testing.T) {
	s := newTestSTA(t)
	var mu sync.Mutex
	ids := map[uint32]bool{}
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 5; i++ {
				_ = s.do(context.Background(), func() error {
					mu.Lock()
					ids[windows.GetCurrentThreadId()] = true
					mu.Unlock()
					return nil
				})
			}
		}()
	}
	wg.Wait()
	if len(ids) != 1 {
		t.Fatalf("las llamadas corrieron en %d hilos distintos", len(ids))
	}
}

func TestSTAPanicRecovered(t *testing.T) {
	s := newTestSTA(t)
	var before, after uint32
	_ = s.do(context.Background(), func() error { before = windows.GetCurrentThreadId(); return nil })
	err := s.do(context.Background(), func() error { panic("x-boom") })
	if err == nil || !strings.Contains(err.Error(), "x-boom") {
		t.Fatalf("err = %v", err)
	}
	if err := s.do(context.Background(), func() error { after = windows.GetCurrentThreadId(); return nil }); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("el STA cambió de hilo tras el panic: %d → %d", before, after)
	}
}

func TestSTAClosed(t *testing.T) {
	s, err := newSTA()
	if err != nil {
		t.Fatal(err)
	}
	s.close()
	s.close()
	if err := s.do(context.Background(), func() error { return nil }); !errors.Is(err, errSTAClosed) {
		t.Fatalf("do tras close = %v", err)
	}
}

func TestStreamCancel(t *testing.T) {
	s := newTestSTA(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	produced := 0
	start := time.Now()
	err := stream(ctx, s, func(yield func(int) bool) error {
		for i := 0; ; i++ {
			produced++
			if !yield(i) {
				return nil
			}
		}
	}, func(v int) error {
		if v == 10 {
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("stream = %v, want context.Canceled", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("stream tardó %v en volver", time.Since(start))
	}
	ok := make(chan error, 1)
	go func() { ok <- s.do(context.Background(), func() error { return nil }) }()
	select {
	case err := <-ok:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("el STA quedó ocupado después de cancelar")
	}
	if produced > 13 {
		t.Fatalf("el productor siguió después de parar: %d items", produced)
	}
}

func TestStreamStop(t *testing.T) {
	s := newTestSTA(t)
	calls := 0
	err := stream(context.Background(), s, func(yield func(int) bool) error {
		for i := 0; i < 100; i++ {
			if !yield(i) {
				return nil
			}
		}
		return nil
	}, func(v int) error {
		calls++
		if calls == 5 {
			return source.ErrStopScan
		}
		return nil
	})
	if err != nil || calls != 5 {
		t.Fatalf("stream = %v, fn corrió %d veces", err, calls)
	}
}

func TestStreamProducerError(t *testing.T) {
	s := newTestSTA(t)
	boom := errors.New("boom")
	err := stream(context.Background(), s, func(yield func(int) bool) error {
		yield(1)
		return boom
	}, func(int) error { return nil })
	if !errors.Is(err, boom) {
		t.Fatalf("stream = %v, want boom", err)
	}
}

// refs devuelve el contador de referencias actual de d.
func refs(d *ole.IDispatch) int32 {
	d.AddRef()
	return d.Release()
}

func newDict(t *testing.T) *ole.IDispatch {
	t.Helper()
	unk, err := oleutil.CreateObject("Scripting.Dictionary")
	if err != nil {
		t.Fatalf("Scripting.Dictionary: %v", err)
	}
	defer unk.Release()
	d, err := unk.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestComHelpersReleaseAll(t *testing.T) {
	s := newTestSTA(t)
	err := s.do(context.Background(), func() error {
		outer, inner := newDict(t), newDict(t)
		defer release(outer, inner)
		if err := call(outer, "Add", "k", inner); err != nil {
			return err
		}
		if err := call(outer, "Add", "n", 7); err != nil {
			return err
		}
		if err := call(outer, "Add", "s", "hola"); err != nil {
			return err
		}
		if err := call(outer, "Add", "b", true); err != nil {
			return err
		}
		if n, err := getInt(outer, "Count"); err != nil || n != 4 {
			t.Errorf("Count = %d, %v", n, err)
		}
		if n, err := getInt(outer, "Item", "n"); err != nil || n != 7 {
			t.Errorf("Item(n) = %d, %v", n, err)
		}
		if v, err := getString(outer, "Item", "s"); err != nil || v != "hola" {
			t.Errorf("Item(s) = %q, %v", v, err)
		}
		if v, err := getBool(outer, "Item", "b"); err != nil || !v {
			t.Errorf("Item(b) = %v, %v", v, err)
		}
		before := refs(inner)
		for i := 0; i < 50; i++ {
			got, err := getObj(outer, "Item", "k")
			if err != nil {
				return err
			}
			release(got)
		}
		if after := refs(inner); after != before {
			t.Errorf("refcount de inner: antes %d, después %d (fuga de referencias)", before, after)
		}
		if _, err := getObj(outer, "Item", "n"); err == nil {
			t.Error("getObj de un valor que no es objeto no devolvió error")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// doWithin falla el test si una llamada al STA no vuelve en 1 s (el STA quedó ocupado).
func doWithin(t *testing.T, s *sta) error {
	t.Helper()
	ok := make(chan error, 1)
	go func() { ok <- s.do(context.Background(), func() error { return nil }) }()
	select {
	case err := <-ok:
		return err
	case <-time.After(time.Second):
		t.Fatal("el STA quedó ocupado")
		return nil
	}
}

func TestStreamAfterClose(t *testing.T) {
	s, err := newSTA()
	if err != nil {
		t.Fatal(err)
	}
	s.close()
	done := make(chan error, 1)
	go func() {
		done <- stream(context.Background(), s, func(yield func(int) bool) error {
			yield(1)
			return nil
		}, func(int) error { return nil })
	}()
	select {
	case err := <-done:
		if !errors.Is(err, errSTAClosed) {
			t.Fatalf("stream = %v, want errSTAClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stream se colgó con el STA cerrado")
	}
}

func TestStreamFnPanic(t *testing.T) {
	s := newTestSTA(t)
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("se esperaba el panic de fn")
			}
		}()
		_ = stream(context.Background(), s, func(yield func(int) bool) error {
			for i := 0; ; i++ {
				if !yield(i) {
					return nil
				}
			}
		}, func(v int) error {
			if v == 3 {
				panic("fn-boom")
			}
			return nil
		})
	}()
	if err := doWithin(t, s); err != nil {
		t.Fatal(err)
	}
}
