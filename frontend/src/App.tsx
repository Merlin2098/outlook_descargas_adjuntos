// Pantalla de humo del Plan 1: verifica bindings y eventos de punta a punta.
// Sin estilos a propósito; se reemplaza por la UI del mock en el Plan 2.
import { useEffect, useState } from 'react'
import {
  CancelJob,
  ListChildren,
  ListRoots,
  PickDirectory,
  SourceInfo,
  StartJob,
} from '../wailsjs/go/main/App'
import { domain, main } from '../wailsjs/go/models'
import { EventsOn } from '../wailsjs/runtime/runtime'

const isoDate = (d: Date) => d.toISOString().slice(0, 10)

function App() {
  const [info, setInfo] = useState<main.SourceInfo>()
  const [roots, setRoots] = useState<domain.Folder[]>([])
  const [children, setChildren] = useState<domain.Folder[]>([])
  const [jobId, setJobId] = useState('')
  const [progress, setProgress] = useState<unknown>()
  const [activityCount, setActivityCount] = useState(0)
  const [terminal, setTerminal] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    SourceInfo().then(setInfo)
    ListRoots().then(setRoots).catch((e) => setError(String(e)))
    const offs = [
      EventsOn('job:progress', setProgress),
      EventsOn('job:activity', (acts: unknown[]) => setActivityCount((n) => n + acts.length)),
      EventsOn('job:done', (p: unknown) => { setTerminal('job:done ' + JSON.stringify(p)); setJobId('') }),
      EventsOn('job:cancelled', (p: unknown) => { setTerminal('job:cancelled ' + JSON.stringify(p)); setJobId('') }),
      EventsOn('job:error', (p: unknown) => { setTerminal('job:error ' + JSON.stringify(p)); setJobId('') }),
    ]
    return () => offs.forEach((off) => off())
  }, [])

  const openRoot = (f: domain.Folder) => ListChildren(f.ref).then(setChildren).catch((e) => setError(String(e)))

  const startDemo = async () => {
    setError('')
    setTerminal('')
    setActivityCount(0)
    const target = children.find((c) => c.name === 'Facturas proveedores')
    if (!target) {
      setError('Abre primero la raíz demo@example.com')
      return
    }
    const dest = await PickDirectory()
    if (!dest) return
    const to = new Date()
    const from = new Date(to.getTime() - 120 * 24 * 3600 * 1000)
    const req = main.JobRequestDTO.createFrom({
      folder: target.ref,
      from: isoDate(from),
      to: isoDate(to),
      extensions: [],
      skipInline: true,
      includeSubfolders: false,
      destDir: dest,
      organize: 'by_month',
    })
    StartJob(req).then(setJobId).catch((e) => setError(String(e)))
  }

  return (
    <div style={{ padding: 16, fontFamily: 'sans-serif' }}>
      <h1>Pantalla de humo — Plan 1</h1>
      <p>Fuente: {info?.name} {info?.demo ? '(modo demo)' : ''}</p>
      {error && <p style={{ color: '#b91c1c' }}>{error}</p>}
      <h2>Raíces</h2>
      <ul>
        {roots.map((r) => (
          <li key={r.ref.entryId}>
            <button onClick={() => openRoot(r)}>{r.name}</button>
          </li>
        ))}
      </ul>
      <h2>Hijos</h2>
      <ul>
        {children.map((c) => (
          <li key={c.ref.entryId}>{c.name} ({c.itemCount})</li>
        ))}
      </ul>
      <button onClick={startDemo} disabled={!!jobId}>Probar descarga demo</button>{' '}
      <button onClick={() => CancelJob(jobId)} disabled={!jobId}>Cancelar</button>
      <h2>Último job:progress</h2>
      <pre>{JSON.stringify(progress, null, 2)}</pre>
      <p>Actividades recibidas: {activityCount}</p>
      <h2>Evento terminal</h2>
      <pre style={{ whiteSpace: 'pre-wrap' }}>{terminal}</pre>
    </div>
  )
}

export default App
