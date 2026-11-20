# Grabación del video demo

El video de [docs/img/demo.mp4](../../docs/img/demo.mp4) y el GIF del README se graban de forma automatizada, en
**modo demo** (datos ficticios) y sin mostrar nada del equipo: ni el escritorio, ni el Explorador, ni Excel.

| Archivo | Qué hace |
| --- | --- |
| `comun.js` | Cursor visible con efecto al hacer clic, rótulos y el contexto de Playwright que graba a 1280×720 |
| `escena-app.js` | Escena 1: la app de punta a punta (tema, árbol, configuración, descarga, cancelar, relanzar, resumen) |
| `escena-tarjetas.js` | Escena 2: tarjetas finales con los datos reales de la descarga grabada |
| `tarjetas/` | Programa Go que genera las tarjetas: archivos en disco, hojas del Excel (con excelize), `go test` y tamaño del binario |
| `armar.mjs` | Une `comun.js` con cada escena en una sola función ejecutable por Playwright MCP |
| `montar.sh` | Une las escenas con ffmpeg y exporta `demo.mp4` y `demo.gif` |

## Pasos

Requisitos: Node, ffmpeg y Playwright MCP (la herramienta `browser_run_code` con `filename`).

1. Borrar la carpeta de destino de una toma anterior (`C:\demo\video`): el resumen lista lo que haya en ella.
2. Arrancar la app en modo demo sin recompilaciones automáticas, en un puerto libre:

   ```bash
   .tools/wails.exe dev -skipbindings -nogorebuild -m -nosyncgomod -devserver 127.0.0.1:34557 -appargs "--demo"
   ```

   Antes de cada toma hay que reiniciarlo: la app restaura el último job y la escena espera la pantalla de configurar.
   Si al arrancar aparece `bind: Solo se permite un uso...`, quedan procesos `outlook_descargas_adjuntos-dev.exe`
   vivos: hay que cerrarlos y volver a arrancar. El tema guardado debe ser el claro (la escena cambia a oscuro).
3. `node scripts/grabar-demo/armar.mjs` genera `.playwright-mcp/escena-*.run.js`.
4. Escena 1: navegar la página de Playwright a
   `about:blank#destino=C%3A%5Cdemo%5Cvideo&app=http%3A%2F%2F127.0.0.1%3A34557` y ejecutar
   `.playwright-mcp/escena-app.run.js`. El video queda en `.playwright-mcp/video/`.
5. Tarjetas con los datos de esa descarga:

   ```bash
   .tools/wails.exe build -clean -trimpath
   go test -count=1 ./... > gotest.txt
   go run ./scripts/grabar-demo/tarjetas -dest 'C:\demo\video' -out .playwright-mcp/tarjetas -gotest gotest.txt
   ```

6. Escena 2: navegar a `about:blank#tarjetas=<URL file:/// de .playwright-mcp/tarjetas>` y ejecutar
   `.playwright-mcp/escena-tarjetas.run.js`.
7. Montaje: `scripts/grabar-demo/montar.sh <app.webm> <tarjetas.webm> docs/img`.

El sonido y el parpadeo de la ventana al terminar son nativos y no se ven en el navegador: el video los menciona en un rótulo.
