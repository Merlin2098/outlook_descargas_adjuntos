# ADR 0004 — Fuente demo determinista detrás de `MailSource`

- **Estado:** aceptada (2026-11-12)

## Contexto

No se puede grabar ni publicar nada de un buzón real, los tests tienen que correr sin Outlook (en cualquier
equipo) y la UI necesita datos con volumen y errores realistas para desarrollarse y para el video.

## Decisión

- La interfaz [internal/source/source.go](../../internal/source/source.go) (`MailSource`) separa la lógica
  de la fuente: `extract` nunca importa el adaptador de Outlook.
- [internal/source/demo](../../internal/source/demo/) implementa `MailSource` en memoria: cuentas y carpetas
  ficticias (`demo@example.com`), cientos de correos repartidos en varios meses, adjuntos PDF, XML, XLSX, DOCX y
  PNG inline (archivos válidos que se pueden abrir: una factura PDF con su XML, un estado de cuenta XLSX…), un item que no es correo, un correo ilegible y un adjunto que falla al guardarse.
- Los datos salen de una semilla fija y se sitúan respecto de la fecha actual (los últimos meses): para una misma
  fecha, la ejecución produce siempre el mismo resultado.
- La latencia simulada solo se activa en la app (`--demo`) para que el progreso se vea; los tests corren sin ella.

## Consecuencias

- `go test ./...` cubre filtrado, descarga, nombres repetidos, cancelación y reporte sin Outlook.
- El video, las capturas y el README usan solo esta fuente. La conexión real se valida en local con
  [docs/validacion-outlook.md](../validacion-outlook.md).
