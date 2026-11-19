# ADR 0003 — Modelo de jobs: `context`, eventos limitados y un evento terminal

- **Estado:** aceptada (2026-11-12)

## Contexto

En el original, cancelar dejaba la UI bloqueada porque el worker suprimía la señal de fin, y la UI leía las
estadísticas del worker sin sincronización. Además, un job de miles de correos puede generar miles de
actualizaciones por segundo, demasiadas para el WebView.

## Decisión

- `StartJob` valida, lanza el job en un goroutine y devuelve un `jobID` al instante. Hay un solo job a la vez
  ([internal/jobs/manager.go](../../internal/jobs/manager.go)).
- La cancelación es un `context.Context`. No hay flags compartidos.
- El progreso y la actividad se acumulan en un sink que publica como mucho cada 100 ms (`job:progress`,
  `job:activity`). Las estadísticas viajan en el payload.
- Todo job termina con **exactamente un** evento terminal: `job:done`, `job:error` o `job:cancelled`, también si
  hay un panic. `GetJobStatus` devuelve el mismo estado para una UI que se recarga a mitad del job.
- Mientras dura el job, `winapi.KeepAwake` evita la suspensión desde un hilo fijo y la restaura al terminar.

## Consecuencias

- La UI es un reductor de eventos ([frontend/src/lib/job.ts](../../frontend/src/lib/job.ts)) que siempre llega a un estado final.
- Cerrar la ventana con un job activo pide confirmación, cancela y espera a que se escriba el reporte parcial.
