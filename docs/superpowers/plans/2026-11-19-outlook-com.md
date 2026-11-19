# Plan 3 — Outlook real (COM), operación de Windows y validación local: plan de implementación

> **Para agentes:** SUB-SKILL REQUERIDA: usa superpowers:subagent-driven-development (recomendado) o
> superpowers:executing-plans para ejecutar este plan tarea por tarea. Los pasos usan casillas (`- [ ]`).

**Objetivo:** que la app funcione contra Outlook Classic real detrás de la misma interfaz `MailSource`, con la operación
de Windows del MVP (no suspender el equipo, beep y parpadeo al terminar, confirmar el cierre con Alt+F4), un checklist de
validación local y las mejoras menores que quedaron del Plan 2.

**Arquitectura:** `internal/outlook` implementa `source.MailSource` con go-ole. Todo el acceso a COM pasa por un único
goroutine STA (`sta`), con `LockOSThread` + `CoInitializeEx(APARTMENTTHREADED)`. Los helpers de `com.go` liberan cada
`IDispatch` y cada `VARIANT`. `Scan` produce los elementos en el STA y los entrega por canal al goroutine del job, así
el callback de `extract` nunca corre dentro del STA. La conversión de fechas, la clasificación de adjuntos y los nombres
son funciones puras con tests. `internal/winapi` envuelve kernel32, user32 y shell32. `jobs.Manager` recibe un hook
`Begin` para mantener el equipo despierto, y `App` avisa al terminar y controla el cierre.

**Stack:** Go 1.27, `github.com/go-ole/go-ole` v1.3.0 (ya en `go.sum`), `golang.org/x/sys/windows` v0.49.0, Wails v2.17.1.

**Especificación:** [AGENTS.md](../../../AGENTS.md) ("Reglas críticas de COM / Outlook", "Reglas de Wails / concurrencia",
"Testing"), [docs/diagnostico-legacy.md](../../diagnostico-legacy.md) (§4.4 operación; §5 B4, B11, B12, B17, B18, B21;
§7 T1–T4; §8 riesgos 1–3) y [docs/demo-y-portafolio.md](../../demo-y-portafolio.md) §2.2.
Pendientes heredados: los "Deferred minors" del Plan 2 y, del Plan 1, el nombre de los `.msg` embebidos.

## Decisiones de este plan

| # | Decisión | Motivo |
| --- | --- | --- |
| C1 | Las fechas `VT_DATE` se convierten desde el `float64` OLE con una función propia (`oleDateToLocal`), sin `VARIANT.Value()` | go-ole v1.3.0 devuelve la hora de pared etiquetada como UTC y pierde los milisegundos (`variant_date_amd64.go`). Riesgo §8.3 |
| C2 | Los items se recorren con `Items.Sort("[ReceivedTime]", True)` + `GetFirst`/`GetNext`. Se mantiene el early exit del Plan 1 y no se usa `Restrict` (T2) | `Restrict` depende del locale; queda como mejora medible para el Plan 4 |
| C3 | `.msg` embebidos (Type 5) se guardan como `.msg` (T1). Subcarpetas opcionales (T4) ya existen en `extract` | Simple y predecible |
| C4 | Inline = `PR_ATTACHMENT_HIDDEN` verdadero, o `PR_ATTACH_CONTENT_ID` no vacío y referenciado como `cid:` en `HTMLBody` | Detecta imágenes de firma sin perder adjuntos reales con Content-ID |
| C5 | Remitente para filtrar y reportar: `"SenderName <SenderEmailAddress>"`, o solo uno de ellos si falta el otro | El filtro "remitente contiene" funciona por nombre o por dirección |
| C6 | `Mail.EntryID` es el EntryID de Outlook. El StoreID de cada correo se recuerda en un mapa dentro del STA para `GetItemFromID(entryID, storeID)` | No ensucia el reporte con IDs compuestos |
| C7 | Conexión perezosa: `GetActiveObject("Outlook.Application")` y, si falla, `CreateObject`. La app arranca sin esperar a Outlook | La UI muestra el error en el árbol si Outlook no está |
| C8 | El agente **no** ejecuta código contra el Outlook real del usuario. El spike y los tests `integration` los corre el usuario, y solo imprimen conteos y tiempos | AGENTS.md: datos reales nunca en contexto, logs ni capturas |
| C9 | Beep siempre al terminar; parpadeo solo si la ventana no está en primer plano (legacy §4.4) | Igual que el original |
| C10 | `OpenDestination`/`OpenReport` usan `ShellExecute` en lugar de `explorer.exe` | Corrige las rutas con comas (minor del Plan 2) |

