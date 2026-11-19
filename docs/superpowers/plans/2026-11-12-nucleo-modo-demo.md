# Plan 1 — Núcleo de extracción + modo demo: plan de implementación

> **Para agentes:** SUB-SKILL REQUERIDA: usa superpowers:subagent-driven-development (recomendado) o
> superpowers:executing-plans para ejecutar este plan tarea por tarea. Los pasos usan casillas (`- [ ]`).

**Objetivo:** dejar el backend Go completo de la descarga genérica de adjuntos: tipos de dominio, fuente demo,
pipeline de extracción, reporte Excel y gestor de jobs con eventos. Todo queda expuesto a Wails y se
verifica con `go test ./...` y con una pantalla de humo en `wails dev -appargs "--demo"`, sin Outlook.

**Arquitectura:** `extract.Run` orquesta escanear → filtrar → descargar → reportar contra la interfaz
`source.MailSource`. La fuente demo implementa esa interfaz en memoria; el adaptador COM llega en el Plan 3.
`jobs.Manager` ejecuta un job a la vez en un goroutine, limita la frecuencia de eventos y garantiza
exactamente un evento terminal. `app.go` (paquete `main`) solo traduce DTOs y delega.

**Stack:** Go 1.27, Wails v2.10 (template `react-ts`), `github.com/xuri/excelize/v2`, `log/slog`, React + TypeScript.

**Especificación:** [AGENTS.md](../../../AGENTS.md) (alcance MVP y reglas), [docs/demo-y-portafolio.md](../../demo-y-portafolio.md)
§2.1 (fuente demo), [docs/diagnostico-legacy.md](../../diagnostico-legacy.md) §5 (bugs que no se portan: B4, B7, B9, B10, B14–B18).

## Hoja de ruta (planes siguientes, cada uno en su propio documento)

| Plan | Entrega |
| --- | --- |
| **1 (este)** | Núcleo + fuente demo + jobs/eventos + reporte + bindings + pantalla de humo |
| 2 ✅ | UI React según `docs/diseno/mock-ui/` (tokens CSS, `i18n/es.ts`, 3 pantallas, árbol asíncrono) |
| 3 | Adaptador COM de Outlook (`internal/outlook`), `internal/winapi` (suspensión, beep, flash), checklist de validación local |
| 4 | Portafolio: README en español, CI en GitHub Actions, release y grabación del video |

## Restricciones globales

- Nombre del módulo, del binario y del proyecto Wails: `outlook_descargas_adjuntos`.
- Identificadores Go en inglés; los textos visibles para el usuario (motivos, errores, encabezados de Excel) en español.
- La lógica de `internal/` no importa Wails. Solo `main.go` y `app.go` importan `github.com/wailsapp/wails/v2`.
- `go test ./...` pasa sin Outlook y sin red. Los tests que escriben archivos usan `t.TempDir()`.
- Fechas: los filtros comparan días completos en `time.Local`: `[From 00:00:00, To 23:59:59.999999999]`.
- Nunca se sobrescribe un archivo existente. Los duplicados se resuelven con `nombre_1.ext`, `nombre_2.ext`, ….
- Nada se omite sin registro: cada correo en rango que no produce descargas genera un `RunRecord` con motivo.
- Eventos Wails: `job:progress`, `job:activity`, `job:done`, `job:error`, `job:cancelled`. Frecuencia máxima de 10 por segundo por tipo.
- Config en `%APPDATA%\outlook_descargas_adjuntos\config.json` y logs en `%LOCALAPPDATA%\outlook_descargas_adjuntos\logs\`.
- Los structs que cruzan a TypeScript (`FolderRef`, `Folder`, `Summary`, `RunRecord`, `Progress`, `Activity`, `Terminal`, los DTOs) llevan tags `json` en camelCase en cada campo. Las duraciones se exponen como `int64` en milisegundos (campo `...Ms`), nunca como `time.Duration`.
- Cada commit termina con la línea `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- `Outlook_AttachExtractor_Rework/` es de solo lectura y queda fuera del repo (en `.gitignore`).

## Foco de revisión

Cinco casos que la especificación implica y que más probablemente afecten al usuario. Cada uno tiene su test en la tarea indicada:

1. **Nombres de adjunto hostiles** (`a<b>.pdf`, `CON.pdf`, `"".pdf`, nombre de 300 caracteres): se guardan con un nombre válido y nunca fallan por el nombre (Tarea 3).
2. **Archivos ya existentes en el destino** (de una ejecución anterior): se renombran con `_N` y nunca se sobrescriben (Tarea 3).
3. **Cancelar a mitad de la descarga:** exactamente un `job:cancelled`, reporte parcial escrito y el gestor acepta un job nuevo (Tarea 7).
4. **Bordes del rango de fechas:** un correo a las 00:00:00 de `From` y otro a las 23:59:59 de `To` entran; uno a las 00:00:00 del día siguiente a `To` no (Tarea 5).
5. **Destino inválido** (vacío, ruta que es un archivo o sin permiso de escritura): `job:error` con un mensaje en español antes de escanear; un segundo `Start` con un job activo devuelve `ErrJobRunning` (Tareas 2 y 7).

---

## Estructura de archivos

