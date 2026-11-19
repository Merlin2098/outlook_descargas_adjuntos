import {useEffect, useRef} from 'react'

interface Props {
  title: string
  message: string
  confirmLabel: string
  cancelLabel: string
  onConfirm: () => void
  onCancel: () => void
}

/** Diálogo modal simple; Escape cancela. */
export function ConfirmDialog({title, message, confirmLabel, cancelLabel, onConfirm, onCancel}: Props) {
  const cancelRef = useRef<HTMLButtonElement>(null)
  // onCancel cambia en cada render del padre (llegan ~10 eventos de progreso por segundo):
  // se lee desde una ref para no repetir el efecto ni robar el foco.
  const onCancelRef = useRef(onCancel)
  onCancelRef.current = onCancel

  useEffect(() => {
    cancelRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancelRef.current()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  return (
    <div className="dialog-backdrop" role="presentation">
      <div className="dialog card" role="alertdialog" aria-modal="true" aria-labelledby="dialog-title">
        <h2 id="dialog-title" className="section-title">{title}</h2>
        <p>{message}</p>
        <div className="dialog-actions">
          <button ref={cancelRef} className="btn btn-outline" onClick={onCancel}>{cancelLabel}</button>
          <button className="btn btn-danger" onClick={onConfirm}>{confirmLabel}</button>
        </div>
      </div>
    </div>
  )
}