## Restricciones globales

- `internal/outlook` y `internal/winapi` llevan `//go:build windows`. `extract`, `jobs` y `domain` no los importan.
- Ningún `*ole.IDispatch` ni `*ole.VARIANT` sale del goroutine STA. Todo `IDispatch` obtenido se libera con `Release` y todo `VARIANT` con `Clear`, incluso en los caminos de error.
- Item con `Class != 43` → `Mail.IsMail = false` (el pipeline lo registra como "No es un correo"). Un error al leer un item → `ScanItem.Err`, nunca aborta el job.
- Carpetas identificadas por `EntryID` + `StoreID`. `FolderRef.Path` es solo para mostrar.
- Fechas en `time.Local`. Mensajes visibles en español.
- Los tests de Outlook real llevan `//go:build windows && integration`, se saltan si Outlook no está y no imprimen asuntos, remitentes ni nombres de archivo.
- Ninguna dependencia nueva fuera de go-ole y x/sys (ambas ya en `go.sum`; pasan de `// indirect` a directas).
- `go test ./...`, `go vet ./...`, `go vet -tags integration ./...`, `cd frontend && npm test` y `npm run build` pasan al cierre de cada tarea que los toque.
- Cada commit termina con `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Foco de revisión

1. **Outlook ausente, cerrado o New Outlook**: `ListRoots` devuelve un error en español ("No se pudo conectar con Outlook Classic…"), la app no se cae y el árbol lo muestra (Tarea 6, `TestConnectError`).
2. **Cancelar mientras el STA recorre una carpeta grande**: `Scan` vuelve con `context.Canceled` sin esperar al final de la carpeta, el goroutine STA queda libre para la siguiente llamada y no se filtran goroutines (Tarea 5, `TestStreamCancel`).
3. **Afinidad de hilo**: todas las llamadas COM corren en el mismo hilo del SO, también después de un panic dentro de una función (Tarea 5, `TestSTAThreadAffinity`, `TestSTAPanicRecovered`).
4. **Suspensión**: `ES_CONTINUOUS` se restaura en el mismo hilo que la activó, también si el job se cancela o entra en panic (Tareas 1 y 2, `TestKeepAwakeSameThread`, `TestBeginEndOnPanic`).
5. **Fechas `VT_DATE` cerca de la medianoche y con milisegundos**: un correo recibido a las 23:59:59.900 hora local cae en ese día, no en el siguiente ni en UTC (Tarea 4, `TestOleDateToLocal`).

---

## Estructura de archivos

```text
internal/winapi/winapi.go(+test)      # KeepAwake, Beep, FlashWindow, FindWindow, IsForeground, OpenPath
internal/jobs/manager.go(+test)       # Options{Begin}, NewManagerWith
app.go / app_test.go / main.go        # notificar al terminar, OnBeforeClose, ConfirmClose, OpenX con winapi, Close de la fuente
internal/outlook/convert.go(+test)    # oleDateToLocal, classifyAttachment, attachmentFileName, senderText, isMailClass, folderPath, connectError
internal/outlook/sta.go(+test)        # goroutine STA: do, close, stream
internal/outlook/com.go(+test)        # helpers getObj/getString/getInt/getBool/getTime/call con Release/Clear
internal/outlook/outlook.go           # Source (MailSource) sobre Outlook
internal/outlook/outlook_integration_test.go  # //go:build windows && integration
cmd/spike-outlook/main.go             # mide listar + iterar N items (lo corre el usuario)
docs/validacion-outlook.md            # checklist manual
dto.go                                # selectSource: sin --demo → outlook.New()
frontend/src/...                      # textos P2, cierre por evento, minors del Plan 2
internal/extract/extract.go(+test)    # ReportPath vacío si falla el reporte
```

---

### Task 1: `internal/winapi`

**Archivos:**
- Crear: `internal/winapi/winapi.go`, `internal/winapi/winapi_test.go`

**Interfaces:**
- Produce:
  ```go
  // KeepAwake pide al sistema no suspender ni apagar la pantalla hasta que se llame a la función devuelta
  // (idempotente). Corre en un goroutine propio con LockOSThread: activa y restaura en el mismo hilo.
  func KeepAwake() (release func())
  func Beep()                                   // MessageBeep(MB_ICONASTERISK = 0x40)
  func FindWindow(title string) uintptr         // user32!FindWindowW(nil, title); 0 si no existe
  func IsForeground(hwnd uintptr) bool          // GetForegroundWindow() == hwnd
  func FlashWindow(hwnd uintptr)                // FlashWindowEx con FLASHW_ALL|FLASHW_TIMERNOFG (0x3|0xC), uCount 0
  func OpenPath(path string) error              // windows.ShellExecute(0, "open", path, nil, nil, SW_SHOWNORMAL)
  // Variable de prueba: la llamada real es kernel32!SetThreadExecutionState.
  var setThreadExecutionState = func(flags uint32) uint32
  const esContinuous, esSystemRequired, esDisplayRequired = 0x80000000, 0x00000001, 0x00000002
  ```

- [ ] **Paso 1: tests que fallan.**
  - `TestKeepAwakeSameThread`: reemplaza `setThreadExecutionState` por una función que registra `(flags, windows.GetCurrentThreadId())`. Tras `release()` hay exactamente dos llamadas: primero `esContinuous|esSystemRequired|esDisplayRequired` y después `esContinuous`, las dos con el mismo thread id. Un segundo `release()` no agrega llamadas.
  - `TestFlashInfoSize`: `unsafe.Sizeof(flashWInfo{}) == 32` en amd64 (`cbSize` debe valer eso).
  - `TestOpenPathMissing`: `OpenPath(filepath.Join(t.TempDir(), "no-existe"))` devuelve error.
- [ ] **Paso 2:** `go test ./internal/winapi/` → FAIL (paquete inexistente).
- [ ] **Paso 3:** implementar con `windows.NewLazySystemDLL` para kernel32 y user32. `OpenPath` comprueba antes con `os.Stat` y devuelve ese error.
- [ ] **Paso 4:** `go test ./internal/winapi/` → PASS. No hay test que abra el Explorador.
- [ ] **Paso 5:** commit `feat(winapi): suspensión, beep, parpadeo y apertura con ShellExecute`.

### Task 2: Equipo despierto durante el job y aviso al terminar

**Archivos:**
- Modificar: `internal/jobs/manager.go`, `app.go`, `main.go`
- Test: `internal/jobs/status_test.go` (agregar), `app_test.go` (agregar)
- Modificar textos: `frontend/src/i18n/es.ts`, `frontend/src/components/{ConfigScreen,ProgressScreen}.tsx`

**Interfaces:**
- Consume: `winapi.KeepAwake`, `winapi.Beep`, `winapi.FindWindow`, `winapi.IsForeground`, `winapi.FlashWindow`, `winapi.OpenPath` (Tarea 1).
- Produce:
  ```go
  // jobs
  type Options struct {
      // Begin corre al inicio de cada job, en el goroutine del job; la función que devuelve corre
      // después del evento terminal, también si hubo panic o cancelación. nil = sin hook.
      Begin func() (end func())
  }
  func NewManagerWith(src source.MailSource, em Emitter, interval time.Duration, log *slog.Logger, o Options) *Manager
  // NewManager queda como NewManagerWith(..., Options{}).

  // main (app.go)
  type notifier interface { Beep(); Flash() }        // implementación real: winapi + FindWindow(appTitle)
  func notifyTerminal(event string, n notifier)       // Beep en job:done|job:error|job:cancelled; Flash si la ventana no está al frente
  const appTitle = "outlook_descargas_adjuntos"       // también en options.App.Title
  ```

- [ ] **Paso 1: tests que fallan.**
  - `TestBeginEndOnce` (jobs): con `Begin` que cuenta inicios y finales, un job terminado deja `begins == 1`, `ends == 1`, y `end` corre **después** del evento terminal (lo registra un `fakeEmitter.onEvent`).
  - `TestBeginEndOnPanic` (jobs): con `panicSource` (ya existe en `manager_test.go`), `ends == 1` y llega `job:error`.
  - `TestBeginEndOnCancel` (jobs): cancelar deja `ends == 1`.
  - `TestNotifyTerminal` (main, tabla): `job:done`, `job:error` y `job:cancelled` → un `Beep`; `job:progress` → nada. `Flash` lo decide el notifier real; aquí el fake cuenta `Flash` y el test exige 1 por terminal.
- [ ] **Paso 2:** `go test ./internal/jobs/ . -run 'TestBeginEnd|TestNotifyTerminal'` → FAIL.
- [ ] **Paso 3:** en `run`, `if m.opts.Begin != nil { end := m.opts.Begin(); defer end() }` antes del `recover` y del terminal, de modo que `end` corre al final del goroutine. `wailsEmitter.Emit` llama a `notifyTerminal` después de `EventsEmit`. `App` crea el manager con `Begin: winapi.KeepAwake` (`func() func()` ya encaja). `OpenDestination`/`OpenReport` usan `winapi.OpenPath`, y se elimina `os/exec`.
- [ ] **Paso 4:** textos que el Plan 2 dejó fuera (P2), ahora verdaderos, en `es.ts`: `config.keepAwake` = "El equipo no entrará en suspensión mientras dure la descarga." (segunda línea del pie de Configurar), `progress.keepAwake` = "Suspensión del equipo bloqueada hasta que termine el job." (nota al pie del aside de En curso, con el icono de luna del mock) y `progress.minimizeHint` = "Puedes minimizar la ventana: te avisaremos con un sonido al terminar.".
- [ ] **Paso 5:** `go test ./... && go vet ./... && cd frontend && npm test && npm run build` → PASS.
- [ ] **Paso 6:** commit `feat: equipo despierto durante el job, beep y parpadeo al terminar`.

### Task 3: Confirmar el cierre con un job activo (Alt+F4 y barra de título)

**Archivos:**
- Modificar: `app.go`, `main.go`, `frontend/src/components/TitleBar.tsx`, `frontend/src/components/ConfirmDialog.tsx`
- Test: `app_test.go`
- Regenerar: `frontend/wailsjs/`

**Interfaces:**
- Produce:
  ```go
  const EventCloseRequested = "app:closeRequested"
  func (a *App) beforeClose(ctx context.Context) (prevent bool) // options.App.OnBeforeClose
  func (a *App) ConfirmClose()                                 // binding: permite cerrar, cancela el job y llama runtime.Quit
  ```
  `App` gana `emit jobs.Emitter` (por defecto `wailsEmitter{a}`; los tests ponen un fake) y `closeOK bool` bajo `a.mu`.

- [ ] **Paso 1: tests que fallan.**
  - `TestBeforeCloseIdle`: sin job, `beforeClose` → `false` y no emite nada.
  - `TestBeforeCloseRunning`: con la fuente demo lenta (`SaveDelay` de 20 ms) y un job en curso, `beforeClose` → `true` y el fake recibe `app:closeRequested`. Tras `ConfirmClose()` (con `a.ctx == nil` no llama a `Quit`), `beforeClose` → `false` y el job termina cancelado (`a.jobs.Wait(10*time.Second)` y `Status().Last.Outcome == "cancelled"`).
- [ ] **Paso 2:** `go test . -run TestBeforeClose` → FAIL.
- [ ] **Paso 3:** implementar. En la UI, `TitleBar` se suscribe a `app:closeRequested` (abre el mismo `ConfirmDialog`). Al confirmar, desde Alt+F4 o desde el botón, llama al binding `ConfirmClose()`. El botón cerrar con job activo abre el diálogo sin pasar por Go; sin job, `Quit()` directo. Se corrige de paso el minor del Plan 2: `ConfirmDialog` enfoca "Cancelar" una sola vez (efecto con `[]`) y lee `onCancel` desde una ref para el Escape.
- [ ] **Paso 4:** `.tools/wails.exe generate module`; `go test ./... && cd frontend && npm test && npm run build` → PASS.
- [ ] **Paso 5:** commit `feat: confirmar el cierre con una descarga en curso (Alt+F4 y barra de título)`.

### Task 4: Conversiones puras de Outlook

**Archivos:**
- Crear: `internal/outlook/convert.go`, `internal/outlook/convert_test.go`

**Interfaces:**
- Produce:
  ```go
  const olMail = 43
  // Attachment.Type de Outlook
  const olByValue, olByReference, olEmbeddedItem, olOLE = 1, 4, 5, 6
  func isMailClass(class int) bool
  func oleDateToLocal(days float64) time.Time            // días desde 1899-12-30 en hora local de pared, con ms
  func classifyAttachment(attType int, hidden bool, contentID string, htmlHasCID bool) domain.AttachmentKind
  func attachmentFileName(name string, kind domain.AttachmentKind, index int) string
  func senderText(name, address string) string
  func folderPath(outlookFolderPath string) string       // "\\\\cuenta\\Bandeja" → "cuenta\\Bandeja"
  func connectError(err error) error                      // envuelve con el texto en español
  // en domain/reasons.go:
  var ErrOutlookUnavailable = errors.New("No se pudo conectar con Outlook Classic. Ábrelo e inténtalo de nuevo, o ejecuta la app con --demo")
  ```

- [ ] **Paso 1: tests que fallan** (`convert_test.go`, tablas):
  - **`TestOleDateToLocal`**: `0` → 1899-12-30 00:00 local; `45000.5` → 2023-03-15 12:00:00 local; `46338.99999884259` → 2026-11-12 23:59:59.900 local (±1 ms, `Location() == time.Local`); `45000.25` → 06:00.
  - `TestClassifyAttachment`: `(1,false,"",false)` → file; `(1,true,"",false)` → inline; `(1,false,"img@x",true)` → inline; `(1,false,"img@x",false)` → file; `(5,…)` → msg; `(6,…)` y `(4,…)` → other.
  - `TestAttachmentFileName`: `("Acta", msg, 2)` → `"Acta.msg"`; `("Acta.msg", msg, 2)` → `"Acta.msg"`; `("", file, 3)` → `"adjunto_3"`; `("", msg, 1)` → `"adjunto_1.msg"`; `("informe.pdf", file, 1)` → igual.
  - `TestSenderText`: `("Ana","ana@x.example")` → `"Ana <ana@x.example>"`; `("","ana@x.example")` → `"ana@x.example"`; `("Ana","")` → `"Ana"`; `("ana@x.example","ana@x.example")` → `"ana@x.example"`.
  - `TestFolderPath`: `\\demo@example.com\Bandeja de entrada` → `demo@example.com\Bandeja de entrada`.
  - `TestConnectError`: `errors.Is(connectError(io.EOF), domain.ErrOutlookUnavailable)` y el mensaje contiene la causa.
  - `TestIsMailClass`: 43 → true; 26 (cita), 46 (informe) y 53 (solicitud de reunión) → false.
- [ ] **Paso 2:** `go test ./internal/outlook/` → FAIL.
- [ ] **Paso 3:** implementar. `oleDateToLocal`: separar días y fracción, `time.Date(1899, 12, 30, 0, 0, 0, 0, time.Local).AddDate(0, 0, int(days))` y sumar la fracción redondeada a ms. Así un día con cambio de horario no se desplaza.
- [ ] **Paso 4:** `go test ./internal/outlook/` → PASS.
- [ ] **Paso 5:** commit `feat(outlook): conversiones de fechas OLE, adjuntos y nombres`.

### Task 5: Goroutine STA y helpers COM

**Archivos:**
- Crear: `internal/outlook/sta.go`, `internal/outlook/com.go` y sus tests

**Interfaces:**
- Produce:
  ```go
  type sta struct { /* calls chan staCall; closed chan struct{} */ }
  func newSTA() (*sta, error)                              // arranca el goroutine: LockOSThread + CoInitializeEx(0, COINIT_APARTMENTTHREADED)
  func (s *sta) do(ctx context.Context, fn func() error) error // corre fn en el STA y espera; si ctx ya terminó antes de empezar, devuelve ctx.Err() sin correrla; un panic en fn vuelve como error y el STA sigue vivo
  func (s *sta) close()                                    // termina el goroutine (CoUninitialize); idempotente; do posterior → errSTAClosed
  // stream corre produce en el STA; cada yield entrega un item al goroutine que llamó, que corre fn.
  // yield devuelve false cuando hay que parar (fn pidió parar, devolvió error o ctx terminó).
  func stream[T any](ctx context.Context, s *sta, produce func(yield func(T) bool) error, fn func(T) error) error
  // com.go — todos liberan lo intermedio; el *ole.IDispatch devuelto es del llamador (Release)
  func getObj(d *ole.IDispatch, name string, args ...any) (*ole.IDispatch, error)
  func callObj(d *ole.IDispatch, method string, args ...any) (*ole.IDispatch, error)
  func getString(d *ole.IDispatch, name string, args ...any) (string, error)
  func getInt(d *ole.IDispatch, name string, args ...any) (int, error)
  func getBool(d *ole.IDispatch, name string, args ...any) (bool, error)
  func getDate(d *ole.IDispatch, name string) (time.Time, error) // usa oleDateToLocal con el float64 del VARIANT (C1)
  func call(d *ole.IDispatch, method string, args ...any) error
  func release(ds ...*ole.IDispatch)                             // ignora nil
  ```

- [ ] **Paso 1: tests que fallan** (sin Outlook):
  - **`TestSTAThreadAffinity`**: 20 llamadas `do` desde 4 goroutines registran `windows.GetCurrentThreadId()`; todas iguales.
  - **`TestSTAPanicRecovered`**: `do` con `panic("x")` devuelve un error que contiene "x"; la siguiente `do` corre en el mismo hilo.
  - `TestSTAClosed`: tras `close()`, `do` → `errSTAClosed`; un segundo `close()` no bloquea.
  - **`TestStreamCancel`**: `produce` emite enteros sin fin; `fn` cancela el ctx al recibir el 10; `stream` devuelve `context.Canceled` en menos de 1 s, el productor deja de correr (ningún `yield` después de que devolvió `false`), y una `do` posterior responde (el STA está libre).
  - `TestStreamStop`: `fn` devuelve `source.ErrStopScan` en el item 5 → `stream` devuelve `nil` y `fn` corrió 5 veces.
  - `TestComHelpersReleaseAll`: con `Scripting.Dictionary` (siempre presente en Windows) dentro del STA: `outer.Add("k", inner)` y `outer.Add("n", 7)`. `getInt(outer, "Count") == 2`, `getInt(outer, "Item", "n") == 7`, y `getObj(outer, "Item", "k")` + `release` deja el refcount de `inner` igual que antes (medido con `AddRef()`/`Release()`).
- [ ] **Paso 2:** `go test ./internal/outlook/` → FAIL.
- [ ] **Paso 3:** implementar. `do` envía `staCall{fn, done chan error}`; si ctx termina **mientras** fn corre, igual espera a fn (nunca hay dos llamadas COM a la vez). `stream`: el STA corre `produce` con un `yield` que hace `select { case items <- v: …; case <-stopped: return false }`; el llamador lee `items` y corre `fn`, y cierra `stopped` al parar. En `getDate`: si `VT_DATE`, `math.Float64frombits(uint64(v.Val))`.
- [ ] **Paso 4:** `go test -race ./internal/outlook/` → PASS.
- [ ] **Paso 5:** commit `feat(outlook): goroutine STA, streaming y helpers COM con liberación garantizada`.

### Task 6: Fuente Outlook (`MailSource`) y selección de fuente

**Archivos:**
- Crear: `internal/outlook/outlook.go`, `internal/outlook/outlook_test.go`, `internal/outlook/outlook_integration_test.go`
- Modificar: `dto.go`, `dto_test.go`, `app.go` (cerrar la fuente al salir), `go.mod` (go-ole y x/sys directas)

**Interfaces:**
- Consume: `sta`, `stream`, los helpers de `com.go` y las conversiones (Tareas 4 y 5).
- Produce:
  ```go
  type Options struct { ProgID string }          // "" = "Outlook.Application"
  func New(o Options) (*Source, error)           // arranca el STA; no conecta todavía (C7)
  func (s *Source) Close() error                 // libera Application/Namespace y cierra el STA
  var _ source.MailSource = (*Source)(nil)
  // Name() == "outlook"
  ```
  `selectSource(args)`: con `--demo` → demo (como hoy); sin `--demo` → `outlook.New(outlook.Options{})`. Desaparece `errNoOutlook`. `App.shutdown` llama a `Close()` si la fuente implementa `io.Closer`, después de `Wait`.

  Comportamiento (sobre el STA, conexión perezosa al primer uso; si falla, `connectError` y se reintenta en la próxima llamada):
  - `ListRoots`: `Namespace.Folders` → por cada raíz, `Folder{Ref{StoreID, EntryID, Path: folderPath(FolderPath)}, Name, ItemCount: Items.Count, HasChildren: Folders.Count > 0}`.
  - `ListChildren(ref)`: `Namespace.GetFolderFromID(ref.EntryID, ref.StoreID).Folders`, con el mismo mapeo.
  - `Scan(ref)`: `Items` → `Sort("[ReceivedTime]", true)` → `GetFirst`/`GetNext` dentro de `stream`. Por item: `Class`; si no es `olMail`, `Mail{IsMail:false, Subject, ReceivedAt}` con lo que se pueda leer. Si es correo: `EntryID`, `Subject`, `senderText(SenderName, SenderEmailAddress)`, `getDate("ReceivedTime")` y adjuntos (`Attachments.Count`, `Item(i)`: `FileName`, `Size`, `Type`, y para Type 1 `PropertyAccessor.GetProperty` de `PR_ATTACHMENT_HIDDEN` `http://schemas.microsoft.com/mapi/proptag/0x7FFE000B` y `PR_ATTACH_CONTENT_ID` `http://schemas.microsoft.com/mapi/proptag/0x3712001F`; `HTMLBody` se lee una sola vez por correo y solo si algún adjunto tiene Content-ID). Recuerda `entryID → storeID`. Cualquier error del item → `ScanItem{Err}`. Cada item y cada adjunto se liberan antes del siguiente.
  - `SaveAttachment(mailEntryID, index, dest)`: `GetItemFromID(entryID, storeID recordado)` → `Attachments.Item(index).SaveAsFile(dest)`. `ctx` se revisa antes de empezar.

