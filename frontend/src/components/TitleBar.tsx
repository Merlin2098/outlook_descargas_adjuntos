import {useEffect, useState} from 'react'
import {ConfirmClose, SourceInfo} from '../../wailsjs/go/main/App'
import {main} from '../../wailsjs/go/models'
import {EventsOn, Quit, WindowMinimise, WindowToggleMaximise} from '../../wailsjs/runtime/runtime'
import {es} from '../i18n/es'
import {ConfirmDialog} from './ConfirmDialog'
import {Icon} from './Icon'

interface Props {
  /** Texto de estado a la derecha (p. ej. "63 % · descargando"). */
  status?: string
  /** Hay un job en curso: cerrar pide confirmación. */
  busy: boolean
}

/** Barra de título propia de la ventana sin marco. Se arrastra desde cualquier punto que no sea un botón. */
export function TitleBar({status, busy}: Props) {
  const [info, setInfo] = useState<main.SourceInfo | null>(null)
  const [confirming, setConfirming] = useState(false)

  useEffect(() => {
    SourceInfo().then(setInfo).catch(() => setInfo(null))
  }, [])

  // Alt+F4 o cerrar desde la barra de tareas con un job activo: el backend lo impide y avisa aquí.
  useEffect(() => EventsOn('app:closeRequested', () => setConfirming(true)), [])

  const close = () => (busy ? setConfirming(true) : Quit())

  return (
    <header className="titlebar">
      <div className="titlebar-logo"><Icon name="download" size={12} strokeWidth={2.6}/></div>
      <span className="mono titlebar-name">{es.appName}</span>
      {info?.version && <span className="mono titlebar-version">v{info.version}</span>}
      {info?.demo && <span className="demo-badge">{es.titleBar.demo}</span>}
      <div className="spacer"/>
      {status && <span className="mono titlebar-status">{status}</span>}
      <div className="window-buttons">
        <button aria-label={es.titleBar.minimize} onClick={WindowMinimise}><Icon name="minimize" size={12} strokeWidth={1.2}/></button>
        <button aria-label={es.titleBar.maximize} onClick={WindowToggleMaximise}><Icon name="maximize" size={12} strokeWidth={1.2}/></button>
        <button aria-label={es.titleBar.close} className="window-close" onClick={close}><Icon name="close" size={12} strokeWidth={1.2}/></button>
      </div>
      {confirming && (
        <ConfirmDialog
          title={es.titleBar.confirmClose.title}
          message={es.titleBar.confirmClose.message}
          confirmLabel={es.titleBar.confirmClose.confirm}
          cancelLabel={es.titleBar.confirmClose.cancel}
          onCancel={() => setConfirming(false)}
          onConfirm={() => {
            setConfirming(false)
            // Cancela el job y cierra; el backend espera a que escriba el reporte parcial.
            ConfirmClose().catch(() => Quit())
          }}
        />
      )}
    </header>
  )
}
