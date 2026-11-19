# Diagnóstico del proyecto legacy — Outlook AttachExtractor (Python)

- **Fecha**: 2026-11-12
- **Alcance**: `Outlook_AttachExtractor_Rework/` (11 commits, ~8.400 líneas incluyendo docs)
- **Propósito**: servir de referencia técnica para la reescritura en Go + Wails + React.

> **Cambio de alcance (2026-11-12).** El rewrite es una **descarga genérica de adjuntos** para portafolio.
> El clasificador de firmas, los modos de búsqueda por frases y el filtro sobre el nombre del adjunto
> **no se portan**. Este documento los conserva solo como registro histórico (§4.1 parcial y §4.3) y
> marca qué bugs dejan de aplicar (§5). El alcance vigente está en `AGENTS.md`, y el plan de demo en
> `docs/demo-y-portafolio.md`.

---

## 1. Resumen ejecutivo

La aplicación cumple su función principal (extraer adjuntos de Outlook Classic con auditoría por
`EntryID`), pero el código promete más de lo que implementa. Los problemas que **sí afectan al rewrite**
(descarga genérica) son técnicos:

- **Cancelar deja la UI bloqueada**: el botón "Iniciar" queda deshabilitado hasta reiniciar la app (B7).
- **Correos que se pierden sin registro**: si un item falla al leerse, se salta sin entrar en la auditoría (B4).
- **Acumulación de objetos COM**: el proceso se degrada o se detiene tras miles de correos (B11).
- **Estado compartido entre hilos sin sincronización** y una auditoría que vive solo en memoria (B10, B15).
- **Carpetas resueltas por nombre** y archivos de log y config ligados al cwd o al `.exe` (B12, B13).

Además había defectos en las reglas de negocio (el clasificador de firmas y los modos de búsqueda ignorados),
que quedan fuera del alcance nuevo y se documentan solo como registro histórico.

La arquitectura en capas (UI → Workers → Core) es razonable y se puede mantener conceptualmente.
La deuda está en el manejo de COM, la concurrencia y la falta total de tests, y eso es justamente lo que
el rewrite en Go puede lucir.

---

## 2. Contexto de negocio

- Antes había una **macro VBA** (ADR 0000) con poca visibilidad del progreso, sin auditoría y difícil de mantener.
- Caso de uso original: control documental interno, con separación de documentos firmados y no firmados (ADR 0004).
  **No aplica al rewrite.**
- Usuarios no técnicos, buzones grandes (miles de correos) y ejecuciones largas en desatendido.
- Solo **Outlook Classic** (COM/MAPI). No funciona con el New Outlook. Procesa solo lo que está en la caché local (OST).

---

## 3. Stack y arquitectura actual

| Capa | Tecnología | Archivos clave |
| --- | --- | --- |
| UI | PySide6 (Qt), temas JSON, splash | `main.py`, `ui/main_window.py`, `ui/tabs/*`, `ui/widgets/*` |
| Workers | `QRunnable` + `QThreadPool` + signals | `workers/base_worker.py`, `extractor_worker.py`, `classifier_worker.py` |
| Core | Template method `BackendBase` | `core/backend_base.py`, `core/email_extractor.py`, `core/sign_classifier.py` |
| Utils | Auditoría (Polars), fechas, logger singleton, energía y notificación (ctypes) | `utils/*.py` |
| Config | `config.json` junto al ejecutable | `config/config_manager.py` |
| Build | PyInstaller onedir + runtime hook COM | `generar_onedir.py`, `runtime_hook_com.py` |

Flujo de datos del extractor:

```text
TabExtractor ──params──▶ ExtractorWorker (QRunnable, CoInitialize)
                              │ callbacks (mensaje, progreso, estado)
                              ▼
                    ExtractorAdjuntosOutlook
                    1. Dispatch("Outlook.Application") → MAPI
                    2. Resolver carpeta por ruta de nombres "Cuenta\Bandeja\Sub"
                    3. FILTRADO: Items.Sort(ReceivedTime desc) → recorrer 1..Count
                       ├─ fecha fuera de rango → auditar RECHAZADO (early exit tras 100 seguidos anteriores al inicio)
                       ├─ asunto OR frases  ∧  tiene adjuntos → candidato
                       └─ resto → auditar RECHAZADO con motivo
                    4. DESCARGA: GetItemFromID(entry_id) → por adjunto:
                       nombre_archivo OR frases → SaveAsFile (con sufijo _N si existe)
                    5. lista_documentos_<ts>.xlsx + auditoria_correos_<ts>.{parquet,xlsx}
```