```text
.gitignore                          # + Outlook_AttachExtractor_Rework/, build/bin, node_modules, frontend/dist
go.mod                              # module outlook_descargas_adjuntos
main.go                             # wails.Run, flags, selección de fuente
app.go                              # struct App: métodos bindeados
dto.go / dto_test.go                # JobRequestDTO ⇄ domain.JobRequest (fechas como "2006-01-02")
internal/domain/types.go            # FolderRef, Folder, Attachment, Mail, Filters, JobRequest, RunRecord, Summary
internal/domain/validate.go(+_test) # JobRequest.Validate, NormalizeExtensions
internal/domain/reasons.go          # textos de motivos y errores (español)
internal/fsutil/names.go(+_test)    # SanitizeFileName, FitPath, UniquePath
internal/source/source.go           # interfaz MailSource, ScanItem, ErrStopScan
internal/source/demo/fixture.go     # tipos del fixture + DefaultFixture
internal/source/demo/demo.go(+_test)# implementación de MailSource
internal/extract/extract.go(+_test) # Run, Sink, Progress, Activity
internal/extract/filter.go(+_test)  # evaluación por correo → adjuntos elegibles o motivo
internal/report/excel.go(+_test)    # WriteExcel
internal/jobs/manager.go(+_test)    # Manager, Emitter, throttling, eventos terminales
internal/config/config.go(+_test)   # Load/Save
internal/logging/logging.go(+_test) # Setup + retención
frontend/src/App.tsx                # pantalla de humo (se reemplaza en el Plan 2)
```

---

### Tarea 1: Scaffold de Wails + repositorio

**Archivos:**
- Crear: todo el template de Wails en la raíz, `.gitignore`
- Modificar: `go.mod` (módulo y versión de Go)

**Interfaces:**
- Produce: módulo `outlook_descargas_adjuntos`; `wails build` y `go test ./...` funcionan.

- [ ] **Paso 1: Generar el template en una carpeta temporal y moverlo a la raíz.** El directorio raíz no está vacío, y `wails init` no genera sobre un directorio que ya tiene contenido.

```bash
cd <raíz del repo>
wails init -n outlook_descargas_adjuntos -t react-ts -d _scaffold
cp -r _scaffold/. . && rm -rf _scaffold
```

- [ ] **Paso 2: Ajustar el módulo.** `go mod edit -module outlook_descargas_adjuntos -go=1.27`. En `wails.json` deja `"name": "outlook_descargas_adjuntos"` y `"outputfilename": "outlook_descargas_adjuntos"`.
- [ ] **Paso 3: Completar el `.gitignore` del template** con `Outlook_AttachExtractor_Rework/`, `_scaffold/` y `*.log`.
- [ ] **Paso 4: Verificar.** `go build ./... && wails build -clean` termina sin errores y genera `build/bin/outlook_descargas_adjuntos.exe`.
- [ ] **Paso 5: Inicializar git y hacer el commit inicial** (`git init -b main`). Commit: `chore: scaffold Wails react-ts para outlook_descargas_adjuntos`. Incluye `AGENTS.md`, `CLAUDE.md` y `docs/`.

---

### Tarea 2: Tipos de dominio y validación

**Archivos:**
- Crear: `internal/domain/types.go`, `internal/domain/reasons.go`, `internal/domain/validate.go`
- Test: `internal/domain/validate_test.go`

**Interfaces:**
- Produce (lo usan todas las tareas siguientes):

```go
type FolderRef struct { StoreID, EntryID, Path string }        // Path legible: `cuenta\carpeta\sub`
type Folder struct { Ref FolderRef; Name string; ItemCount int; HasChildren bool }

type AttachmentKind string
const ( KindFile AttachmentKind = "file"; KindInline = "inline"; KindEmbeddedMsg = "msg"; KindOther = "other" )
type Attachment struct { Index int; FileName string; Size int64; Kind AttachmentKind } // Index base 1, como en COM

type Mail struct { EntryID, Subject, Sender string; ReceivedAt time.Time; IsMail bool; Attachments []Attachment }

type Organize string
const ( OrganizeFlat Organize = "flat"; OrganizeByMonth Organize = "by_month" )

type Filters struct {
    From, To time.Time            // solo cuenta la fecha (día) en time.Local
    SubjectContains, SenderContains string
    Extensions []string           // normalizadas: minúsculas, con punto; vacío = todas
    SkipInline, IncludeSubfolders bool
}
type JobRequest struct { Folder FolderRef; Filters Filters; DestDir string; Organize Organize }

type RecordStatus string
const ( StatusSaved RecordStatus = "saved"; StatusSkipped = "skipped"; StatusError = "error" )
type RunRecord struct {
    Status RecordStatus; Reason string
    FolderPath, MailEntryID, Subject, Sender string; ReceivedAt time.Time
    FileName, SavedPath string; Size int64
}
type Summary struct {
    Scanned, OutOfRange, InRange, MailsWithDownloads, Saved, Skipped, Errors int
    Bytes int64; ByExt map[string]int; Reasons map[string]int
    Started, Finished time.Time; Cancelled bool
}
func (r JobRequest) Validate() error
func NormalizeExtensions(in []string) []string
```