- [ ] **Paso 1: tests que fallan.**
  - `TestConnectError` (outlook_test.go, sin Outlook): `New(Options{ProgID: "OutlookDescargas.NoExiste"})` y `ListRoots` → `errors.Is(err, domain.ErrOutlookUnavailable)`; `Close()` sin error; `Name() == "outlook"`.
  - `TestSelectSource` (dto_test.go): `--demo` → `Name() == "demo"`; sin flag → `Name() == "outlook"` y sin error (no conecta), cerrando la fuente con `Close`.
  - `outlook_integration_test.go` (`//go:build windows && integration`; `t.Skip` si `ListRoots` devuelve `ErrOutlookUnavailable`): `TestIntegrationListRoots` (≥ 1 raíz con StoreID y EntryID no vacíos); `TestIntegrationScanFirst50` (la bandeja de entrada de la primera raíz, si existe, devuelve hasta 50 items en orden descendente sin `Err` de conexión; imprime solo el conteo); `TestIntegrationSaveFirstAttachment` (guarda el primer adjunto `file` encontrado en `t.TempDir()` y comprueba `Size > 0`, sin imprimir el nombre).
- [ ] **Paso 2:** `go test ./internal/outlook/ . -run 'TestConnectError|TestSelectSource'` → FAIL.
- [ ] **Paso 3:** implementar. Mantener `outlook.go` en funciones cortas: `connect`, `folderInfo`, `readMail`, `readAttachments`, `isInline`.
- [ ] **Paso 4:** `go test ./... && go vet ./... && go vet -tags integration ./...` → PASS. **No** correr `go test -tags integration` (C8).
- [ ] **Paso 5:** commit `feat(outlook): fuente Outlook Classic sobre COM detrás de MailSource`.

