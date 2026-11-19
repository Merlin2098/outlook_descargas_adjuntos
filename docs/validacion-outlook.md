# Validación local con Outlook Classic

Checklist manual para comprobar la app contra un Outlook real. **No se graba ni se publica nada de este proceso:**
no guardes en el repo capturas, reportes ni archivos del buzón real. El video del portafolio usa solo el modo demo
(ver [demo-y-portafolio.md](demo-y-portafolio.md) §2).

Requisitos: Windows con **Outlook Classic** abierto y con su caché local (OST) sincronizada. El New Outlook no expone COM.

## 1. Tests de integración

```bash
go test -tags integration ./internal/outlook/ -v
```

- [ ] `TestIntegrationListRoots` pasa (lista cuentas y PST con StoreID y EntryID).
- [ ] `TestIntegrationScanFirst50` pasa (50 items de la Bandeja de entrada en orden descendente).
- [ ] `TestIntegrationSaveFirstAttachment` pasa o se salta si no hay adjuntos.
- [ ] La salida solo muestra conteos (ningún asunto, remitente ni nombre de archivo).

## 2. Spike de rendimiento y memoria (B11)

```bash
go run ./cmd/spike-outlook -n 10000
go run ./cmd/spike-outlook -n 10000 -folder "cuenta\Carpeta grande"
```

- [ ] Abre el Administrador de tareas y anota la memoria de `OUTLOOK.EXE` al inicio y al final.
- [ ] La memoria de Outlook no crece sin control a lo largo de la corrida (el original se degradaba tras miles de correos).
- [ ] Anota los items/s en la tabla de resultados.

## 3. App sin `--demo`

```bash
.tools/wails.exe dev
```

- [ ] El árbol lista las cuentas y los PST; las subcarpetas cargan al expandir sin congelar la ventana.
- [ ] Con Outlook cerrado, el árbol muestra "No se pudo conectar con Outlook Classic…"; al abrir Outlook y pulsar recargar, aparece.
- [ ] Extraer un rango corto **por mes**: los archivos quedan en `YYYY-MM\` y se genera `reporte_*.xlsx`.
- [ ] Extraer el mismo rango en modo **plano** en otra carpeta de destino.
- [ ] Repetir sobre el mismo destino: los duplicados se renombran con `_1`, `_2`… y nada se sobrescribe.
- [ ] El filtro de extensiones (`pdf, xlsx`) solo guarda esas extensiones.
- [ ] En un buzón Exchange, el filtro "remitente contiene" con la dirección SMTP de un remitente **interno** lo encuentra, y el reporte muestra su dirección SMTP (no un `/O=…/CN=…`).
- [ ] Cerrar la app mientras Outlook muestra un diálogo modal no deja el proceso colgado (termina en unos segundos).
- [ ] Con "Omitir imágenes inline", las imágenes de firma (`image001.png`) no se guardan y aparecen como "Solo imágenes inline".
- [ ] Un correo con otro correo adjunto guarda el adjunto como `.msg`.
- [ ] Un adjunto bloqueado por Outlook (por ejemplo, `.exe`) queda como error en el reporte y el job sigue.
- [ ] Las convocatorias de reunión o los informes de entrega aparecen como "No es un correo".
- [ ] Un correo recibido cerca de la medianoche entra en el día correcto del rango.
- [ ] Cancelar a mitad: la UI vuelve al resumen "Descarga cancelada" en menos de 1 s y se puede relanzar.
- [ ] Durante el job, `powercfg /requests` (en una consola de administrador) muestra la app en SYSTEM y DISPLAY; al terminar ya no.
- [ ] Con la ventana minimizada, al terminar suena el aviso y el botón de la barra de tareas parpadea.
- [ ] Alt+F4 con un job activo pide confirmación; "Seguir descargando" no cierra; "Cancelar y salir" cierra tras escribir el reporte parcial.
- [ ] La ventana se arrastra desde la barra de título y los botones minimizar, maximizar y cerrar funcionan.
- [ ] "Abrir carpeta" funciona con un destino cuya ruta tiene una coma (`D:\Pruebas,2026`). "Abrir reporte" abre el Excel.

## 4. Resultados (para el benchmark D5)

Publica solo cifras, nunca datos del buzón.

| Medida | Legacy (Python) | Rewrite (Go) |
| --- | --- | --- |
| Items/s al recorrer una carpeta de 10.000 correos | | |
| Memoria de OUTLOOK.EXE al inicio → al final | | |
| Tiempo total de una extracción de un mes | | |
| Tamaño del ejecutable | | |
