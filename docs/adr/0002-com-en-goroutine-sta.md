# ADR 0002 — Todo el acceso a COM en un único goroutine STA

- **Estado:** aceptada (2026-11-19)

## Contexto

El modelo de objetos de Outlook es COM con apartment single-threaded: un objeto solo se puede usar desde el
hilo que lo creó, y ese hilo tiene que inicializar COM. En Go los goroutines saltan entre hilos del sistema.
El original además acumulaba objetos COM sin liberar y se degradaba tras miles de correos.

## Decisión

- Un goroutine dueño de COM ([internal/outlook/sta.go](../../internal/outlook/sta.go)) hace `LockOSThread`,
  `CoInitializeEx(COINIT_APARTMENTTHREADED)` y atiende llamadas por un canal (`do`). Ningún `*ole.IDispatch`
  sale de él; un panic dentro de una llamada se convierte en error y el STA sigue vivo.
- El recorrido de una carpeta usa `stream`: el STA produce los correos ya convertidos a tipos de `domain` y el
  goroutine del job los consume. Cancelar, parar o un panic del consumidor liberan el STA.
- Helpers en [internal/outlook/com.go](../../internal/outlook/com.go) (`getObj`, `callObj`, `release`...)
  liberan cada `IDispatch` y `VARIANT`. Los tests lo verifican contando referencias sobre `Scripting.Dictionary`.
- Las carpetas se identifican por `EntryID` + `StoreID`. Los errores de desconexión (Outlook cerrado) sueltan la
  conexión para reconectar en la siguiente llamada.
- `Close` espera como mucho 2 s: si Outlook está bloqueado (por ejemplo, con un diálogo modal), se abandona el STA.

## Consecuencias

- No hay llamadas COM concurrentes: el árbol de carpetas espera si hay un recorrido en curso.
- El adaptador se prueba sin Outlook (STA, streaming, helpers) y contra Outlook real con tests de integración que corre el usuario.