### Task 7: Spike, checklist de validación y documentación

**Archivos:**
- Crear: `cmd/spike-outlook/main.go`, `docs/validacion-outlook.md`
- Modificar: `AGENTS.md` (comandos: ejecutar sin `--demo`, tests de integración y spike), `docs/demo-y-portafolio.md` (§2.2 apunta al checklist), `docs/superpowers/plans/2026-11-12-nucleo-modo-demo.md` (hoja de ruta: Plan 3 ✅)

**Interfaces:**
- Consume: `outlook.New`, `MailSource`.

- [ ] **Paso 1:** `cmd/spike-outlook` (`//go:build windows`), con flags `-n 10000` y `-folder "<ruta legible>"` (por defecto, la bandeja de entrada de la primera raíz). Lista las raíces (imprime **cantidad**, no nombres), recorre hasta N items con `Scan` y muestra items/s, cuántos son correos, cuántos adjuntos y el tiempo total. Al final indica que se observe la memoria de `OUTLOOK.EXE` en el Administrador de tareas durante la corrida (B11). No imprime asuntos, remitentes, nombres de archivo ni nombres de carpeta.
- [ ] **Paso 2:** `docs/validacion-outlook.md` con casillas:
  - **Preparación:** Outlook Classic abierto; `go test -tags integration ./internal/outlook/ -v`.
  - **Spike:** `go run ./cmd/spike-outlook -n 10000`; anotar items/s y la memoria de OUTLOOK.EXE al inicio y al final.
  - **App sin `--demo`:** el árbol lista cuentas y PST, las subcarpetas cargan sin congelar la UI, extraer un rango corto (por mes y plana), el filtro de extensiones, omitir inline (firmas), un `.msg` embebido guardado como `.msg`, un adjunto bloqueado (`.exe`) que queda como error en el reporte, cancelar a mitad (la UI se recupera), relanzar, el equipo no se suspende (`powercfg /requests` muestra la app durante el job), beep y parpadeo con la ventana minimizada, Alt+F4 con un job activo pide confirmación, "Abrir carpeta" con una ruta que tiene coma.
  - **Fechas:** un correo recibido cerca de la medianoche entra en el día correcto.
  - **Sin datos reales en ningún artefacto:** no guardar capturas ni reportes del buzón real en el repo.
  - **Resultados (para el benchmark D5):** una tabla vacía para anotar las cifras.
