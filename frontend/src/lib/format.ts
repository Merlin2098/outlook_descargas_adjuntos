// Formato es-ES hecho a mano: Intl es-ES no agrupa los números de 4 dígitos (1248 en vez de 1.248).

/** Entero con punto de miles: 1248 → "1.248". */
export function formatInt(n: number): string {
  const sign = n < 0 ? '-' : ''
  const digits = Math.round(Math.abs(n)).toString()
  return sign + digits.replace(/\B(?=(\d{3})+(?!\d))/g, '.')
}

function oneDecimal(n: number): string {
  return n.toFixed(1).replace('.', ',')
}

/** Tamaño legible: KB sin decimales; MB y GB con un decimal ("181,2 MB"). */
export function formatBytes(b: number): string {
  const kb = b / 1024
  if (kb < 1024) return `${formatInt(kb)} KB`
  const mb = kb / 1024
  if (mb < 1024) return `${oneDecimal(mb)} MB`
  return `${oneDecimal(mb / 1024)} GB`
}

const pad = (n: number) => n.toString().padStart(2, '0')

/** Hora local "HH:MM:SS" de un timestamp ISO. */
export function formatClock(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

/** Duración "mm:ss", o "h:mm:ss" desde una hora. */
export function formatDuration(ms: number): string {
  const total = Math.max(0, Math.round(ms / 1000))
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${pad(m)}:${pad(s)}`
}

/** Segundos con un decimal: "6,1 s". */
export function formatSeconds(ms: number): string {
  return `${oneDecimal(ms / 1000)} s`
}

/** Porcentaje entero: 0.634 → "63 %". */
export function formatPercent(ratio: number): string {
  return `${Math.round(ratio * 100)} %`
}
