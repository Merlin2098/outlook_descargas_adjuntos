# ADR 0001 — Go + Wails v2 + React para el rewrite

- **Estado:** aceptada (2026-11-12)

## Contexto

La herramienta original es una app de escritorio en Python (PySide6 + pywin32) empaquetada con PyInstaller.
Funciona, pero el paquete es pesado, el acceso a COM no libera objetos de forma determinista y no tiene tests.
El rewrite es además un proyecto de portafolio: tiene que mostrar un stack concreto con código limpio.

## Decisión

- **Backend en Go**: un binario único, concurrencia con goroutines y `context`, y acceso a COM con
  [go-ole](https://github.com/go-ole/go-ole) y a Win32 con `golang.org/x/sys/windows`.
- **Wails v2** como puente: la UI es web (WebView2), los métodos de [app.go](../../app.go) se exponen como
  bindings tipados y el progreso viaja por eventos.
- **React 18 + TypeScript + Vite** en [frontend/](../../frontend/), con la lógica pura en `src/lib/` probada con Vitest.
- El Wails CLI se fija en v2.17.1 (el v2.10 falla al generar bindings con Go 1.27).

## Consecuencias

- Binario de unos 16 MB que solo necesita WebView2, sin runtime de Python.
- Los tipos del backend llegan a TypeScript generados (`frontend/wailsjs/go/models.ts`), no escritos a mano.
- COM desde Go es de bajo nivel (sin early binding): exige el diseño del [ADR 0002](0002-com-en-goroutine-sta.md).
- Solo Windows, igual que el original: Outlook Classic solo existe ahí.
