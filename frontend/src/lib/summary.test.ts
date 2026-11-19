import {describe, expect, it} from 'vitest'
import {domain} from '../../wailsjs/go/models'
import {es} from '../i18n/es'
import {extRows, ratePerSecond, reasonRows, subfolderRows, summaryTitle} from './summary'

describe('extRows', () => {
  it('ordena por cantidad y calcula la proporción', () => {
    const rows = extRows({'.xlsx': 33, '.pdf': 402, '.xml': 206})
    expect(rows.map((r) => r.ext)).toEqual(['.pdf', '.xml', '.xlsx'])
    expect(rows[0].ratio).toBeCloseTo(0.627, 3)
    expect(extRows({})).toEqual([])
  })
})

describe('reasonRows', () => {
  it('mezcla omitidos y errores ordenados por cantidad', () => {
    const s = domain.Summary.createFrom({
      reasons: {'Solo imágenes inline': 4, 'Sin adjuntos con las extensiones elegidas': 19},
      errorReasons: {'Adjunto bloqueado por Outlook': 1},
    })
    expect(reasonRows(s)).toEqual([
      {reason: 'Sin adjuntos con las extensiones elegidas', kind: 'skipped', count: 19},
      {reason: 'Solo imágenes inline', kind: 'skipped', count: 4},
      {reason: 'Adjunto bloqueado por Outlook', kind: 'error', count: 1},
    ])
  })
  it('tolera mapas ausentes', () => {
    expect(reasonRows(domain.Summary.createFrom({}))).toEqual([])
  })
})

describe('subfolderRows', () => {
  it('ordena por nombre', () => {
    expect(subfolderRows({'2026-10': 212, '2026-08': 198})).toEqual([
      {name: '2026-08', count: 198}, {name: '2026-10', count: 212}])
  })
})

describe('ratePerSecond', () => {
  it('divide guardados por segundos transcurridos', () => {
    const s = domain.Summary.createFrom({saved: 641, started: '2026-11-12T10:40:00Z', finished: '2026-11-12T10:42:31Z'})
    expect(ratePerSecond(s)).toBeCloseTo(4.245, 3)
  })
  it('devuelve 0 sin tiempo transcurrido', () => {
    const s = domain.Summary.createFrom({saved: 5, started: '2026-11-12T10:40:00Z', finished: '2026-11-12T10:40:00Z'})
    expect(ratePerSecond(s)).toBe(0)
  })
})

describe('summaryTitle', () => {
  it('usa el texto de cada resultado', () => {
    expect(summaryTitle('done')).toBe(es.summary.title.done)
    expect(summaryTitle('cancelled')).toBe(es.summary.title.cancelled)
    expect(summaryTitle('error')).toBe(es.summary.title.error)
  })
})
