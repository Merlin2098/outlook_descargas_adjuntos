// Fechas como "YYYY-MM-DD" en hora local: es lo que espera el backend (time.Local).

export interface DateRange {
  from: string
  to: string
}

const pad = (n: number) => n.toString().padStart(2, '0')

/** Fecha local "YYYY-MM-DD" (nunca toISOString, que convierte a UTC). */
export function toIsoDate(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function parse(iso: string): Date | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso)
  return m ? new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3])) : null
}

/** Presets de la pantalla de configuración. */
export function presets(today: Date): {last30: DateRange; lastQuarter: DateRange; thisYear: DateRange} {
  const y = today.getFullYear()
  const m = today.getMonth()
  const to = toIsoDate(today)
  return {
    last30: {from: toIsoDate(new Date(y, m, today.getDate() - 30)), to},
    // Los 3 meses calendario completos anteriores al actual.
    lastQuarter: {from: toIsoDate(new Date(y, m - 3, 1)), to: toIsoDate(new Date(y, m, 0))},
    thisYear: {from: toIsoDate(new Date(y, 0, 1)), to},
  }
}

/** Días del rango contando ambos extremos; 0 si es inválido o está invertido. */
export function daysInclusive(r: DateRange): number {
  const a = parse(r.from)
  const b = parse(r.to)
  if (!a || !b || a > b) return 0
  // Date.UTC evita que un cambio de horario sume o reste una hora.
  const utc = (d: Date) => Date.UTC(d.getFullYear(), d.getMonth(), d.getDate())
  return Math.round((utc(b) - utc(a)) / 86_400_000) + 1
}

/** Meses "YYYY-MM" que toca el rango, como mucho max. */
export function monthsInRange(r: DateRange, max: number): string[] {
  const a = parse(r.from)
  const b = parse(r.to)
  if (!a || !b || a > b) return []
  const out: string[] = []
  for (let d = new Date(a.getFullYear(), a.getMonth(), 1); d <= b && out.length < max; d.setMonth(d.getMonth() + 1)) {
    out.push(`${d.getFullYear()}-${pad(d.getMonth() + 1)}`)
  }
  return out
}
