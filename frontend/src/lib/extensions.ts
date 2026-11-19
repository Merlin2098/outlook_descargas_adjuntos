/** Agrega las extensiones escritas (separadas por coma, punto y coma o espacio) sin punto, en minúsculas y sin duplicados. */
export function addExtensions(current: string[], input: string): string[] {
  const out = [...current]
  for (const raw of input.split(/[\s,;]+/)) {
    const ext = raw.trim().replace(/^\.+/, '').toLowerCase()
    if (ext && !ext.includes('.') && !out.includes(ext)) out.push(ext)
  }
  return out
}
