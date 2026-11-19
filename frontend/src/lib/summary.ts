import {domain} from '../../wailsjs/go/models'
import {es} from '../i18n/es'
import {JobOutcome} from './job'

export interface ExtRow {
  ext: string
  count: number
  ratio: number
}

export interface ReasonRow {
  reason: string
  kind: 'skipped' | 'error'
  count: number
}

type Counts = Record<string, number> | null | undefined

/** Archivos por extensión, de más a menos, con su proporción sobre el total. */
export function extRows(byExt: Counts): ExtRow[] {
  const entries = Object.entries(byExt ?? {})
  const total = entries.reduce((n, [, c]) => n + c, 0)
  return entries
    .map(([ext, count]) => ({ext, count, ratio: total > 0 ? count / total : 0}))
    .sort((a, b) => b.count - a.count || a.ext.localeCompare(b.ext))
}

/** Omitidos y errores con su motivo, de más a menos. */
export function reasonRows(s: domain.Summary): ReasonRow[] {
  const rows: ReasonRow[] = [
    ...Object.entries(s.reasons ?? {}).map(([reason, count]) => ({reason, kind: 'skipped' as const, count})),
    ...Object.entries(s.errorReasons ?? {}).map(([reason, count]) => ({reason, kind: 'error' as const, count})),
  ]
  return rows.sort((a, b) => b.count - a.count)
}

/** Archivos guardados por subcarpeta, por nombre. */
export function subfolderRows(bySub: Counts): {name: string; count: number}[] {
  return Object.entries(bySub ?? {})
    .map(([name, count]) => ({name, count}))
    .sort((a, b) => a.name.localeCompare(b.name))
}

/** Duración del job en ms según sus marcas de inicio y fin (0 si faltan). */
export function elapsedMs(s: domain.Summary): number {
  const ms = Date.parse(String(s.finished)) - Date.parse(String(s.started))
  return Number.isFinite(ms) && ms > 0 ? ms : 0
}

/** Adjuntos guardados por segundo. */
export function ratePerSecond(s: domain.Summary): number {
  const ms = elapsedMs(s)
  return ms > 0 ? s.saved / (ms / 1000) : 0
}

export function summaryTitle(outcome: JobOutcome): string {
  return es.summary.title[outcome]
}
