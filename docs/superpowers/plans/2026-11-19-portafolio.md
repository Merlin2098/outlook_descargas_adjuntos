# Plan 4 — Portafolio: tema oscuro, pulido, CI, README y publicación: plan de implementación

> **Para agentes:** SUB-SKILL REQUERIDA: usa superpowers:subagent-driven-development (recomendado) o
> superpowers:executing-plans para ejecutar este plan tarea por tarea. Los pasos usan casillas (`- [ ]`).

**Objetivo:** dejar el repositorio listo para publicarse como proyecto de portafolio: tema oscuro (último punto del
MVP), las mejoras menores del Plan 3, el ícono y los metadatos del ejecutable, la CI y la release en GitHub Actions,
las capturas, el README, los ADRs, una auditoría de datos sensibles y un historial limpio con la identidad pública
del autor. **No se hace push ni se publica nada:** eso lo hace el usuario.

**Arquitectura:** el tema se resuelve con variables CSS bajo `:root[data-theme="dark"]` y se persiste en `config.json`.
La CI corre en `windows-latest` con los mismos comandos que se usan en local. La release se dispara con un tag `v*`,
compila con el Wails CLI v2.17.1 e inyecta la versión con `-ldflags`. El historial se reescribe una sola vez, al final,
con `git filter-branch`, después de un respaldo en `git bundle`.

**Stack:** Go 1.27, Wails v2.17.1, React 18 + Vite 5 + Vitest 2, GitHub Actions (`actions/checkout@v4`,
`actions/setup-go@v5`, `actions/setup-node@v4`) y `gh`, que ya viene en los runners.

**Especificación:** [AGENTS.md](../../../AGENTS.md) (MVP: "Tema claro/oscuro"; "Portafolio y datos"; "Límites"),
[docs/demo-y-portafolio.md](../../demo-y-portafolio.md) §1, §3 y §4 (qué demostrar, guion, repositorio público) y
la sección "Deferred minors" del cierre del Plan 3 (M1–M8).

## Decisiones de este plan

| # | Decisión | Motivo |
| --- | --- | --- |
| D-a | Tema oscuro derivado de `theme_dark.json` del original, con dos ajustes de accesibilidad: los botones primarios llevan texto oscuro `#0D0F22` sobre `#38BDF8` (el blanco del original no llega a 4,5:1) y los textos de estado se aclaran | Contraste AA |
| D-b | Selector en la barra de título (sol/luna) y persistencia en `config.json` (`theme`: `"light"` o `"dark"`, por defecto `"light"`) | El original guardaba el tema |
| D-c | M4 (leer adjuntos de correos fuera de rango) **no** se optimiza: queda en "Limitaciones" del README y se mide en el benchmark D5 | Requiere cambiar la interfaz `MailSource`; primero hay que medir |
| D-d | El GIF y el video los graba el usuario siguiendo el guion. El README usa capturas en modo demo hasta que exista `docs/img/demo.gif` | Grabar requiere OBS y una persona |
| D-e | Las cifras del benchmark legacy vs. rewrite las aporta el usuario tras la validación local; el README no publica números inventados ni vacíos | D5 en demo-y-portafolio.md |
| D-f | `outlook_descargas_adjuntos.exe` (versionado por error en el primer commit, con rutas locales de compilación) sale del repo y del historial | Datos del entorno local |
| D-g | Los datos de identidad (nombre público, email público o el noreply de GitHub, usuario de GitHub) los pide el ejecutor al usuario **al empezar la Tarea 8**. La reescritura del historial necesita además su confirmación explícita | Es irreversible y el dato es del usuario |

## Restricciones globales

