//go:build windows

// Package winapi envuelve las pocas llamadas Win32 que necesita la app: evitar la suspensión
// durante un job, avisar al terminar (beep y parpadeo) y abrir rutas con el shell.
package winapi

import (
	"os"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	user32   = windows.NewLazySystemDLL("user32.dll")

	procSetThreadExecutionState = kernel32.NewProc("SetThreadExecutionState")
	procMessageBeep             = user32.NewProc("MessageBeep")
	procGetForegroundWindow     = user32.NewProc("GetForegroundWindow")
	procFlashWindowEx           = user32.NewProc("FlashWindowEx")
	procEnumWindows             = user32.NewProc("EnumWindows")
	procGetWindowTextW          = user32.NewProc("GetWindowTextW")
)

// Flags de SetThreadExecutionState.
const (
	esContinuous      = 0x80000000
	esSystemRequired  = 0x00000001
	esDisplayRequired = 0x00000002
)

const (
	mbIconAsterisk  = 0x40
	flashwAll       = 0x3
	flashwTimerNoFG = 0xC
	swShowNormal    = 1
)

// setThreadExecutionState es una variable para poder sustituirla en los tests.
var setThreadExecutionState = func(flags uint32) uint32 {
	r, _, _ := procSetThreadExecutionState.Call(uintptr(flags))
	return uint32(r)
}

// KeepAwake pide al sistema no suspender ni apagar la pantalla hasta que se llame a la función
// devuelta (idempotente). SetThreadExecutionState es por hilo: un goroutine con LockOSThread
// activa el estado y lo restaura en el mismo hilo.
func KeepAwake() (release func()) {
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(done)
		setThreadExecutionState(esContinuous | esSystemRequired | esDisplayRequired)
		<-stop
		setThreadExecutionState(esContinuous)
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(stop)
			<-done
		})
	}
}

// Beep reproduce el sonido de información del sistema.
func Beep() {
	procMessageBeep.Call(mbIconAsterisk)
}

// FindOwnWindow devuelve la ventana de nivel superior de este proceso con ese título, o 0.
// A diferencia de FindWindowW, no confunde la app con otra ventana que tenga el mismo título.
func FindOwnWindow(title string) uintptr {
	pid := windows.GetCurrentProcessId()
	var found uintptr
	buf := make([]uint16, 512)
	cb := windows.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var owner uint32
		if _, err := windows.GetWindowThreadProcessId(windows.HWND(hwnd), &owner); err != nil || owner != pid {
			return 1
		}
		n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if windows.UTF16ToString(buf[:n]) == title {
			found = hwnd
			return 0 // detiene la enumeración
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	return found
}

// IsForeground indica si hwnd es la ventana en primer plano.
func IsForeground(hwnd uintptr) bool {
	h, _, _ := procGetForegroundWindow.Call()
	return hwnd != 0 && h == hwnd
}

// flashWInfo es FLASHWINFO de user32.
type flashWInfo struct {
	cbSize    uint32
	hwnd      uintptr
	dwFlags   uint32
	uCount    uint32
	dwTimeout uint32
}

// FlashWindow hace parpadear la ventana en la barra de tareas hasta que pase a primer plano.
func FlashWindow(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	fi := flashWInfo{hwnd: hwnd, dwFlags: flashwAll | flashwTimerNoFG}
	fi.cbSize = uint32(unsafe.Sizeof(fi))
	procFlashWindowEx.Call(uintptr(unsafe.Pointer(&fi)))
}

// OpenPath abre una carpeta o un archivo con su aplicación asociada (ShellExecute "open").
// A diferencia de explorer.exe, no reinterpreta las comas de la ruta. ShellExecute puede usar COM
// (extensiones del shell), así que corre en un hilo propio con COM inicializado como pide su documentación.
func OpenPath(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		// S_FALSE (el hilo ya tenía COM) también exige su CoUninitialize; RPC_E_CHANGED_MODE no.
		if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE); err == nil || err == windows.Errno(1) {
			defer windows.CoUninitialize()
		}
		done <- windows.ShellExecute(0, verb, file, nil, nil, swShowNormal)
	}()
	return <-done
}
