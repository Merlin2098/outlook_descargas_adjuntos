# AGENTS.md — outlook_descargas_adjuntos (Go + Wails)

Guía para agentes de código (Claude Code, Codex, Cursor, etc.). Es la fuente canónica de
reglas del proyecto; `CLAUDE.md` solo la importa y agrega notas específicas de Claude.

## Qué es este proyecto

Proyecto de **portafolio**: una app de escritorio Windows que descarga en bloque los adjuntos de una
carpeta de **Outlook Classic** (vía COM), con progreso en vivo, cancelación y un reporte de la ejecución.

Es la reescritura en **Go + Wails v2 + React (TypeScript)** de una herramienta interna hecha en Python
(PySide6 + pywin32). El rewrite es una **descarga genérica**: no porta las reglas de negocio del original
(clasificación de documentos firmados, filtro de frases sobre el nombre del adjunto).

Objetivos, en este orden:

1. **Mostrar el stack Go + Wails** con código limpio: concurrencia bien resuelta, COM desde Go, bindings tipados y eventos en tiempo real.
2. **Grabar un video demo** de la extracción en **modo demo** (datos ficticios; ver [docs/demo-y-portafolio.md](docs/demo-y-portafolio.md)).
3. Ser una herramienta que funcione de verdad: la conexión con Outlook real se valida localmente, pero no se graba.

Antes de tocar la parte de Outlook, lee **[docs/diagnostico-legacy.md](docs/diagnostico-legacy.md)**:
documenta los problemas técnicos del original que no hay que repetir.

## Alcance funcional (MVP)

- Explorar las cuentas y carpetas de Outlook (árbol con carga asíncrona) y elegir una carpeta. Opcionalmente, incluir sus subcarpetas.
- Rango de fechas, que es obligatorio.
- Filtros opcionales y genéricos: el asunto contiene un texto, el remitente contiene un texto, extensiones de adjunto (`pdf, xlsx, ...`).
- Ignorar por defecto los adjuntos inline (imágenes de firma).
- Carpeta de destino, con dos formas de organizar: plana o por subcarpeta `YYYY-MM`. Los nombres repetidos se resuelven con el sufijo `_N`.
- Progreso en vivo por fase, registro de actividad en pantalla, cancelación y resumen al terminar.
- Reporte de la ejecución en Excel, con una fila por adjunto y una hoja de correos omitidos y errores con su motivo.
- Tema claro/oscuro, prevención de suspensión del equipo durante el job y aviso (beep y parpadeo de la ventana) al terminar.
- **Modo demo**: una fuente de correos simulada que permite ejecutar la app sin Outlook.

**Fuera de alcance**: el clasificador de firmas, los modos de búsqueda por frases y Parquet. No los reintroduzcas sin pedirlo.

## Layout del repositorio

```text
outlook_adjuntos/
├── AGENTS.md / CLAUDE.md
├── docs/                              # diagnóstico, demo/portafolio, ADRs (docs/adr/)
├── scripts/                           # scripts auxiliares (datos demo, build)
├── Outlook_AttachExtractor_Rework/    # LEGACY Python — SOLO LECTURA (repo git propio)
└── (app Go + Wails, a crear — ver "Estructura objetivo")
```

**`Outlook_AttachExtractor_Rework/` es de solo lectura.** Sirve como referencia técnica y nada más.
No lo modifiques ni ejecutes sus scripts. Ignora `agents2/` y cualquier referencia a "Tinker".

## Stack y toolchain (verificado en la máquina de desarrollo)

| Herramienta | Versión |
| --- | --- |
| Go | 1.27.x (windows/amd64) |
| Wails CLI | **v2.17.1 local en `.tools/wails.exe`** (instalar con `scripts/instalar-herramientas.ps1`). El CLI global v2.10.x falla con Go 1.27 al generar bindings |
| Node / npm | 24.x / 12.x |

Librerías previstas (verifica la API real en la documentación o en el código fuente de cada módulo antes de usarla; no inventes firmas):

- COM / Outlook: `github.com/go-ole/go-ole` (+ `oleutil`)
- Win32 (energía, flash de ventana, beep): `golang.org/x/sys/windows`
- Excel: `github.com/xuri/excelize/v2`
- Logging: `log/slog` de la stdlib
- Frontend: React 18 + TypeScript 5.6 + Vite 5.4 (base: template `react-ts` de Wails), Vitest 2.1 para la lógica de `src/lib/`,
  fuentes locales con `@fontsource/inter` y `@fontsource/fira-code` (sin Google Fonts: la app es 100 % local)

