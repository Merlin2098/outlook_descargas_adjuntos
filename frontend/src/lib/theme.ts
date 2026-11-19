/** Tema de la UI. Se guarda en config.json y se aplica con el atributo data-theme de <html>. */
export type Theme = 'light' | 'dark'

/** Cualquier valor que no sea 'dark' se trata como claro (el tema por defecto). */
export function normalizeTheme(v: unknown): Theme {
  return v === 'dark' ? 'dark' : 'light'
}

export function toggleTheme(t: Theme): Theme {
  return t === 'dark' ? 'light' : 'dark'
}

export function applyTheme(t: Theme): void {
  document.documentElement.dataset.theme = t
}
