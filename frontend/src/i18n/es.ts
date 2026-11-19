// Textos de la UI. Los componentes no llevan literales en español: todo sale de aquí.
export const es = {
  appName: 'outlook_descargas_adjuntos',
  titleBar: {
    demo: 'MODO DEMO · datos simulados',
    minimize: 'Minimizar',
    maximize: 'Maximizar',
    close: 'Cerrar',
  },
  config: {
    errors: {
      noFolder: 'Debe seleccionar una carpeta de Outlook',
      noDest: 'Debe seleccionar una carpeta de destino',
      badRange: 'La fecha de inicio es posterior a la fecha fin',
    },
  },
  summary: {
    title: {
      done: 'Descarga completada',
      cancelled: 'Descarga cancelada',
      error: 'La descarga terminó con un error',
    },
  },
} as const
