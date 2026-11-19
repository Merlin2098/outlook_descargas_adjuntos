import {describe, expect, it} from 'vitest'
import tokens from './tokens.css?raw'

/** Variables de un bloque CSS (`:root {…}` o `:root[data-theme="dark"] {…}`). */
function block(selector: string): Record<string, string> {
  const start = tokens.indexOf(selector + ' {')
  if (start < 0) throw new Error(`no existe ${selector}`)
  const body = tokens.slice(start, tokens.indexOf('}', start))
  const vars: Record<string, string> = {}
  for (const m of body.matchAll(/(--[\w-]+):\s*([^;]+);/g)) vars[m[1]] = m[2].trim()
  return vars
}

const light = block(':root')
const dark = {...light, ...block(':root[data-theme="dark"]')}

function luminance(hex: string): number {
  const v = hex.replace('#', '')
  const [r, g, b] = [0, 2, 4].map((i) => parseInt(v.slice(i, i + 2), 16) / 255)
    .map((c) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4))
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

function contrast(theme: Record<string, string>, fg: string, bg: string): number {
  const [a, b] = [luminance(theme[fg]), luminance(theme[bg])]
  return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05)
}

describe('contraste del tema oscuro', () => {
  it('el porcentaje de la barra se lee sobre el relleno y sobre la pista', () => {
    expect(contrast(dark, '--progress-text', '--progress-from')).toBeGreaterThanOrEqual(4.5)
    expect(contrast(dark, '--progress-text', '--progress-to')).toBeGreaterThanOrEqual(4.5)
    expect(contrast(dark, '--progress-text', '--surface-alt')).toBeGreaterThanOrEqual(4.5)
  })

  it('el conteo de la carpeta seleccionada se lee sobre el fondo primario', () => {
    expect(contrast(dark, '--on-primary-muted', '--primary-strong')).toBeGreaterThanOrEqual(4.5)
    expect(contrast(light, '--on-primary-muted', '--primary-strong')).toBeGreaterThanOrEqual(4.5)
  })
})