- Solo datos del modo demo en capturas, README y ADRs. Nada del buzón real, ni el nombre de la empresa original, ni rutas o usuarios del equipo local.
- Español en la UI, el README y `docs/`. Código en inglés.
- Sin push, sin crear repositorios remotos, sin publicar releases: los workflows se escriben y se validan en local.
- La CI usa exactamente: `npm ci`, `npm test`, `npm run build` (en `frontend/`), `gofmt -l .` sin salida, `go vet ./...`, `go vet -tags integration ./...` y `go test ./...`. `frontend/dist` debe existir antes de compilar Go (`//go:embed all:frontend/dist`).
- Versiones fijadas: Wails CLI `v2.17.1`, Node `24`, Go desde `go.mod`.
- `go test ./...`, `cd frontend && npm test` y `npm run build` pasan al cierre de cada tarea que los toque.
- Cada commit termina con `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Foco de revisión

1. **Contraste del tema oscuro**: ningún texto queda ilegible (botones primarios, carpeta seleccionada, insignia demo, filas de error, chips de extensión), y el tema elegido sobrevive a reiniciar la app (Tarea 1, `TestConfigTheme` y la verificación con capturas).
2. **CI que pasa en local pero falla en el runner**: `frontend/dist` inexistente al compilar Go, `npm ci` sin lockfile actualizado o `gofmt` con CRLF. Los pasos del workflow se ejecutan en local en el mismo orden (Tarea 4, paso de ensayo).
3. **Datos sensibles que se cuelan en la publicación**: el `.exe` con rutas locales, emails, la palabra de la empresa, rutas `C:\Users\…` en docs o capturas (Tarea 7, auditoría con `git grep` sobre **todo** el historial después de la reescritura).
4. **`stream` después de `Close` o con `fn` en panic**: no se cuelga y el STA queda libre (Tarea 2, `TestStreamAfterClose`, `TestStreamFnPanic`).
5. **La reescritura del historial pierde trabajo**: el respaldo `git bundle` se verifica antes (`git bundle verify`), el número de commits y el árbol final (`git diff` vacío contra el respaldo, salvo el `.exe`) coinciden (Tarea 8).

---

### Task 1: Tema oscuro

**Archivos:**
- Modificar: `internal/config/config.go` (+test), `frontend/src/styles/tokens.css`, `frontend/src/components/TitleBar.tsx`, `frontend/src/components/Icon.tsx`, `frontend/src/App.tsx`, `frontend/src/i18n/es.ts`
- Crear: `frontend/src/lib/theme.ts` (+test)

**Interfaces:**
- Produce:
  ```go
  // config.Config
  Theme string `json:"theme"` // "light" | "dark"; Load normaliza cualquier otro valor a "light"
  ```
  ```ts
  export type Theme = 'light' | 'dark'
  export function normalizeTheme(v: unknown): Theme      // 'dark' → 'dark'; cualquier otro → 'light'
  export function toggleTheme(t: Theme): Theme
  export function applyTheme(t: Theme): void              // document.documentElement.dataset.theme = t
  ```

- [ ] **Paso 1: tests que fallan.**
  - `TestConfigTheme` (config): `Default().Theme == "light"`; guardar `Theme: "dark"` y cargar devuelve `"dark"`; un `config.json` con `"theme": "morado"` carga como `"light"`.
  - `theme.test.ts`: `normalizeTheme('dark') === 'dark'`, `normalizeTheme('x') === 'light'`, `normalizeTheme(undefined) === 'light'`, `toggleTheme('light') === 'dark'` y al revés.
- [ ] **Paso 2:** `go test ./internal/config/` y `npm test` → FAIL.
- [ ] **Paso 3:** implementar. `tokens.css` agrega `:root[data-theme="dark"] { … }` que redefine **todas** las variables con estos valores:
  - Base: `--bg #0D0F22`, `--surface #1B2233`, `--surface-alt #252D42`, `--border #334155`, `--border-soft #2C3549`, `--text #E2E8F0`, `--text-2 #94A3B8`, `--text-3 #7C8BA1`.
  - Primario y estados: `--primary #38BDF8`, `--primary-strong #38BDF8`, `--on-primary #0D0F22`, `--accent #E879F9`, `--success #22C55E`, `--warning #FBBF24`, `--error #F43F5E`, `--info #38BDF8`.
  - Texto de estado: `--success-text #4ADE80`, `--warning-text #FCD34D`, `--error-text #FB7185`.
  - Tonos suaves: `--primary-soft rgba(56,189,248,0.15)`, `--primary-ink #7DD3FC`, `--muted-ink #64748B`, `--ink-strong #E2E8F0`, `--success-soft rgba(34,197,94,0.12)`, `--success-border rgba(34,197,94,0.35)`, `--success-ink #86EFAC`, `--warning-soft rgba(251,191,36,0.12)`, `--warning-border rgba(251,191,36,0.35)`, `--warning-ink #FDE68A`, `--error-soft rgba(244,63,94,0.12)`, `--error-border rgba(244,63,94,0.35)`, `--error-ink #FDA4AF`, `--error-row #FDA4AF`, `--accent-soft rgba(232,121,249,0.12)`, `--accent-ink #F0ABFC`, `--info-soft rgba(56,189,248,0.10)`, `--info-border rgba(56,189,248,0.30)`, `--info-ink #BAE6FD`, `--info-icon #38BDF8`.
  - Las sombras se apagan con una variable `--shadow` (`0 1px 2px rgba(0,0,0,0.05)` en claro, `none` en oscuro) que reemplaza los `box-shadow` literales de `app.css`.

  `App` lee `GetConfig().theme` al montar y llama a `applyTheme`. `TitleBar` agrega un botón (iconos `sun`/`moon`, `aria-label` `es.titleBar.themeToDark` / `themeToLight`) que alterna, aplica y guarda con `SaveConfig({...cfg, theme})`. `main.go` usa `BackgroundColour` según `cfg.Theme` (13,15,34 en oscuro).