- [ ] **Paso 3:** `go build ./... && go vet ./...` → PASS.
- [ ] **Paso 4:** commit `docs: checklist de validación con Outlook real y spike de medición`.

### Task 8: Mejoras menores pendientes del Plan 2

**Archivos:**
- Modificar: `internal/extract/extract.go` (+test), `frontend/src/lib/job.ts` (+test), `frontend/src/components/{SummaryScreen,ProgressScreen,FolderTree,ConfigScreen}.tsx`, `frontend/src/styles/{tokens,app}.css`

**Interfaces:**
- Produce: `JobState.destDir: string` (de `request.destDir` en `started`, de `status.destDir` en `restored`) y `JobState.activityBase: number` (cuántas actividades se recortaron desde el inicio del job; la key de cada fila es `activityBase + i`). En `extract`, la variable `writeReport = report.WriteExcel` como seam de prueba.

- [ ] **Paso 1: tests que fallan.**
  - `TestRunReportWriteFails` (extract): con `writeReport` reemplazado por una función que falla, `Run` devuelve un error y `res.ReportPath == ""`.
  - `job.test.ts`: `started` fija `destDir` desde el request; `restored` usa `status.destDir` aunque `request` sea `null`; tras `ACTIVITY_MAX + 5` actividades, `activityBase === 5`, y `started` lo vuelve a 0.
