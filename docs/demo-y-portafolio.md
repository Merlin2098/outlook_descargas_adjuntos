# Demo y portafolio

Cómo se presenta el proyecto: qué destacar, cómo conseguir datos ficticios para el video y qué
debe tener el repositorio público.

## 1. Qué tiene que demostrar

| Aspecto | Dónde se ve |
| --- | --- |
| Go + Wails: backend nativo con UI web | Binario único y pequeño; arranque rápido |
| Concurrencia en Go | Job en goroutine, cancelación con `context`, eventos limitados (throttle) a la UI |
| Interop Windows desde Go | COM con go-ole (Outlook), Win32 (suspensión, parpadeo de ventana) |
| Bindings tipados Go → TypeScript | `wailsjs/go/models` usados en React |
| Robustez | Errores por correo que no abortan el job, reporte de omitidos con su motivo, cancelación limpia |
| Testabilidad | Interfaz `MailSource` + fuente demo → `go test ./...` sin Outlook |

## 2. Datos de demostración

Nunca se graba un buzón real. El video usa solo la fuente demo (§2.1); Outlook real se valida en local (§2.2).

### 2.1 Fuente demo en la app (`--demo`)

- Implementación de `MailSource` en memoria: cuentas y carpetas ficticias, unos 300 correos repartidos en
  varios meses, adjuntos generados (PDF, XLSX, DOCX, PNG inline) y algunos casos de error a propósito
  (un item que no es correo, un adjunto que falla al guardarse).
- Simula latencia por correo para que el progreso se vea en el video.
- Sirve para desarrollar la UI, para los tests y para que cualquiera pruebe la app sin Outlook.
- En la UI se indica con una etiqueta "Modo demo".

### 2.2 Validación local con Outlook real (no se graba)

**Decisión: el video se graba en modo demo.** La conexión con Outlook real se valida localmente,
sin grabarla y sin publicar ningún resultado:

- Checklist manual (`docs/validacion-outlook.md`, por crear): listar cuentas y carpetas, extraer un
  rango corto, cancelar a mitad del job, revisar el reporte y medir tiempo y memoria en una carpeta grande.
- Tests `//go:build windows && integration` para el adaptador COM.
- El PST demo deja de ser necesario. Solo se haría si más adelante se quiere un benchmark reproducible.

## 3. Guion del video (60–90 s, 1280×720)

1. Arranque de la app (tema oscuro), con la etiqueta de la versión visible.
2. Explorar Outlook: abrir el árbol de cuentas y carpetas (carga asíncrona) y elegir la carpeta demo.
3. Configurar: rango de fechas, filtro por extensiones (`pdf, xlsx`), organización por mes.
4. Iniciar: progreso por fase, contador de correos y adjuntos, registro de actividad en pantalla.
5. Cancelar y relanzar (opcional): mostrar que la UI se recupera al instante.
6. Fin: resumen, beep y parpadeo de la ventana, botón "Abrir carpeta". Mostrar los archivos en el Explorador y el Excel de la ejecución.
7. Cierre: terminal con `go test ./...` en verde y el tamaño del binario.

Herramientas de grabación: OBS o la Xbox Game Bar. Exportar el MP4 para el README y un GIF corto (≤ 10 MB) para la vista previa de GitHub.

## 4. Repositorio público

- `README.md` (idioma según D1): GIF o video, qué hace, diagrama de arquitectura, decisiones técnicas
  (COM en un goroutine STA, modelo de jobs, eventos), cómo ejecutarlo (`--demo`), limitaciones
  (solo Outlook Classic, solo Windows) y la sección "Rewrite desde Python", con las mejoras medidas.
- ADRs en `docs/adr/` con las decisiones técnicas relevantes.
- CI: GitHub Actions en `windows-latest` con `go vet`, `go test ./...` y `npm run build`, más el badge en el README.
- Releases con el `.exe` firmado o, si no está firmado, con una nota sobre SmartScreen.
- Licencia MIT, igual que el original.
- Revisar que no queden datos, nombres ni rutas del entorno corporativo (`git grep` antes de publicar).

## 5. Decisiones

| # | Pregunta | Estado |
| --- | --- | --- |
| D1 | Idioma de la UI y del README | ✅ **Español** (2026-11-12) |
| D2 | Video con Outlook real o con el modo demo | ✅ **Modo demo**; Outlook real solo se valida localmente (2026-11-12) |
| D3 | Formato del reporte | Abierta. Recomendación: solo Excel; CSV como opción secundaria |
| D4 | Nombre público del proyecto | ✅ **`outlook_descargas_adjuntos`** (2026-11-12) |
| D5 | Benchmark legacy vs. rewrite | Abierta. Recomendación: medir sobre una carpeta real durante la validación local y publicar solo las cifras |