- [ ] **Paso 4:** `go test ./... && cd frontend && npm test && npm run build` → PASS. Con `wails dev -appargs "--demo"` abierto en `http://localhost:34115` a 1280×720, capturar las tres pantallas en oscuro (en `.playwright-mcp/`) y revisar el contraste de los elementos del Foco 1.
- [ ] **Paso 5:** commit `feat: tema oscuro con selector en la barra de título y persistencia`.

### Task 2: Mejoras menores del Plan 3

**Archivos:**
- Modificar: `internal/outlook/{sta.go,outlook.go,convert.go}` (+tests), `internal/winapi/winapi.go` (+test), `app.go`

**Interfaces:**
- Produce:
  ```go
  // outlook
  const storesMax = 100_000
  func (s *Source) rememberStore(entryID, storeID string) // vacía el mapa al superar storesMax (GetItemFromID funciona sin StoreID, solo más lento)
  func comError(op string, err error) error              // "Outlook devolvió un error al <op>: <detalle>"; envuelve err
  // winapi
  func FindOwnWindow(title string) uintptr               // EnumWindows + GetWindowThreadProcessId == GetCurrentProcessId
  ```

- [ ] **Paso 1: tests que fallan.**
  - **`TestStreamAfterClose`** (M1): con el STA cerrado, `stream` con `context.Background()` vuelve `errSTAClosed` en menos de 1 s.
  - **`TestStreamFnPanic`** (M2): `fn` hace panic en el item 3; el test recupera el panic y comprueba que una `do` posterior responde en menos de 1 s (el productor no quedó bloqueado).
  - `TestRememberStoreBounded` (M5): tras `storesMax + 1` llamadas, `len(s.stores) <= storesMax` y la última entrada está.
  - `TestComError` (M8): `comError("listar carpetas", io.EOF)` contiene "Outlook devolvió un error al listar carpetas" y `errors.Is(…, io.EOF)`.
  - `TestFindOwnWindowMissing` (M6): `FindOwnWindow("título-que-no-existe-123") == 0`.
- [ ] **Paso 2:** `go test ./internal/outlook/ ./internal/winapi/` → FAIL.
- [ ] **Paso 3:** implementar.
  - M1: el goroutine de `stream` marca si `produce` llegó a correr y, si `do` volvió sin correrla, cierra `items`.
  - M2: `defer stop()` justo después de crear `stop`.
  - M3: corregir el comentario de `stream`: `fn` no debe llamar a la fuente y las demás llamadas esperan a que termine el recorrido.
  - M5: usar `rememberStore` en `readItem`.
  - M6: `app.go` usa `FindOwnWindow`.
  - M7: `OpenPath` corre `ShellExecute` en un goroutine con `LockOSThread` y `windows.CoInitializeEx(0, COINIT_APARTMENTTHREADED|COINIT_DISABLE_OLE1DDE)` + `CoUninitialize`, y espera su resultado.
  - M8: los errores que no son de desconexión en `ListRoots`, `ListChildren`, `Scan` (nivel carpeta) y `SaveAttachment` pasan por `comError` con la operación en español.
- [ ] **Paso 4:** `go test -race ./internal/outlook/ ./internal/winapi/ && go test ./... && go vet -tags integration ./...` → PASS.
- [ ] **Paso 5:** commit `fix: mejoras menores del Plan 3 (stream, mapa de stores, ventana propia, ShellExecute, errores en español)`.

