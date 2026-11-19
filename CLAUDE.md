# CLAUDE.md

See @AGENTS.md

Contexto de diagnóstico del sistema legacy: @docs/diagnostico-legacy.md
Objetivo de portafolio y video demo: @docs/demo-y-portafolio.md

## Notas específicas para Claude Code

- Responde y documenta en español. El código Go usa identificadores en inglés.
- Para iterar en la UI usa `wails dev -appargs "--demo"`: no requiere Outlook y no expone datos reales.
- Para revisar el legacy, lee los archivos de `Outlook_AttachExtractor_Rework/` por tramos
  (con offset/limit). Los archivos de `core/` y `ui/` superan las 250 líneas.
- Antes de implementar un módulo nuevo, usa el flujo de brainstorming → plan → TDD.
  Para COM/Outlook, valida primero con un spike pequeño (`cmd/spike-*`) antes de integrarlo.
- Si tienes dudas sobre la API de go-ole, Wails v2, excelize o parquet-go, consulta la documentación
  del módulo (`go doc <pkg>.<Symbol>`) en vez de escribir de memoria.
- No ejecutes `wails build` ni `wails dev` en cada cambio: `go build ./... && go test ./...` basta
  para iterar. Usa `wails dev` solo para verificar la UI o los bindings.
- Las pruebas manuales contra Outlook real las hace el usuario. Si un cambio las requiere,
  dilo explícitamente y describe los pasos para verificarlo.
- Este directorio no es un repo git (el legacy tiene su propio `.git`). No inicialices git sin pedirlo.
