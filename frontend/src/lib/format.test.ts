import {describe, expect, it} from 'vitest'
import {formatBytes, formatClock, formatDuration, formatInt, formatPercent, formatSeconds} from './format'

describe('formatInt', () => {
  it('agrupa miles con punto, también con 4 dígitos', () => {
    expect(formatInt(1248)).toBe('1.248')
    expect(formatInt(412)).toBe('412')
    expect(formatInt(1234567)).toBe('1.234.567')
    expect(formatInt(0)).toBe('0')
  })
})

describe('formatBytes', () => {
  it('usa KB sin decimales y MB/GB con un decimal y coma', () => {
    expect(formatBytes(0)).toBe('0 KB')
    expect(formatBytes(1536)).toBe('2 KB')
    expect(formatBytes(284 * 1024)).toBe('284 KB')
    expect(formatBytes(190_000_000)).toBe('181,2 MB')
    expect(formatBytes(1.4 * 1024 ** 3)).toBe('1,4 GB')
  })
})

describe('formatClock', () => {
  it('muestra la hora local', () => {
    expect(formatClock(new Date(2026, 10, 12, 10, 42, 11).toISOString())).toBe('10:42:11')
  })
})

describe('formatDuration', () => {
  it('usa mm:ss y h:mm:ss desde una hora', () => {
    expect(formatDuration(96_000)).toBe('01:36')
    expect(formatDuration(0)).toBe('00:00')
    expect(formatDuration(3_723_000)).toBe('1:02:03')
  })
})

describe('formatSeconds', () => {
  it('usa un decimal con coma', () => {
    expect(formatSeconds(400)).toBe('0,4 s')
    expect(formatSeconds(6100)).toBe('6,1 s')
  })
})

describe('formatPercent', () => {
  it('redondea a entero con espacio antes de %', () => {
    expect(formatPercent(0.634)).toBe('63 %')
    expect(formatPercent(1)).toBe('100 %')
  })
})
