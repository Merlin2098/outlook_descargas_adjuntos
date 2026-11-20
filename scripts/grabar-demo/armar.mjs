// Arma las escenas ejecutables por Playwright MCP (browser_run_code con `filename`): cada una es una
// sola función `async (page) => { comun.js + escena }`. Salida en .playwright-mcp/ (ignorada por git).
//   node scripts/grabar-demo/armar.mjs
import {readFileSync, writeFileSync, mkdirSync} from 'node:fs'
import {dirname, join} from 'node:path'
import {fileURLToPath} from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const out = join(here, '..', '..', '.playwright-mcp')
mkdirSync(out, {recursive: true})
const comun = readFileSync(join(here, 'comun.js'), 'utf8')
for (const escena of ['escena-app', 'escena-tarjetas']) {
  const body = readFileSync(join(here, `${escena}.js`), 'utf8')
  const target = join(out, `${escena}.run.js`)
  writeFileSync(target, `async (page) => {\n${comun}\n${body}\n}\n`)
  console.log(target)
}