- [ ] **Paso 2:** `go test ./internal/extract/ -run TestRunReportWriteFails` y `npm test` → FAIL.
- [ ] **Paso 3:** implementar. En la UI:
  - `SummaryScreen` usa `job.destDir`.
  - `ProgressScreen` usa la key `activityBase + i`.
  - `FolderTree` descarta las respuestas de `ListChildren`/`ListRoots` de una generación anterior a "Recargar" (un contador en una ref).
  - `ConfigScreen` calcula los presets en cada render (sin `useMemo`), para que no queden viejos pasada la medianoche.
  - Los hex de `app.css` pasan a `tokens.css`: `--primary-soft` `#E0F2FE`, `--primary-ink` `#075985`, `--muted-ink` `#94A3B8`, `--ink-strong` `#0F172A`, `--on-primary` `#FFFFFF`, `--success-soft`/`--success-border`/`--success-ink` `#ECFDF5`/`#A7F3D0`/`#065F46`, `--warning-soft`/`--warning-border`/`--warning-ink` `#FFFBEB`/`#FDE68A`/`#92400E`, `--error-soft`/`--error-border`/`--error-ink` `#FEF2F2`/`#FECACA`/`#991B1B`, `--error-row` `#B91C1C`, `--accent-soft`/`--accent-ink` `#F5F3FF`/`#6D28D9`, `--info-soft`/`--info-border`/`--info-ink` `#ECFEFF`/`#A5F3FC`/`#155E75`. Las sombras `rgba` quedan como están.
  - Comprobación: `grep -E '#[0-9A-Fa-f]{3,6}\b' frontend/src/styles/app.css` no devuelve nada.
- [ ] **Paso 4:** `go test ./... && cd frontend && npm test && npm run build` → PASS.
- [ ] **Paso 5:** commit `fix: mejoras menores del Plan 2 (reporte fallido, destino tras recargar, keys, tokens, árbol)`.

**Verificación manual del usuario (no la ejecuta el agente):** el checklist completo de `docs/validacion-outlook.md`.