### Task 3: Ícono, metadatos y versión del ejecutable

**Archivos:**
- Crear: `docs/diseno/icono.svg`
- Reemplazar: `build/appicon.png` (1024×1024); borrar `build/windows/icon.ico` para que Wails lo regenere
- Modificar: `wails.json` (bloque `info`), `dto.go` (`appVersion` pasa a `var`), `.gitignore`

**Interfaces:**
- Produce: `var appVersion = "0.1.0"` en `package main`, sobrescribible con `-ldflags "-X main.appVersion=X.Y.Z"`.

- [ ] **Paso 1:** `docs/diseno/icono.svg`: cuadrado de 1024 con radio 224, degradado `#0284C7` → `#0369A1`, la flecha de descarga del logo de la barra de título en blanco (trazo 88, puntas redondeadas) y, detrás, un sobre estilizado en `rgba(255,255,255,0.35)`. Renderizarlo a PNG de 1024×1024 con fondo transparente usando Playwright (`page.setContent` + `page.screenshot({omitBackground: true})`) y guardarlo en `build/appicon.png`.
- [ ] **Paso 2:** `wails.json` agrega `"info": {"companyName": "", "productName": "outlook_descargas_adjuntos", "productVersion": "0.1.0", "comments": "Descarga en bloque los adjuntos de una carpeta de Outlook Classic"}`. El `copyright` se completa en la Tarea 8. `.gitignore` agrega `*.exe`. Sacar del índice el ejecutable de la raíz con `git rm --cached outlook_descargas_adjuntos.exe` (el historial se limpia en la Tarea 8).
- [ ] **Paso 3:** `TestSourceInfoVersion` sigue pasando; agregar `TestAppVersionLdflags`: `go build -ldflags "-X main.appVersion=9.9.9" -o <tmp>.exe .` y luego ejecutar `<tmp>.exe --version`, que imprime `9.9.9` y sale con 0. Implica agregar a `main.go` el flag `--version`, que imprime `appVersion` y sale antes de `wails.Run`. El test vive en `version_test.go` con `testing.Short()` → skip.
- [ ] **Paso 4:** `go test ./...` → PASS; `.tools/wails.exe build -clean -trimpath` genera `build/bin/outlook_descargas_adjuntos.exe` con el ícono nuevo (comprobar que `build/windows/icon.ico` se regeneró).
- [ ] **Paso 5:** commit `build: ícono propio, metadatos del ejecutable y versión inyectable`.

### Task 4: CI y release en GitHub Actions

**Archivos:**
- Crear: `.github/workflows/ci.yml`, `.github/workflows/release.yml`

- [ ] **Paso 1:** `ci.yml`: en `push` y `pull_request` a `main`, `runs-on: windows-latest`, `permissions: contents: read`. Pasos:
  1. `actions/checkout@v4`.
  2. `actions/setup-go@v5` con `go-version-file: go.mod` y caché.
  3. `actions/setup-node@v4` con `node-version: 24`, `cache: npm` y `cache-dependency-path: frontend/package-lock.json`.
  4. En `frontend/`: `npm ci`, `npm test`, `npm run build`.
  5. `gofmt -l .` debe salir vacío (paso en `bash`, que falla si hay salida).
  6. `go vet ./...` y `go vet -tags integration ./...`.
  7. `go test ./...`.
- [ ] **Paso 2:** `release.yml`: en `push` de tags `v*`, `permissions: contents: write`. Pasos:
  1. Los mismos de preparación que `ci.yml`.
  2. `go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.1`.
  3. `wails build -clean -trimpath -ldflags "-X main.appVersion=${GITHUB_REF_NAME#v}"`.
  4. `gh release create "$GITHUB_REF_NAME" build/bin/outlook_descargas_adjuntos.exe --title "$GITHUB_REF_NAME" --generate-notes`, con `GH_TOKEN: ${{ github.token }}`.
  5. Una nota en el cuerpo sobre SmartScreen (el binario no está firmado).