Flujo del clasificador: lista los archivos del **nivel superior** de la carpeta de origen y los **mueve**
(`shutil.move`) a `Documentos Firmados/` o `Documentos sin Firmar/`, creadas **dentro de la misma carpeta de origen**.

---

## 4. Comportamiento funcional del legacy (referencia)

> Del extractor se conserva la mecánica (rango de fechas por días completos, orden descendente con
> early exit, relectura por `EntryID`, sufijo `_N` para duplicados) y lo operativo de §4.4. Las filas de
> frases y filtro dual de §4.1, los formatos de §4.2 y todo §4.3 son **solo históricos**: los reemplaza
> el alcance de `AGENTS.md`.

### 4.1 Extractor

| Regla | Detalle legacy | Ubicación |
| --- | --- | --- |
| Parámetros obligatorios | carpeta destino, carpeta Outlook, rango de fechas | `email_extractor.py:79-100` |
| Rango máximo | 365 días | `email_extractor.py:96-98` |
| Días completos | inicio 00:00:00, fin 23:59:59.999999 | `date_handler.py:28-53` |
| Orden | `Items.Sort("[ReceivedTime]", True)` (más recientes primero) | `email_extractor.py:288-289` |
| Early exit | 100 correos **consecutivos** anteriores a la fecha de inicio → corta | `email_extractor.py:334-364` |
| Filtro de asunto | `any(frase in asunto.lower())`; sin frases = todos | `email_extractor.py:389-395` |
| Requisito de adjuntos | Solo se aceptan correos con `Attachments.Count > 0` | `email_extractor.py:398` |
| **Filtro dual** | Además, el **nombre del adjunto** debe contener alguna frase | `email_extractor.py:540-547` |
| Relectura por ID | Descarga vía `GetItemFromID(EntryID)` y avisa si el asunto no coincide | `email_extractor.py:501-512` |
| Nombres duplicados | `nombre_1.ext`, `nombre_2.ext`… y timestamp después de 1000 | `backend_base.py:249-279` |
| Parseo de frases | Separador `\|`; comillas `"..."` permiten `\|` literal; se recortan espacios; advertencia si hay comillas sin cerrar | `phrase_search_widget.py:152-195` |
| Fechas por defecto en la UI | hoy − 30 días → hoy | `date_range_widget.py:39,51` |

### 4.2 Salidas del extractor (las consumen los usuarios)

**`lista_documentos_dd.mm.YYYY_HH.MM.SS.xlsx`**: hoja "Lista de Documentos", tabla `TablaDocumentos`
(estilo TableStyleMedium9), encabezado `#16A085` con texto blanco. Columnas:
`Nº | Nombre del archivo | Fecha de descarga (dd/mm/YYYY HH:MM:SS) | Fecha correo (dd/mm/YYYY) | Hora correo (HH:MM:SS)`.
Solo se genera si se descargó al menos un archivo.

**`auditoria_correos_<ts>.parquet` y `.xlsx`** (hoja "Auditoria"), un registro por correo evaluado:

| Columna | Tipo | Valores |
| --- | --- | --- |
| `entry_id` | str | EntryID o `UNKNOWN_<idx>` |
| `received_time` | datetime naive | |
| `subject`, `sender` | str | `sender` = `UNKNOWN` si falla |
| `cumple_fecha`, `cumple_frases`, `tiene_adjuntos` | bool | |
| `num_adjuntos` | int | |
| `adjuntos_nombres` | str (JSON array) | `["ERROR_READING_ATTACHMENTS"]` si falla |
| `adjuntos_descargados` | int | |
| `estado_final` | str | `PROCESADO` \| `RECHAZADO` \| `ERROR` |
| `motivo_rechazo` | str | p. ej. "Fuera de rango de fechas (anterior al inicio)", "No coincide con frases de búsqueda \| No tiene adjuntos", "Adjuntos no coinciden con frases de filtrado" |
| `fase_proceso` | str | `FILTRADO` \| `DESCARGA` |