- `reasons.go` define estas constantes de texto, tal cual:
  - `ReasonNotMail = "No es un correo"`
  - `ReasonNoAttachments = "Sin adjuntos"`
  - `ReasonOnlyInline = "Solo imágenes inline"`
  - `ReasonNoMatchingExt = "Sin adjuntos con las extensiones elegidas"`
  - `ReasonUnsupportedKind = "Tipo de adjunto no soportado"`
  - `ReasonSubject = "El asunto no coincide"`
  - `ReasonSender = "El remitente no coincide"`
  - `ReasonReadError = "No se pudo leer el elemento"`

  Y estos errores: `ErrNoFolder = errors.New("Debe seleccionar una carpeta de Outlook")`, `ErrNoDest = errors.New("Debe seleccionar una carpeta de destino")`, `ErrDestIsFile = errors.New("El destino es un archivo, no una carpeta")`, `ErrDestNotWritable = errors.New("No se puede escribir en la carpeta de destino")`, `ErrBadRange = errors.New("La fecha de inicio es posterior a la fecha fin")`.

- [ ] **Paso 1: Escribir los tests de tabla que fallan** en `validate_test.go`:
  - `TestValidate`: `Folder.EntryID==""` → `ErrNoFolder`; `DestDir==""` → `ErrNoDest`; un `DestDir` que apunta a un archivo creado en `t.TempDir()` → `ErrDestIsFile`; `From` posterior a `To` → `ErrBadRange`; `From==To` (mismo día) → `nil`; un `DestDir` que no existe dentro de `t.TempDir()` → `nil`, y después de llamar a `Validate` el directorio existe (`Validate` hace `MkdirAll` y prueba escritura creando y borrando un archivo temporal).
  - `TestNormalizeExtensions`: `[" PDF", "xlsx", ".Xml", "", "pdf"]` → `[".pdf", ".xlsx", ".xml"]` (orden de primera aparición, sin duplicados ni vacíos).
- [ ] **Paso 2:** `go test ./internal/domain/...` → FALLA (no compila).
- [ ] **Paso 3: Implementar** `types.go`, `reasons.go` y `validate.go` con las firmas de arriba. Para el chequeo de escritura usa `os.CreateTemp(dest, ".write-test-*")` y luego `os.Remove`; si falla, devuelve `ErrDestNotWritable`.
- [ ] **Paso 4:** `go test ./internal/domain/...` → PASA.
- [ ] **Paso 5: Commit** `feat(domain): tipos de dominio y validación de JobRequest`.

---

### Tarea 3: Nombres de archivo seguros

**Archivos:**
- Crear: `internal/fsutil/names.go`
- Test: `internal/fsutil/names_test.go`

**Interfaces:**
- Produce:
  - `func SanitizeFileName(name string) string`
  - `func FitPath(dir, name string, max int) (string, error)`: devuelve el nombre recortado; error `ErrPathTooLong` si `dir` ocupa ya más de `max-16`.
  - `func UniquePath(dir, name string) (string, error)`: devuelve la ruta completa libre; error después de 9999 intentos.
  - `const MaxPath = 240`

- [ ] **Paso 1: Escribir los tests que fallan:**
  - `TestSanitizeFileName` (tabla):
    - `a<b>:c"d/e\f|g?h*.pdf` → `a_b__c_d_e_f_g_h_.pdf`
    - `CON.pdf` → `_CON.pdf`; `lpt1.txt` → `_lpt1.txt`
    - `""` → `adjunto`; `"   "` → `adjunto`
    - `informe. ` → `informe`
    - `"fac\x01tura.pdf"` → `fac_tura.pdf`
    - `año_2026.xlsx` → `año_2026.xlsx` (sin cambios)
  - `TestFitPath`:
    - nombre de 300 `a` + `.pdf` con `dir` de 50 caracteres → `len(filepath.Join(dir, res)) <= 240`, termina en `.pdf`, sin error;
    - `dir` de 230 caracteres → `ErrPathTooLong`.
  - `TestUniquePath`:
    - en un `t.TempDir()` con `f.pdf` y `f_1.pdf` existentes, `UniquePath(dir, "f.pdf")` → `filepath.Join(dir, "f_2.pdf")`;
    - en un directorio vacío → `filepath.Join(dir, "f.pdf")`;
    - nombre sin extensión `LEEME` con `LEEME` existente → `LEEME_1`.
- [ ] **Paso 2:** `go test ./internal/fsutil/...` → FALLA.
- [ ] **Paso 3: Implementar.** Reemplaza `<>:"/\|?*` y los runes < 0x20 por `_`, recorta espacios y puntos finales, y antepone `_` a los nombres reservados de Windows (`CON PRN AUX NUL COM1–9 LPT1–9`), comparando el stem sin distinguir mayúsculas. `FitPath` recorta el stem (por runas, no por bytes) y conserva la extensión, con un margen de 6 caracteres para el sufijo `_NNNN`.
- [ ] **Paso 4:** `go test ./internal/fsutil/...` → PASA.
- [ ] **Paso 5: Commit** `feat(fsutil): sanitizado, recorte y nombres únicos sin sobrescritura`.

---

### Tarea 4: Interfaz MailSource y fuente demo

**Archivos:**
- Crear: `internal/source/source.go`, `internal/source/demo/fixture.go`, `internal/source/demo/demo.go`
- Test: `internal/source/demo/demo_test.go`

**Interfaces:**
- Consume: `domain.*` (Tarea 2).
- Produce:

```go
// internal/source
type ScanItem struct { Mail domain.Mail; Err error }   // Err != nil: no se pudo leer el elemento
var ErrStopScan = errors.New("stop scan")               // fn lo devuelve para cortar sin error
type MailSource interface {
    Name() string                                                       // "demo" | "outlook"
    ListRoots(ctx context.Context) ([]domain.Folder, error)
    ListChildren(ctx context.Context, ref domain.FolderRef) ([]domain.Folder, error)
    Scan(ctx context.Context, ref domain.FolderRef, fn func(ScanItem) error) error // orden ReceivedAt descendente
    SaveAttachment(ctx context.Context, mailEntryID string, index int, destPath string) error
}

// internal/source/demo
type FixtureAttachment struct { FileName string; Size int64; Kind domain.AttachmentKind; FailWith string }
type FixtureItem struct { Mail domain.Mail; ReadErr string; Files []FixtureAttachment }
type FixtureFolder struct { Name string; Items []FixtureItem; Children []FixtureFolder }
type Fixture struct { Stores []FixtureFolder }               // nivel raíz = cuentas o stores
type Options struct { ScanDelay, SaveDelay time.Duration }
func New(f Fixture, o Options) *Source
func DefaultFixture(now time.Time, seed int64) Fixture
```

- `DefaultFixture` (determinista para el mismo `now` y `seed`) reproduce el árbol del mock:
  - store `demo@example.com` con `Bandeja de entrada` (1248), `Elementos enviados` (402), `Facturas proveedores` (312), `Proyectos` (0 elementos) con hijos `Obra Norte` (86) y `Obra Sur` (41), y `Archivo 2025` (2910, fechado en 2025);
  - store `Archivo local (PST)` con `Importados` (120);
  - fechas repartidas en los 120 días anteriores a `now`.
- `Facturas proveedores` contiene:
  - correos `Factura F001-00NNN` con `factura_F001-00NNN.pdf` (80–400 KB) y `.xml` (8–20 KB); cerca del 15 % llevan además `estado_cuenta_<mes>.xlsx`;
  - al menos 4 correos con un único adjunto `image001.png` de tipo `KindInline`;
  - al menos 3 elementos con `IsMail=false` (asunto `Invitación: revisión mensual`);
  - al menos 15 correos con un único `.docx`;
  - 1 adjunto `herramienta.exe` con `FailWith: "Adjunto bloqueado por Outlook"`;
  - 1 adjunto con `FailWith: "Error de E/S simulado"`;
  - estos dos adjuntos que fallan van como **segundo** adjunto de correos cuyo primer adjunto es un `.pdf` válido, para que esos correos cuenten igual en `MailsWithDownloads`;
  - 1 elemento con `ReadErr: "propiedad no disponible"`;
  - dos correos distintos con el mismo nombre de adjunto, `factura_F001-00223.pdf`.