Si quieres agregar una dependencia que no está en esta lista, justifícala y prefiere la stdlib.

## Estructura objetivo (propuesta)

```text
main.go                 # wails.Run, opciones de ventana, bind de servicios
app.go                  # ciclo de vida (startup/shutdown), ctx de Wails
internal/
  domain/               # tipos puros: Mail, Attachment, Folder, Filters, RunRecord, Progress
  source/               # interfaz MailSource + implementación demo (fake)
  outlook/              # implementación COM de MailSource (//go:build windows)
  extract/              # orquestación del job: filtrar → descargar → reportar
  report/               # Excel de la ejecución
  jobs/                 # registro de jobs, cancelación, eventos a la UI
  winapi/               # SetThreadExecutionState, FlashWindowEx, MessageBeep
  config/               # config.json en %APPDATA%
  logging/              # slog → archivos en %LOCALAPPDATA%
frontend/               # React; frontend/wailsjs/ es generado por Wails
```

Regla de dependencias: `domain` no importa nada externo. `extract` depende de la **interfaz**
`MailSource`, nunca de `outlook/`. Toda la lógica se testea con la fuente demo, sin Outlook.

## Comandos

```bash
# "wails" = .tools/wails.exe (v2.17.1, fijado para este proyecto)
.tools/wails.exe dev -appargs "--demo"     # desarrollo con la fuente simulada (sin Outlook)
.tools/wails.exe dev                       # contra Outlook Classic real (validación local; no grabar)
.tools/wails.exe build -clean -trimpath    # binario de release en build/bin/
go test ./...                   # tests (deben pasar sin Outlook instalado)
go test -tags integration ./internal/outlook/ -v   # contra Outlook real: lo corre el usuario
go run ./cmd/spike-outlook -n 10000                # mide el recorrido COM: lo corre el usuario
# checklist manual: docs/validacion-outlook.md
go vet ./...
gofmt -l . && goimports -l .    # sin salida = formateado
cd frontend && npm test         # tests de la lógica del frontend (Vitest, src/lib/*.test.ts)
cd frontend && npm run build    # typecheck + build del frontend
.tools/wails.exe generate module   # regenera frontend/wailsjs/ tras cambiar bindings
```

Wails regenera `frontend/wailsjs/`. **No edites esos archivos a mano.**

## Reglas críticas de COM / Outlook

1. **Un solo goroutine dueño de COM.** Todo acceso a Outlook corre en un goroutine con
   `runtime.LockOSThread()` + `ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)` y su
   `CoUninitialize` con `defer`. Ningún `*ole.IDispatch` cruza a otro goroutine; se comunican por canales.
2. **Libera cada IDispatch** (`defer x.Release()`) y cada `VARIANT` que lo contenga. El original se cuelga
   tras miles de correos por acumulación de objetos COM. Encapsula el acceso en helpers que hagan `Release` de forma consistente.
3. **No todo item es un MailItem.** Comprueba `Class == 43` (olMail). Los demás items se registran como omitidos con su motivo.
4. **Identifica carpetas por `EntryID` + `StoreID`**, no por la ruta de nombres. La ruta legible es solo para la UI.
5. **Fechas**: `ReceivedTime` es un `VT_DATE` en hora local. Normaliza a `time.Local` y compara días completos. Cúbrelo con tests.
6. **Adjuntos**: filtra por `Attachment.Type` (by value = 1) y detecta los inline. Sanea los nombres de archivo y maneja rutas de más de 260 caracteres.
7. Un error en un correo nunca aborta el job: queda en el reporte con su motivo y el job sigue. Nunca uses `continue` sin dejar registro.

## Reglas de Wails / concurrencia

- Los métodos expuestos con bind **retornan rápido**: lanzan el job y devuelven un `jobID`. El progreso va
  por `runtime.EventsEmit(ctx, "job:progress", payload)` con un payload tipado y **limitado a unos 10 eventos por segundo**.
