import {describe, expect, it} from 'vitest'
import {addExtensions} from './extensions'

describe('addExtensions', () => {
  it('normaliza, separa por coma, punto y coma o espacio, y no duplica', () => {
    expect(addExtensions(['pdf'], '.XLSX, pdf; xml')).toEqual(['pdf', 'xlsx', 'xml'])
  })
  it('ignora entradas vacías o solo puntos', () => {
    expect(addExtensions(['pdf'], ' , . ;.. ')).toEqual(['pdf'])
    expect(addExtensions([], '')).toEqual([])
  })
})
