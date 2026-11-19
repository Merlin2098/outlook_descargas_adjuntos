// Iconos SVG del mock (docs/diseno/mock-ui). Usan currentColor salvo que se indique otro trazo.
import {ReactElement} from 'react'

export type IconName =
  | 'download' | 'minimize' | 'maximize' | 'close' | 'refresh' | 'chevronDown' | 'chevronRight'
  | 'mail' | 'database' | 'check' | 'alert' | 'folder' | 'file' | 'stop' | 'x' | 'lock'

const paths: Record<IconName, {viewBox: string; body: ReactElement; fill?: boolean}> = {
  download: {viewBox: '0 0 24 24', body: <><path d="M12 4v11"/><path d="m7 10 5 5 5-5"/><path d="M5 20h14"/></>},
  minimize: {viewBox: '0 0 12 12', body: <path d="M1 6h10"/>},
  maximize: {viewBox: '0 0 12 12', body: <rect x="1.5" y="1.5" width="9" height="9"/>},
  close: {viewBox: '0 0 12 12', body: <path d="M1.5 1.5l9 9M10.5 1.5l-9 9"/>},
  refresh: {viewBox: '0 0 24 24', body: <><path d="M20 11a8 8 0 1 0-2.3 5.7"/><path d="M20 4v7h-7"/></>},
  chevronDown: {viewBox: '0 0 24 24', body: <path d="m6 9 6 6 6-6"/>},
  chevronRight: {viewBox: '0 0 24 24', body: <path d="m9 6 6 6-6 6"/>},
  mail: {viewBox: '0 0 24 24', body: <><rect x="3" y="5" width="18" height="14" rx="2"/><path d="m3 7 9 6 9-6"/></>},
  database: {viewBox: '0 0 24 24', body: <><ellipse cx="12" cy="6" rx="8" ry="3"/><path d="M4 6v12c0 1.7 3.6 3 8 3s8-1.3 8-3V6"/></>},
  check: {viewBox: '0 0 24 24', body: <path d="m5 12 5 5 9-10"/>},
  alert: {viewBox: '0 0 24 24', body: <><path d="M12 8v5"/><path d="M12 16.5v.01"/><circle cx="12" cy="12" r="9"/></>},
  folder: {viewBox: '0 0 24 24', body: <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>},
  file: {viewBox: '0 0 24 24', body: <><path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5"/></>},
  stop: {viewBox: '0 0 24 24', body: <rect x="5" y="5" width="14" height="14" rx="2"/>, fill: true},
  lock: {viewBox: '0 0 24 24', body: <><rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V7a4 4 0 0 1 8 0v4"/></>},
  x: {viewBox: '0 0 24 24', body: <path d="M6 6l12 12M18 6 6 18"/>},
}

interface Props {
  name: IconName
  size?: number
  strokeWidth?: number
  className?: string
}

export function Icon({name, size = 14, strokeWidth = 2, className}: Props) {
  const p = paths[name]
  return (
    <svg width={size} height={size} viewBox={p.viewBox} fill={p.fill ? 'currentColor' : 'none'} stroke={p.fill ? 'none' : 'currentColor'}
         strokeWidth={strokeWidth} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" className={className}>
      {p.body}
    </svg>
  )
}