- `FolderRef`: `StoreID` es el nombre del store y `EntryID` es la ruta completa con `\`. `SaveAttachment` escribe `Size` bytes deterministas (un patrón repetido) y respeta `ctx` y `SaveDelay`.

- [ ] **Paso 1: Escribir los tests que fallan:**
  - `TestDefaultFixtureDeterministic`: dos llamadas con el mismo `now` y `seed` dan `reflect.DeepEqual`.
  - `TestListRootsAndChildren`: las raíces son `["demo@example.com", "Archivo local (PST)"]`; los hijos de la primera incluyen `Facturas proveedores` con `ItemCount == 312`; `Proyectos.HasChildren == true`.
  - `TestScanOrderAndStop`: `Scan` en `Facturas proveedores` entrega 312 elementos con `ReceivedAt` no creciente; si `fn` devuelve `ErrStopScan` en el 10.º, `Scan` retorna `nil` tras exactamente 10 llamadas.
  - `TestScanReadErr`: hay exactamente un `ScanItem` con `Err != nil`.
  - `TestSaveAttachment`: guarda el adjunto 1 de un correo y el archivo mide `Size` bytes; el adjunto con `FailWith` devuelve un error cuyo `Error()` contiene ese texto; con `ctx` cancelado devuelve `context.Canceled`.
- [ ] **Paso 2:** `go test ./internal/source/...` → FALLA.
- [ ] **Paso 3: Implementar** `source.go`, `fixture.go` y `demo.go`. Usa `math/rand/v2` con `rand.NewPCG(uint64(seed), 0)` para que el resultado sea determinista. `Scan` ordena una copia por `ReceivedAt` descendente y espera `ScanDelay` entre elementos con un `select` sobre `ctx.Done()`.
- [ ] **Paso 4:** `go test ./internal/source/...` → PASA.
- [ ] **Paso 5: Commit** `feat(source): interfaz MailSource y fuente demo determinista`.

---

### Tarea 5: Filtrado por correo

**Archivos:**
- Crear: `internal/extract/filter.go`
- Test: `internal/extract/filter_test.go`

**Interfaces:**
- Consume: `domain.Mail`, `domain.Filters`, las constantes `domain.Reason*`.
- Produce:
  - `func dayBounds(f domain.Filters) (start, end time.Time)` (en `time.Local`)
  - `type rangeCheck int` con valores `inRange`, `beforeRange` y `afterRange`; `func checkRange(t, start, end time.Time) rangeCheck`
  - `func evaluate(m domain.Mail, f domain.Filters) (eligible []domain.Attachment, reason string)`: con `reason == ""` hay al menos un adjunto elegible.

- Orden de evaluación (fija qué motivo se informa):
  1. `!IsMail` → `ReasonNotMail`
  2. el asunto no contiene el texto → `ReasonSubject`
  3. el remitente no contiene el texto → `ReasonSender`
  4. 0 adjuntos → `ReasonNoAttachments`
  5. se descartan `KindOther` y, si `SkipInline`, los `KindInline`. Si no queda ninguno: `ReasonOnlyInline` cuando todos eran inline, si no `ReasonUnsupportedKind`
  6. filtro de extensiones; si no queda ninguno → `ReasonNoMatchingExt`

  Las comparaciones de texto no distinguen mayúsculas (`strings.Contains(strings.ToLower(…))`); un filtro vacío deja pasar todo. `KindEmbeddedMsg` cuenta como extensión `.msg`.

- [ ] **Paso 1: Escribir los tests de tabla que fallan:**
  - `TestCheckRange` (Foco de revisión n.º 4): con `From = 2026-08-01` y `To = 2026-10-31` en `time.Local`:
    - `2026-08-01 00:00:00` → `inRange`
    - `2026-10-31 23:59:59` → `inRange`
    - `2026-11-01 00:00:00` → `afterRange`
    - `2026-07-31 23:59:59` → `beforeRange`
  - `TestEvaluate`, un caso por cada motivo del orden anterior, más:
    - asunto `"FACTURA F001"` con filtro `"factura"` → elegible;
    - `.PDF` con extensiones `[".pdf"]` → elegible;
    - correo con `[image001.png (inline), f.pdf]` y `SkipInline` → elegible solo `f.pdf`;
    - `KindEmbeddedMsg` `reunion.msg` con extensiones `[".msg"]` → elegible.
- [ ] **Paso 2:** `go test ./internal/extract/ -run 'TestCheckRange|TestEvaluate'` → FALLA.
- [ ] **Paso 3: Implementar** `filter.go`.
- [ ] **Paso 4:** el mismo comando → PASA.
- [ ] **Paso 5: Commit** `feat(extract): evaluación de rango y filtros por correo`.

---

### Tarea 6: Pipeline de extracción y reporte Excel

**Archivos:**
- Crear: `internal/extract/extract.go`, `internal/report/excel.go`
- Test: `internal/extract/extract_test.go`, `internal/report/excel_test.go`

**Interfaces:**
- Consume: Tareas 2–5.
- Produce:

```go
// internal/extract
type Phase string
const ( PhaseConnecting Phase = "connecting"; PhaseScanning = "scanning"; PhaseDownloading = "downloading"; PhaseReporting = "reporting" )
type Progress struct {
    Phase Phase; Scanned, InRange, MailsDone, MailsTotal, Saved, Skipped, Errors int
    Bytes int64; ElapsedMs int64
}
type ActivityKind string
const ( ActSaved ActivityKind = "saved"; ActRenamed = "renamed"; ActSkipped = "skipped"; ActError = "error" )
type Activity struct { At time.Time; Kind ActivityKind; Name, Subfolder, Reason string; Size int64 }
type Sink interface { Progress(Progress); Activity(Activity) }
type Result struct { Summary domain.Summary; Records []domain.RunRecord; ReportPath string }
func Run(ctx context.Context, src source.MailSource, req domain.JobRequest, sink Sink, now func() time.Time) (Result, error)

// internal/report
func FileName(t time.Time) string // "reporte_2006-01-02_1504.xlsx"
func WriteExcel(path string, req domain.JobRequest, res []domain.RunRecord, sum domain.Summary) error
```

- Comportamiento de `Run`:
  1. `req.Validate()`; si falla, devuelve el error sin escanear.
  2. Fase `scanning`: recorre la carpeta y, con `IncludeSubfolders`, sus subcarpetas en profundidad (vía `ListChildren`). Por cada `ScanItem`:
     - con `Err` → registro `error` con `ReasonReadError`, y cuenta en `InRange` (no se puede saber su fecha, así que se reporta de forma conservadora);
     - después del rango → `OutOfRange++`;
     - antes del rango → `OutOfRange++`, y tras 100 consecutivos devuelve `ErrStopScan` para esa carpeta;
     - en rango → `evaluate`: con motivo, registro `skipped`; si no, se agrega a la lista de candidatos.
  3. Fase `downloading`: por cada candidato y adjunto elegible:
     - subcarpeta `""` o `ReceivedAt.In(time.Local).Format("2006-01")`;
     - `SanitizeFileName` → `FitPath(dir, name, MaxPath)` → `UniquePath`;
     - `SaveAttachment` y `os.Stat` para el tamaño;
     - registro `saved` y actividad `ActRenamed` si el nombre final difiere del sanitizado.

     Un error de guardado genera un registro `error` con `err.Error()` como motivo y el proceso continúa.
  4. Antes de cada elemento, si `ctx` está cancelado, se corta el bucle y se marca `Summary.Cancelled = true`.
  5. Fase `reporting`: escribe siempre el Excel en `DestDir/report.FileName(now())`, también al cancelar. Devuelve `ctx.Err()` si se canceló.
- Hojas del Excel:
  - `Resumen`: pares clave/valor con origen, rango, filtros, destino, inicio, fin y cada contador de `Summary`.
  - `Adjuntos`: columnas `Fecha correo | Remitente | Asunto | Archivo | Ruta guardada | Tamaño (KB)`.
  - `Omitidos`: columnas `Estado | Motivo | Fecha correo | Remitente | Asunto | Archivo`.
  - Encabezado en negrita, texto blanco y relleno `#0369A1`; panel de encabezado fijo; ancho de columna = máx(10, min(60, el texto más largo + 2)).

