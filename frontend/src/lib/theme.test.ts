import {describe, expect, it} from 'vitest'
import {normalizeTheme, toggleTheme} from './theme'

describe('theme', () => {
  it('normaliza cualquier valor desconocido a claro', () => {
    expect(normalizeTheme('dark')).toBe('dark')
    expect(normalizeTheme('light')).toBe('light')
    expect(normalizeTheme('x')).toBe('light')
    expect(normalizeTheme(undefined)).toBe('light')
  })

  it('alterna entre claro y oscuro', () => {
    expect(toggleTheme('light')).toBe('dark')
    expect(toggleTheme('dark')).toBe('light')
  })
})
