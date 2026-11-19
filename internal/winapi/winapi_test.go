//go:build windows

package winapi

import (
	"path/filepath"
	"sync"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

type stateCall struct {
	flags  uint32
	thread uint32
}

func TestKeepAwakeSameThread(t *testing.T) {
	var mu sync.Mutex
	var calls []stateCall
	orig := setThreadExecutionState
	setThreadExecutionState = func(flags uint32) uint32 {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, stateCall{flags, windows.GetCurrentThreadId()})
		return 0
	}
	defer func() { setThreadExecutionState = orig }()

	release := KeepAwake()
	release()
	release()

	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 2 {
		t.Fatalf("llamadas = %d, want 2: %+v", len(calls), calls)
	}
	if calls[0].flags != esContinuous|esSystemRequired|esDisplayRequired || calls[1].flags != esContinuous {
		t.Fatalf("flags = %#x, %#x", calls[0].flags, calls[1].flags)
	}
	if calls[0].thread != calls[1].thread {
		t.Fatalf("activado en el hilo %d y restaurado en el %d", calls[0].thread, calls[1].thread)
	}
}

func TestFlashInfoSize(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("solo amd64/arm64")
	}
	if got := unsafe.Sizeof(flashWInfo{}); got != 32 {
		t.Fatalf("sizeof(FLASHWINFO) = %d, want 32", got)
	}
}

func TestOpenPathMissing(t *testing.T) {
	if err := OpenPath(filepath.Join(t.TempDir(), "no-existe")); err == nil {
		t.Fatal("OpenPath de una ruta inexistente no devolvió error")
	}
}
