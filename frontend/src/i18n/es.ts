// Textos de la UI. Los componentes no llevan literales en español: todo sale de aquí.
export const es = {
  appName: 'outlook_descargas_adjuntos',
  titleBar: {
    demo: 'MODO DEMO · datos simulados',
    minimize: 'Minimizar',
    maximize: 'Maximizar',
    close: 'Cerrar',
    confirmClose: {
      title: 'Hay una descarga en curso',
      message: 'Si sales ahora se cancela la descarga. Los archivos ya guardados se conservan y se escribe el reporte parcial.',
      confirm: 'Cancelar y salir',
      cancel: 'Seguir descargando',
    },
  },
  progress: {
    phaseShort: {
      connecting: 'conectando',
      scanning: 'escaneando',
      downloading: 'descargando',
      reporting: 'generando reporte',
    },
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
