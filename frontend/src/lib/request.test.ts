import {describe, expect, it} from 'vitest'
import {es} from '../i18n/es'
import {buildRequest, FormState, validateForm} from './request'

const full: FormState = {
  folder: {storeId: 'demo@example.com', entryId: 'demo@example.com\Facturas', path: 'demo@example.com\Facturas'},
  range: {from: '2026-08-01', to: '2026-10-31'},
  destDir: 'C:\destino',
  organize: 'by_month',
  subjectContains: 'factura',
  senderContains: '@proveedor',
  extensions: ['pdf', 'xml'],
  skipInline: true,
  includeSubfolders: false,
}

describe('validateForm', () => {
  it('pide carpeta, destino y rango válido', () => {
    expect(validateForm({...full, folder: null})).toBe(es.config.errors.noFolder)
    expect(validateForm({...full, destDir: '  '})).toBe(es.config.errors.noDest)
    expect(validateForm({...full, range: {from: '2026-11-02', to: '2026-11-01'}})).toBe(es.config.errors.badRange)
    expect(validateForm({...full, range: {from: '', to: '2026-11-01'}})).toBe(es.config.errors.badRange)
    expect(validateForm(full)).toBeNull()
  })
})

describe('buildRequest', () => {
  it('copia todos los campos y las fechas tal cual', () => {
    const r = buildRequest(full)
    expect(r.folder).toEqual(full.folder)
    expect(r.from).toBe('2026-08-01')
    expect(r.to).toBe('2026-10-31')
    expect(r.subjectContains).toBe('factura')
    expect(r.senderContains).toBe('@proveedor')
    expect(r.extensions).toEqual(['pdf', 'xml'])
    expect(r.skipInline).toBe(true)
    expect(r.includeSubfolders).toBe(false)
    expect(r.destDir).toBe('C:\destino')
    expect(r.organize).toBe('by_month')
  })
})