### 4.3 Clasificador

- Patrones de firmado (`sign_classifier.py:43-46`): `\bfirmad[oa]s?\b` y `(?<!not[\s_-])\bsigned\b`, sin distinguir mayúsculas.
  **Todo lo que no coincide va a "sin firmar"** (los patrones `sin_firmar`/`not_signed` del ADR 0004 no se evalúan explícitamente).
- Carpetas: `Documentos Firmados` y `Documentos sin Firmar` dentro del origen.
- Valida que el origen exista y tenga permisos de escritura.
- Estadísticas: `total, firmados, sin_firmar, omitidos, errores, tiempo_total`.

### 4.4 Operación

- Evita la suspensión del equipo y el apagado de pantalla durante la extracción
  (`SetThreadExecutionState(ES_CONTINUOUS|ES_SYSTEM_REQUIRED|ES_DISPLAY_REQUIRED)`).
  **Solo en el extractor**, no en el clasificador.
- Al terminar: `MessageBeep(MB_ICONASTERISK)` + `FlashWindowEx` si la ventana no está en primer plano.
- Logs: `OutlookExtractor_sesion_<ts>.log` y `OutlookExtractor_errores_<ts>.log`; se conservan 50 pares y se borran los de más de 30 días.
- Al cerrar con un job activo pide confirmación. Guarda el tamaño de la ventana y el tema (claro/oscuro).
- Selector de carpetas de Outlook en árbol con carga diferida (lazy loading) de subcarpetas.

---

## 5. Bugs y defectos conocidos (NO portar)

Severidad: 🔴 afecta el resultado de negocio · 🟠 afecta la operación o la UX · 🟡 menor.

**Con el nuevo alcance dejan de aplicar** B1, B2, B3, B5, B6 (clasificador y frases), B8 (pestaña del
clasificador) y la parte de B19 que corresponde al clasificador. **Siguen aplicando**, porque son de COM,
concurrencia y E/S: B4, B7, B9–B18, B20 y B21. Las lecciones de B6 (nunca sobrescribir) y de B4 (nada se
omite sin registro) se mantienen como principios generales.