- [ ] **Paso 1: Escribir los tests que fallan** (en `extract_test.go`, contra `demo.New(demo.DefaultFixture(now, 42), demo.Options{})` y un `Sink` de captura):
  - `TestRunFacturasProveedores`:
    - `Summary.InRange == Summary.MailsWithDownloads + Summary.Skipped + 1` (el 1 es el elemento con `ReadErr`). `Skipped` cuenta registros `skipped`, que son siempre de nivel correo. `Errors` cuenta los registros `error` (de lectura y de guardado) y aquí vale 3;
    - existe un registro con `ReasonNotMail`, uno con `ReasonOnlyInline`, uno con `ReasonReadError` y dos con status `error` (el `.exe` y el de E/S);
    - cada `SavedPath` existe en disco;
    - ambos `factura_F001-00223.pdf` existen, uno renombrado `_1`, y hay un `Activity` `ActRenamed`;
    - el Excel existe en `res.ReportPath`.
  - `TestRunByMonth`: con `OrganizeByMonth`, cada `SavedPath` tiene como directorio padre un nombre que coincide con `^\d{4}-\d{2}$`.
  - `TestRunExtensionsFilter`: con `Extensions [".xml"]`, todos los guardados terminan en `.xml`.
  - `TestRunExistingFiles` (Foco n.º 2): con `factura_F001-00218.pdf` creado de antemano con contenido `"previo"` en el destino, después de `Run` ese archivo conserva `"previo"`.
  - `TestRunCancel`: un `Sink` que cancela `ctx` al recibir el 5.º `ActSaved` → `errors.Is(err, context.Canceled)`, `Summary.Cancelled == true`, `Saved` entre 5 y 6, y el reporte existe.
  - `TestRunIncludeSubfolders`: sobre `Proyectos` (0 elementos propios), sin el flag `Scanned == 0`; con el flag, `Scanned == 127`.
  - `TestRunInvalidDest` (Foco n.º 5): con `DestDir` apuntando a un archivo → `ErrDestIsFile` y el `Sink` no recibe ningún `Progress` con `PhaseScanning`.

  En `excel_test.go`, `TestWriteExcel` reabre el archivo con `excelize.OpenFile` y comprueba:
  - las hojas `["Resumen", "Adjuntos", "Omitidos"]`;
  - `GetCellValue("Adjuntos", "D1") == "Archivo"`;
  - `GetCellValue("Adjuntos", "D2")` igual al nombre del primer registro `saved`;
  - el número de filas de `Omitidos` igual a 1 más los registros `skipped` y `error`.
- [ ] **Paso 2:** `go test ./internal/extract/... ./internal/report/...` → FALLA.
- [ ] **Paso 3: Implementar** `report/excel.go` (con `go get github.com/xuri/excelize/v2`) y `extract/extract.go`. El `Progress` se emite después de cada elemento escaneado y de cada adjunto; limitar la frecuencia no es responsabilidad de `extract`.
- [ ] **Paso 4:** el mismo comando → PASA. Además, `go vet ./...` queda limpio.
- [ ] **Paso 5: Commit** `feat(extract): pipeline escanear-descargar-reportar y reporte Excel`.

---

### Tarea 7: Gestor de jobs y eventos

**Archivos:**
- Crear: `internal/jobs/manager.go`
- Test: `internal/jobs/manager_test.go`

**Interfaces:**
- Consume: `extract.Run`, `extract.Sink`, `extract.Progress`, `extract.Activity`, `source.MailSource`.
- Produce:

```go
const ( EventProgress = "job:progress"; EventActivity = "job:activity"; EventDone = "job:done"; EventError = "job:error"; EventCancelled = "job:cancelled" )
type Emitter interface { Emit(event string, payload any) }
type Terminal struct { JobID string; Summary domain.Summary; ReportPath string; Message string }
var ErrJobRunning = errors.New("Ya hay una descarga en curso")
var ErrUnknownJob = errors.New("Job desconocido")
type Manager struct { /* … */ }
func NewManager(src source.MailSource, em Emitter, interval time.Duration, log *slog.Logger) *Manager
func (m *Manager) Start(req domain.JobRequest) (jobID string, err error)
func (m *Manager) Cancel(jobID string) error
func (m *Manager) Running() bool
```

- Comportamiento:
  - Solo un job a la vez, con un `jobID` aleatorio en hexadecimal de 16 caracteres.
  - El job corre en un goroutine con `context.WithCancel`.
  - `job:progress` se emite como mucho una vez por `interval`; el último `Progress` se emite siempre antes del evento terminal.
  - Las `Activity` se acumulan y se emiten como `[]extract.Activity` en `job:activity` como mucho una vez por `interval`; los pendientes se vacían antes del terminal.
  - Exactamente un evento terminal (`Terminal` como payload): `job:done` si `err == nil`, `job:cancelled` si `errors.Is(err, context.Canceled)` y `job:error` en cualquier otro caso, con `Message = err.Error()`.
  - Un `panic` dentro del job se recupera y se convierte en `job:error` con `Message = "Error interno: " + fmt.Sprint(r)`.
  - `Running()` pasa a `false` antes de emitir el evento terminal.

