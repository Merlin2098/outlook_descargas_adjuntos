# build/

Recursos de compilación de Wails.

- `appicon.png`: ícono de la app (1024×1024), generado desde [docs/diseno/icono.svg](../docs/diseno/icono.svg).
  Wails crea `windows/icon.ico` a partir de él si no existe.
- `windows/info.json`: metadatos del `.exe` (producto, versión, copyright), que se completan con el bloque `info` de `wails.json`.
- `windows/wails.exe.manifest`: manifiesto de la aplicación.
- `bin/`: salida de `wails build` (ignorada por git).

La versión del binario se inyecta al compilar: `wails build -ldflags "-X main.appVersion=X.Y.Z"` (lo hace la release a partir del tag).
