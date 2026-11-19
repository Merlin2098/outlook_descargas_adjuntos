import {describe, expect, it} from 'vitest'
import {daysInclusive, formatDateEs, monthsInRange, presets, toIsoDate} from './dates'

describe('presets', () => {
  it('calcula los tres presets desde hoy', () => {
    const p = presets(new Date(2026, 10, 12))
    expect(p.last30).toEqual({from: '2026-10-13', to: '2026-11-12'})
    expect(p.lastQuarter).toEqual({from: '2026-08-01', to: '2026-10-31'})
    expect(p.thisYear).toEqual({from: '2026-01-01', to: '2026-11-12'})
  })
  it('el último trimestre cruza el año', () => {
    expect(presets(new Date(2026, 1, 10)).lastQuarter).toEqual({from: '2025-11-01', to: '2026-01-31'})
  })
})

describe('daysInclusive', () => {
  it('cuenta ambos extremos', () => {
    expect(daysInclusive({from: '2026-08-01', to: '2026-10-31'})).toBe(92)
    expect(daysInclusive({from: '2026-11-01', to: '2026-11-01'})).toBe(1)
  })
  it('devuelve 0 con un rango invertido', () => {
    expect(daysInclusive({from: '2026-11-02', to: '2026-11-01'})).toBe(0)
  })
})

describe('monthsInRange', () => {
  it('lista los meses del rango hasta max', () => {
    expect(monthsInRange({from: '2026-08-15', to: '2026-10-01'}, 5)).toEqual(['2026-08', '2026-09', '2026-10'])
    expect(monthsInRange({from: '2026-01-01', to: '2026-12-31'}, 3)).toEqual(['2026-01', '2026-02', '2026-03'])
    expect(monthsInRange({from: '2026-12-01', to: '2026-11-01'}, 3)).toEqual([])
  })
})

describe('toIsoDate', () => {
  it('toIsoDate uses local date', () => {
    expect(toIsoDate(new Date(2026, 10, 30, 23, 30))).toBe('2026-11-30')
    expect(toIsoDate(new Date(2026, 0, 1, 0, 5))).toBe('2026-01-01')
  })
})

describe('formatDateEs', () => {
  it('muestra dd/mm/aaaa', () => {
    expect(formatDateEs('2026-08-01')).toBe('01/08/2026')
    expect(formatDateEs('')).toBe('')
  })
})