- [ ] **Paso 1: Escribir los tests que fallan.** Usan un `Emitter` falso con un mutex que registra `(event, payload)` en orden, la fuente demo con `SaveDelay: 2*time.Millisecond` y un `t.TempDir()`. Esperan el terminal con un canal o un `Eventually` de 5 s.
  - `TestStartDone`: exactamente un terminal, que es `job:done` y es el último evento; el evento anterior relacionado con progreso es un `job:progress` con `Phase == "reporting"`.
  - `TestThrottle`: con `interval = 50ms`, `count(job:progress) < 1 + duración/50ms + 2`.
  - `TestCancel` (Foco n.º 3): `Cancel(id)` tras el primer `job:activity` → un único terminal `job:cancelled` con un `ReportPath` existente; luego `Start` acepta un job nuevo.
  - `TestSecondStart`: `Start` mientras hay otro job activo → `ErrJobRunning`.
  - `TestInvalidRequest`: un `DestDir` vacío → un único terminal `job:error` con `Message == domain.ErrNoDest.Error()`.
  - `TestPanic`: con una `MailSource` falsa cuyo `Scan` hace panic → un único `job:error` y `Running() == false`.
  - `TestCancelUnknown`: `Cancel("x")` → `ErrUnknownJob`.
- [ ] **Paso 2:** `go test -race ./internal/jobs/...` → FALLA.
- [ ] **Paso 3: Implementar** `manager.go`. El `Sink` interno guarda el último `Progress` y las actividades pendientes bajo un mutex, y un `time.Ticker` hace el flush.
- [ ] **Paso 4:** `go test -race ./internal/jobs/...` → PASA, sin carreras.
- [ ] **Paso 5: Commit** `feat(jobs): gestor de jobs con throttling y evento terminal garantizado`.

---

### Tarea 8: Configuración y logging

**Archivos:**
- Crear: `internal/config/config.go`, `internal/logging/logging.go`
- Test: `internal/config/config_test.go`, `internal/logging/logging_test.go`

**Interfaces:**
- Produce:

```go
// internal/config
type Config struct {
    LastDestDir  string          `json:"lastDestDir"`
    Organize     domain.Organize `json:"organize"`
    SkipInline   bool            `json:"skipInline"`
    WindowWidth  int             `json:"windowWidth"`
    WindowHeight int             `json:"windowHeight"`
}
func Default() Config                 // Organize by_month, SkipInline true, 1280×720
func Dir() (string, error)            // os.UserConfigDir()/outlook_descargas_adjuntos
func Load(dir string) (Config, error) // archivo inexistente → Default(), sin error
func Save(dir string, c Config) error // escritura atómica: tmp + os.Rename

// internal/logging
func LogDir() (string, error)          // os.UserCacheDir()/outlook_descargas_adjuntos/logs  (= %LOCALAPPDATA%)
func Setup(dir string, now time.Time) (*slog.Logger, io.Closer, error)
```

- `Setup` crea `sesion_YYYYMMDD_HHMMSS.log` con `slog.NewTextHandler` en nivel Info. Antes borra los `sesion_*.log` con más de 30 días y deja como máximo los 20 más recientes.

- [ ] **Paso 1: Escribir los tests que fallan:**
  - `TestLoadMissing`: `Load` de un directorio vacío → `Default()`.
  - `TestSaveLoadRoundTrip`: los valores guardados se leen iguales.
  - `TestLoadCorrupt`: un JSON inválido → `Default()` y un error no nulo.
  - `TestSetupRetention`: en un `t.TempDir()` con 25 logs de hace 1 a 25 días y 3 de hace 40 días (con `Chtimes`), después de `Setup` quedan exactamente 20 archivos `sesion_*.log`: los 19 más nuevos más el nuevo.
- [ ] **Paso 2:** `go test ./internal/config/... ./internal/logging/...` → FALLA.
- [ ] **Paso 3: Implementar** ambos paquetes.
- [ ] **Paso 4:** el mismo comando → PASA.
- [ ] **Paso 5: Commit** `feat: configuración en AppData y logs con retención`.

---

### Tarea 9: Bindings de Wails, selección de fuente y pantalla de humo

**Archivos:**
- Modificar: `main.go`, `app.go` (reemplazan el `Greet` del template)
- Crear: `dto.go`, `dto_test.go`
- Modificar: `frontend/src/App.tsx`; borrar `frontend/src/App.css` y los assets del template que ya no se usen

**Interfaces:**
- Consume: Tareas 2–8.
- Produce (lo usa el Plan 2 desde `wailsjs/go/main/App` y `wailsjs/go/models`):