- Cancelación con `context.Context`. Nada de flags booleanos compartidos sin sincronización.
- **Todo job termina con un evento terminal** (`job:done` | `job:error` | `job:cancelled`). El original
  deja la UI bloqueada tras cancelar; no repitas ese bug.
- La UI solo conoce el estado del job por los eventos o por `GetJobStatus(jobID)`.
- `SetThreadExecutionState` es **por hilo**: llámalo desde un goroutine con `LockOSThread` que viva
  mientras dure el job, y restaura `ES_CONTINUOUS` al terminar, incluso si hay panic o se cancela.

## Portafolio y datos (obligatorio)

- **Nunca uses datos reales** de un buzón corporativo en tests, fixtures, capturas, el video ni el README.
  Usa solo la fuente demo o el buzón demo generado por `scripts/` (ver `docs/demo-y-portafolio.md`).
- No menciones el nombre de la empresa original ni su contexto interno en el código, los commits ni la documentación pública.
- La calidad visible importa: el README debe tener un GIF o video, un diagrama de arquitectura y la explicación de las decisiones técnicas.
  La UI debe verse bien en una grabación a 1280×720.

## Convenciones de código

- **Go**: identificadores en inglés, `gofmt`/`goimports`, errores con contexto
  (`fmt.Errorf("open folder %s: %w", id, err)`), sin `panic` en lógica de negocio, `slog` con campos estructurados.
- **Frontend**: TypeScript estricto, componentes funcionales, sin `any`. Los tipos del backend se importan de `wailsjs/go/models`.
- **Referencia visual (aprobada el 2026-11-12)**: `docs/diseno/mock-ui/*.dc.html`, tres pantallas: configurar, en curso y resumen. Se basa en el
  **tema claro del original** (`Outlook_AttachExtractor_Rework/config/theme_light.json`): paleta slate + azul cielo.
  - Tokens: fondo `#F8FAFC`, superficie `#FFFFFF`, superficie alternativa `#F1F5F9`, borde `#CBD5E1` / `#E2E8F0`,
    texto `#1E293B` / `#475569` / `#64748B`, primario `#0284C7`, acento `#8B5CF6`, éxito `#10B981`, aviso `#F59E0B`,
    error `#EF4444`, info `#06B6D4`.
  - Accesibilidad: para texto y fondos de botón usa `#0369A1` (`primary_dark` del original; `#0284C7` sobre blanco no llega
    a 4,5:1). Para texto de estado usa `#047857`, `#B45309` y `#DC2626`; los tonos base quedan para iconos y barras.
  - Estilo heredado: Inter (UI) y Fira Code (consola y rutas), radio de 4 px en controles y de 8 px en tarjetas, bordes de 2 px en
    los campos, botones en MAYÚSCULAS con peso 600, títulos de sección de 16 px en el color primario, carpeta seleccionada con fondo
    primario sólido y barra de progreso de 24 px con degradado y porcentaje centrado.
  - Ventana sin marco con barra de título propia. Convierte los colores a variables CSS; no copies los estilos inline del mock.
  - Tema oscuro: opcional y posterior; derivarlo de `theme_dark.json`.
- **Idioma: español** en la UI, el README y `docs/`. Los textos de la UI van centralizados en `frontend/src/i18n/es.ts`.
- Nombre público del proyecto, del repo y del binario: **`outlook_descargas_adjuntos`**.
- Rutas de usuario: config en `%APPDATA%\outlook_descargas_adjuntos\`, logs en `%LOCALAPPDATA%\outlook_descargas_adjuntos\logs\`.

## Testing

- TDD para `extract`, `report`, `jobs` y el saneamiento de nombres: tests de tabla, con la fuente demo e inyección de errores.
- Los tests contra Outlook real llevan `//go:build windows && integration` y no corren en `go test ./...`.
- Los tests que escriben archivos usan `t.TempDir()`.

## Límites — pregunta antes de hacerlo

- Modificar cualquier cosa bajo `Outlook_AttachExtractor_Rework/`.
- Ampliar el alcance más allá del MVP (por ejemplo, reintroducir el clasificador o agregar otros proveedores de correo).
- Agregar dependencias pesadas, CGO o llamadas de red (la app es 100 % local).
- Crear o modificar stores o PST en el Outlook del usuario fuera de los scripts demo documentados.
- Inicializar git, hacer commits o publicar algo.