| # | Sev. | Defecto | Evidencia | Corrección esperada en Go |
| --- | --- | --- | --- | --- |
| B1 | 🔴 | `\b` no separa en `_` (`_` es carácter de palabra): `contrato_firmado.pdf`, `CONTRATO_FIRMADO.pdf` y `doc_signed.pdf` → **sin firmar** | Verificado ejecutando el regex | Tokenizar el nombre por `[\s_\-.]+` y evaluar los tokens. Tests de tabla con todos los casos |
| B2 | 🔴 | Falso positivo: `no firmado.pdf` / `no_firmado` → **firmados** (solo existe un lookbehind para `not signed`) | Verificado | Evaluar primero las negaciones (`no/sin firmar`, `not/un signed`, `unsigned`) y después las afirmaciones |
| B3 | 🔴 | Los modos "Todas las palabras / Alguna / Frase exacta" se recogen en la UI pero **nunca llegan al backend**; siempre se aplica OR. El default de la UI ("Todas") contradice lo que ocurre | `tab_extractor.py:278` solo lo loguea; `ExtractorWorker` no lo recibe | Pasar el modo al core e implementarlo (ver D1) |
| B4 | 🔴 | Los correos cuyo procesamiento lanza una excepción en el filtrado (p. ej. items que no son MailItem y no tienen `ReceivedTime`) se registran en el log y se saltan con `continue` **sin entrar en la auditoría** | `email_extractor.py:466-468` | Registrar siempre: `estado=ERROR` u `OMITIDO` con el motivo |
| B5 | 🔴 | El clasificador mueve **todos** los archivos del origen, incluidos `lista_documentos_*.xlsx` y `auditoria_correos_*` si se ejecuta sobre la salida del extractor | `sign_classifier.py:96` | Excluir los artefactos propios (o escribirlos en una subcarpeta `_reportes/`) |
| B6 | 🔴 | Posible sobrescritura silenciosa en el clasificador: en Windows, `shutil.move` hace `copy2` + `unlink` si `rename` falla, y pisa el archivo de destino si ya existe (por ejemplo, de una ejecución anterior) | `sign_classifier.py:200,212` | Resolver el nombre único (`_N`) también al mover |
| B7 | 🟠 | Al cancelar, el worker suprime `finished`/`error` → la UI nunca vuelve al estado inactivo | `base_worker.py:82-89`; `tab_extractor.py:282-287` | Evento terminal obligatorio `job:cancelled` |
| B8 | 🟠 | `TabClasificador` no tiene `is_running()` → al cerrar la app no se confirma ni se cancela la clasificación en curso | `main_window.py:297` (`hasattr`) | Registro central de jobs activos |
| B9 | 🟠 | La auditoría actualiza buscando linealmente por `entry_id` → O(n²) en buzones grandes | `audit_mails.py:109-116` | Usar un mapa `entry_id → índice` |
| B10 | 🟠 | La auditoría vive en memoria y solo se escribe al final: si el proceso se cae, se pierde toda | `audit_mails.py` | Escritura incremental (checkpoint periódico) |
| B11 | 🟠 | Acumulación de objetos COM en ejecuciones largas (documentado en el README como "se detiene tras miles de correos"); solo se mitiga con `gc.collect()` cada 100 | `email_extractor.py:611-613` | `Release()` determinista de cada IDispatch |
| B12 | 🟠 | Carpeta de Outlook resuelta por **nombre**: falla con cuentas o carpetas homónimas y con `\` en el nombre | `email_extractor.py:222-242` | `EntryID` + `StoreID` |
| B13 | 🟠 | Logs en `cwd/logs` y config junto al `.exe`: fallan si la app se instala en Program Files o se lanza desde un acceso directo | `logger.py:46`, `config_manager.py:65-68` | `%LOCALAPPDATA%` / `%APPDATA%` |
| B14 | 🟠 | El extractor no implementa pausa (la base sí tiene `pausar()`) y usa su propio flag `_cancelado` en lugar del `Event` de la base; no usa el template `ejecutar()` | `email_extractor.py:71,102-104` | Un único modelo de job con `context` |
| B15 | 🟠 | La UI lee `worker.extractor.estadisticas` desde el hilo principal mientras el worker la modifica (condición de carrera) | `tab_extractor.py:357-364` | Las estadísticas viajan en el payload del evento |
| B16 | 🟡 | La barra de progreso reutiliza 0–100 % para dos fases; en el filtrado el total es `Items.Count` de toda la carpeta, aunque el early exit corte antes | `email_extractor.py:463-464,609` | Progreso por fase y total indeterminado durante el filtrado |
| B17 | 🟡 | Se descargan adjuntos inline o embebidos (imágenes de firma) en modo sin filtro; no se revisa `Attachment.Type` | `email_extractor.py:535` | Filtrar por tipo (ver D4) |
| B18 | 🟡 | No se sanean los nombres de archivo ni se controla la longitud de ruta (>260 caracteres) | `email_extractor.py:549` | Saneamiento + prefijo `\\?\` o truncado |
| B19 | 🟡 | `omitidos` nunca se incrementa; `max_reintentos` y `solo_cache_local` existen en la config pero no se usan | `sign_classifier.py:32`, `email_extractor.py:73-77` | Eliminar o implementar |
| B20 | 🟡 | El runtime hook borra la caché `gen_py` de win32com en cada arranque | `runtime_hook_com.py` | No aplica en Go |
| B21 | 🟡 | El selector de carpetas de Outlook hace llamadas COM en el hilo de la UI (se congela con árboles grandes) | `outlook_folder_selector.py:60-61,167` | Llamadas en el goroutine COM; la UI pide los hijos de forma asíncrona |

---

## 6. Brechas entre documentación y código

| Lo que dice el README / ADR | Realidad |
| --- | --- |
| Tres modos de búsqueda (todas / alguna / ninguna) | Solo existen OR y "sin filtro" (B3) |
| "Date range detection based on available mailbox contents" | No implementado; la UI usa hoy − 30 días por defecto |
| Clasificador: "select the destination folder" | No hay destino: crea las subcarpetas dentro del origen |
| "Safe duplicate handling with optional subfolder organization" en el clasificador | No hay manejo de duplicados (B6) ni opción de subcarpetas |
| Patrones `sin_firmar`, `not_signed` | No se evalúan; todo lo que no coincide va a "sin firmar" por defecto |
| Ejemplo de `config.json` con `"tema": "claro"` y `window_size` | El real usa `"dark"/"light"` y `ui.window.width/height` |
| Versión 3.0.0 | `main.py` declara 1.0.0 |
| Suite de tests (`pytest`, `pytest-qt` en requirements) | **No existe ningún test** |

`requirements.txt` incluye dependencias que no se usan (pandas, pyarrow, duckdb, pydantic, hypothesis,
jsonschema, etc.) agregadas por el instalador del framework "Tinker". `agents2/` es un resto de ese
framework y no forma parte de la app.

---

## 7. Decisiones abiertas para el rewrite

Ya resueltas por el cambio de alcance: sin modos de frases, sin filtro dual, sin clasificador y sin Parquet
(el reporte es Excel). Quedan estas decisiones técnicas; las de presentación (idioma, video, nombre) están
en `docs/demo-y-portafolio.md` §5.

| # | Pregunta | Opciones | Recomendación |
| --- | --- | --- | --- |
| T1 | Adjuntos `.msg` embebidos | Guardar como `.msg` / extraer recursivamente sus adjuntos | Guardar como `.msg` (simple y predecible) |
| T2 | Optimización del filtrado | Recorrido completo + early exit (legacy) / `Items.Restrict` por fecha | Probar `Restrict` en un spike (el formato de fecha depende del locale); mantener el early exit como fallback. Buen material para el README si se mide |
| T3 | Límite de 365 días | Mantener / quitar | Quitarlo y mostrar una advertencia si el rango es grande |
| T4 | Incluir subcarpetas | Sí (recursivo) / No | Opción desactivada por defecto |

---

## 8. Riesgos técnicos de la migración a Go + Wails

1. **COM en Go (go-ole)** es de bajo nivel: no hay early binding y hay que gestionar `VARIANT`, `Release`
   y la afinidad de hilo a mano. Mitigación: hacer un spike temprano (listar carpetas, iterar 10.000
   correos, guardar adjuntos) y medir memoria y tiempo frente al legacy.
2. **Outlook Object Model Guard**: leer `SenderEmailAddress` puede disparar el aviso de seguridad si no hay
   antivirus reconocido. Hay que validarlo en el entorno del usuario.
3. **Fechas `VT_DATE`**: verificar que la conversión de go-ole respeta la hora local y no la desplaza por UTC.
4. **Volumen de eventos** a la UI de Wails: aplicar throttling para no saturar el WebView2.
5. **WebView2** debe estar presente (viene con Windows 11; en Windows 10 puede requerir el bootstrapper del instalador de Wails).
6. **Excel**: excelize permite tablas con estilo, pero no tiene autofit nativo; el ancho de las columnas se aproxima calculándolo.
7. **Datos para el video**: resuelto. Se graba en modo demo y Outlook real solo se valida localmente (ver `docs/demo-y-portafolio.md` §2.2).

---

## 9. Plan sugerido

1. **Fundaciones**: scaffold de Wails (`react-ts`), `internal/domain`, logging y config en AppData.
2. **Fuente demo + extractor con TDD**: interfaz `MailSource`, fuente simulada con errores inyectados y el job de extracción con eventos y cancelación (B4, B7, B9, B10, B14, B15).
3. **UI React contra el modo demo**: así hay algo visible pronto y se itera el diseño pensando en el video.
4. **Spike COM**: goroutine STA, listar carpetas por EntryID/StoreID, iterar 10.000 correos con `Release` y medir memoria (B11, B12).
5. **Fuente Outlook real** detrás de la misma interfaz, más el reporte Excel.
6. **Operación**: prevención de suspensión, beep y parpadeo de la ventana, confirmación al cerrar.
7. **Validación local con Outlook real**: checklist manual y tests de integración, sin grabación.
8. **Portafolio**: README, CI, release, benchmark legacy vs. rewrite y grabación del video.