- [ ] **Paso 3: ensayo local** de los pasos de `ci.yml`, en el mismo orden y en un clon limpio: `git clone . <scratchpad>/ci-dry && cd <scratchpad>/ci-dry && (cd frontend && npm ci && npm test && npm run build) && gofmt -l . && go vet ./... && go vet -tags integration ./... && go test ./...` → todo PASS y `gofmt` sin salida. Validar que los dos YAML parsean (`python -c "import yaml,sys; yaml.safe_load(open(sys.argv[1]))"` o, si no hay PyYAML, `node -e` con el parser de un paquete ya instalado; si no hay ninguno, revisión manual registrada en el ledger).
- [ ] **Paso 4:** commit `ci: workflows de CI y release en windows-latest`.

### Task 5: Capturas del modo demo

**Archivos:**
- Crear: `docs/img/configurar.png`, `docs/img/en-curso.png`, `docs/img/resumen.png`, `docs/img/configurar-oscuro.png`, `docs/img/resumen-oscuro.png`

- [ ] **Paso 1:** con `.tools/wails.exe dev -appargs "--demo"`, abrir `http://localhost:34115` en Playwright a 1280×720. Como destino, escribir `C:\demo\adjuntos` (la app la crea; es una ruta neutra que puede aparecer en las capturas) y borrarla al terminar. Capturar:
  - Configurar con "Facturas proveedores", "Último trimestre", asunto `factura` y extensiones `.pdf .xml`.
  - En curso cerca del 50 %.
  - Resumen completado.
  - Configurar y Resumen en oscuro.
- [ ] **Paso 2:** revisar cada imagen: ningún usuario, ruta del equipo ni dato real. Optimizar el peso (PNG ≤ 400 KB cada una; si pasa, reducir con `scale: css`).
- [ ] **Paso 3:** commit `docs: capturas del modo demo`.

### Task 6: README y ADRs

**Archivos:**
- Crear: `README.md`, `docs/adr/0001-go-wails.md`, `docs/adr/0002-com-en-goroutine-sta.md`, `docs/adr/0003-jobs-y-eventos.md`, `docs/adr/0004-fuente-demo.md`
- Modificar: `docs/demo-y-portafolio.md` (guion: el tema oscuro ya existe; D3 resuelta = solo Excel), `build/README.md` (borrar o reemplazar el texto de plantilla de Wails)

- [ ] **Paso 1:** `README.md` en español, con estas secciones en este orden:
  1. Título y una línea de qué es.
  2. Badge de CI (se agrega en la Tarea 8, cuando se conoce el usuario de GitHub).
  3. Captura `docs/img/configurar.png` (o `docs/img/demo.gif` si existe).
  4. **Qué hace**: el MVP de AGENTS.md en viñetas.
  5. **Pruébalo sin Outlook**: requisitos, descargar la release o `wails dev -appargs "--demo"`.
  6. **Arquitectura**: diagrama Mermaid `flowchart` UI React ⇄ bindings/eventos Wails ⇄ App ⇄ jobs.Manager ⇄ extract ⇄ MailSource (demo | outlook en el STA) ⇄ Outlook COM; más `report` y `winapi`.
  7. **Decisiones técnicas**, un párrafo cada una con enlace a su ADR:
     - COM en un goroutine STA con streaming y liberación garantizada.
     - Jobs con cancelación por `context`, eventos limitados a unos 10 por segundo y exactamente un evento terminal.
     - Fechas OLE como hora de reloj.
     - Fuente demo determinista para tests y video.
     - Nada se omite sin registro.
  8. **Pruebas**: los comandos y qué cubren, incluidos los de integración que se corren en local.
  9. **Rewrite desde Python**: tabla con los bugs del original que se corrigieron (B4, B7, B9–B18, B21, con una línea cada uno) y un apartado "Rendimiento", que dice que las cifras se publican tras la validación local (D-e).
  10. **Limitaciones**: solo Windows y Outlook Classic, solo la caché local (OST), Object Model Guard, M4 y el binario sin firmar (SmartScreen).
  11. **Licencia** (MIT).
  12. Capturas en oscuro.

  Sin el nombre de la empresa original ni su contexto interno: "una herramienta interna hecha en Python".
- [ ] **Paso 2:** ADRs, formato corto (Contexto · Decisión · Consecuencias), 15–30 líneas cada uno, con enlaces al código.
- [ ] **Paso 3:** verificar los enlaces relativos del README y de los ADRs con un script que extraiga los `](...)` locales y compruebe que existen → sin rotos.
- [ ] **Paso 4:** commit `docs: README del portafolio y ADRs`.

### Task 7: Auditoría de publicación