```go
type JobRequestDTO struct {
    Folder domain.FolderRef `json:"folder"`; From string `json:"from"`; To string `json:"to"`  // "2006-01-02"
    SubjectContains string `json:"subjectContains"`; SenderContains string `json:"senderContains"`
    Extensions []string `json:"extensions"`; SkipInline bool `json:"skipInline"`; IncludeSubfolders bool `json:"includeSubfolders"`
    DestDir string `json:"destDir"`; Organize string `json:"organize"`
}
func (d JobRequestDTO) ToDomain() (domain.JobRequest, error) // fechas en time.Local; error "Fecha inválida: <valor>"
type SourceInfo struct { Name string `json:"name"`; Demo bool `json:"demo"` }

func (a *App) SourceInfo() SourceInfo
func (a *App) ListRoots() ([]domain.Folder, error)
func (a *App) ListChildren(ref domain.FolderRef) ([]domain.Folder, error)
func (a *App) PickDirectory() (string, error)        // runtime.OpenDirectoryDialog
func (a *App) StartJob(req JobRequestDTO) (string, error)
func (a *App) CancelJob(jobID string) error
func (a *App) OpenPath(path string) error            // exec.Command("explorer", path).Start()
func (a *App) GetConfig() config.Config
func (a *App) SaveConfig(c config.Config) error
func selectSource(args []string) (source.MailSource, error)
```

- Selección de fuente:
  - `--demo` en `os.Args` → `demo.New(demo.DefaultFixture(time.Now(), 42), demo.Options{ScanDelay: 10*time.Millisecond, SaveDelay: 90*time.Millisecond})`;
  - sin el flag → error `"La fuente Outlook llega en el Plan 3; ejecute con --demo"`. `main` registra el error en el log y arranca igualmente, con `SourceInfo{Name: "ninguna"}`.
- El `Emitter` de producción envuelve `runtime.EventsEmit(a.ctx, …)`. Opciones de `wails.Run`:
  - `Title: "outlook_descargas_adjuntos"`;
  - `Width`/`Height` desde la config (1280×720 por defecto), con `MinWidth: 1100` y `MinHeight: 680`;
  - `Frameless: false` por ahora (la barra de título propia llega en el Plan 2).
  - En `OnShutdown`, si hay un job activo, se cancela.

- [ ] **Paso 1: Escribir los tests que fallan** en `dto_test.go`:
  - `TestToDomain`: `From "2026-08-01"`, `To "2026-10-31"` → `Filters.From.Location() == time.Local`; `Extensions [" PDF"]` se normaliza a `[".pdf"]`; `Organize "by_month"` → `domain.OrganizeByMonth`.
  - `TestToDomainBadDate`: `From "31/10/2026"` → un error que contiene `"Fecha inválida"`.
  - `TestSelectSource`: `["app", "--demo"]` → `Name() == "demo"`; `["app"]` → error.
- [ ] **Paso 2:** `go test . -run 'TestToDomain|TestSelectSource'` → FALLA.
- [ ] **Paso 3: Implementar** `dto.go`, `app.go` y `main.go`. La pantalla de humo de `App.tsx` es temporal, sin estilos y con texto en español:
  - muestra `SourceInfo`;
  - lista las raíces de `ListRoots` y los hijos de la primera raíz al hacer clic;
  - un botón "Probar descarga demo" pide destino con `PickDirectory` y llama a `StartJob` sobre `Facturas proveedores` con el rango de los últimos 120 días;
  - muestra en un `<pre>` el último `job:progress`, cuenta los `job:activity` y muestra el evento terminal recibido.
- [ ] **Paso 4: Verificar:**
  - `go test ./... && go vet ./...` → PASA;
  - `wails build -clean` → OK, y `frontend/wailsjs/go/models.ts` contiene `JobRequestDTO` y `Folder`;
  - verificación manual: `wails dev -appargs "--demo"` → aparecen las dos raíces; "Probar descarga demo" en una carpeta temporal termina con `job:done`, y la carpeta contiene subcarpetas `AAAA-MM` y `reporte_*.xlsx`; repetir y cancelar a mitad muestra `job:cancelled` y permite iniciar otra vez.
- [ ] **Paso 5: Commit** `feat(app): bindings Wails, fuente demo por flag y pantalla de humo`.

---

## Autorrevisión (hecha)

- **Cobertura del alcance (AGENTS.md, MVP):**
  - carpeta y subcarpetas → Tareas 4 y 6;
  - rango obligatorio → Tareas 2 y 5;
  - filtros de asunto, remitente y extensiones → Tarea 5;
  - omitir inline → Tarea 5;
  - organización plana o por mes y duplicados `_N` → Tareas 3 y 6;
  - progreso por fase, actividad y cancelación → Tareas 6 y 7;
  - reporte Excel con hoja de omitidos → Tarea 6;
  - modo demo → Tareas 4 y 9;
  - config y logs en AppData → Tarea 8.
  - Quedan fuera de este plan, a propósito: UI final y tema (Plan 2); suspensión, beep y flash (Plan 3, `internal/winapi`); confirmación al cerrar (Plan 2, con `OnBeforeClose`).
- **Bugs del legacy cubiertos:** B4 (`ReasonReadError` y `ReasonNotMail` quedan registrados), B7 (evento terminal garantizado), B9 (sin búsquedas lineales; los registros se agregan), B14 (un solo modelo de cancelación con `ctx`), B15 (las estadísticas viajan en los eventos), B16 (progreso por fase), B17 (filtro por `Kind`), B18 (Tarea 3).
- **B10** (auditoría solo en memoria) queda **mitigado, no resuelto**: el reporte se escribe siempre al final, incluso al cancelar, pero un crash del proceso lo pierde. Si hace falta, la escritura incremental es candidata para después del Plan 3.