**Archivos:**
- Modificar: lo que la auditoría encuentre (por ejemplo, `docs/diagnostico-legacy.md`)

- [ ] **Paso 1:** sobre el árbol actual (`git grep -n -i`), buscar: el dominio y el nombre de la empresa original, `ext.`, `@` seguido de dominios que no sean `example`/`example.com`/`anthropic.com`/`localhost.invalid`, `C:\\Users\\`, `C:/Users/`, `AppData`, `Temp\\claude`, `scratchpad`, `Ricardo`, y nombres de personas o clientes en fixtures. También binarios versionados (`git ls-files | grep -iE '\.(exe|dll|xlsx|pst|msg)$'`).
- [ ] **Paso 2:** `docs/diagnostico-legacy.md`: mantener el análisis técnico, quitar el contexto de negocio interno si identifica a la organización (dejar "control documental interno" genérico) y revisar que no queden nombres. `CLAUDE.md` y `AGENTS.md` se publican tal cual (son parte de la historia del proyecto); revisar que no tengan datos personales.
- [ ] **Paso 3:** registrar en el ledger cada hallazgo y su corrección; los que dependan del historial (el `.exe` y el autor de los commits) quedan para la Tarea 8.
- [ ] **Paso 4:** commit `docs: limpieza para la publicación`.

### Task 8: Identidad pública, licencia e historial limpio

**Archivos:**
- Crear: `LICENSE`
- Modificar: `wails.json` (`info.copyright`), `README.md` (badge de CI), configuración local de git

- [ ] **Paso 1: datos del usuario.** Pedir con AskUserQuestion:
  - Nombre público.
  - Email público (sugerir `<id>+<usuario>@users.noreply.github.com`).
  - Usuario de GitHub.
  - Confirmación explícita para reescribir **todo** el historial (autor de los 40+ commits y eliminación del `.exe`).

  Sin esa confirmación, la tarea termina en el Paso 3 y el resto queda pendiente.
- [ ] **Paso 2:** `LICENSE` MIT con `Copyright (c) 2026 <nombre>`; `wails.json` `info.copyright` = `© 2026 <nombre>`; README con el badge `https://github.com/<usuario>/outlook_descargas_adjuntos/actions/workflows/ci.yml/badge.svg`. Commit `docs: licencia MIT y badge de CI`.
- [ ] **Paso 3: respaldo.** `git bundle create <scratchpad>/respaldo-antes-de-reescribir.bundle --all` y `git bundle verify` → OK. Anotar el número de commits (`git rev-list --count HEAD`) y el hash del árbol (`git rev-parse HEAD^{tree}`).
- [ ] **Paso 4: reescritura.** `git config user.name/user.email` con los datos públicos. Luego `git filter-branch -f --index-filter 'git rm --cached --ignore-unmatch outlook_descargas_adjuntos.exe' --env-filter '<asigna GIT_AUTHOR_NAME/EMAIL y GIT_COMMITTER_NAME/EMAIL con los datos públicos>' -- --all`, seguido de `rm -rf .git/refs/original && git reflog expire --expire=now --all && git gc --prune=now`.
- [ ] **Paso 5: verificación.**
  - Mismo número de commits.
  - `git log --format='%an <%ae>|%cn <%ce>' | sort -u` muestra solo la identidad pública.
  - `git log --all --oneline -- outlook_descargas_adjuntos.exe` vacío.
  - `git diff <árbol anotado> HEAD --stat` muestra solo el `.exe` como diferencia.
  - Se repite la búsqueda de la Tarea 7 sobre **todo** el historial (`git grep -i <patrón> $(git rev-list --all)` para el dominio y el nombre de la empresa, `ext.`, `C:/Users/` y `localhost.invalid`): sin resultados.
  - `go test ./...` y `npm test` → PASS.
- [ ] **Paso 6:** `git tag -a v0.1.0 -m "Primera versión pública"`. **No** hacer push. En el mensaje final, dar los comandos para que el usuario cree el repo y publique: `gh repo create <usuario>/outlook_descargas_adjuntos --public --source . --push`, `git push origin v0.1.0` (dispara la release).

**Verificación manual del usuario (no la ejecuta el agente):** grabar el video y el GIF con el guion de
`docs/demo-y-portafolio.md` §3 y agregarlos a `docs/img/`; completar las cifras del benchmark tras `docs/validacion-outlook.md`;
crear el repo público y hacer push.
