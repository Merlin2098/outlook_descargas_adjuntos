import {useEffect, useState} from 'react'
import {GetConfig} from '../wailsjs/go/main/App'
import {ConfigScreen} from './components/ConfigScreen'
import {ProgressScreen} from './components/ProgressScreen'
import {SummaryScreen} from './components/SummaryScreen'
import {TitleBar} from './components/TitleBar'
import {useJob} from './hooks/useJob'
import {es} from './i18n/es'
import {presets} from './lib/dates'
import {formatPercent} from './lib/format'
import {percent} from './lib/job'
import {FormState} from './lib/request'

const initialForm = (): FormState => ({
  folder: null,
  range: presets(new Date()).last30,
  destDir: '',
  organize: 'by_month',
  subjectContains: '',
  senderContains: '',
  extensions: [],
  skipInline: true,
  includeSubfolders: false,
})

/** Elige la pantalla según el estado del job: configurar → en curso → resumen. */
function App() {
  const {job, start, cancel, reset} = useJob()
  const [form, setForm] = useState<FormState>(initialForm)

  useEffect(() => {
    GetConfig()
      .then((c) => setForm((f) => ({
        ...f,
        destDir: f.destDir || c.lastDestDir || '',
        organize: c.organize === 'flat' ? 'flat' : 'by_month',
        skipInline: c.skipInline,
      })))
      .catch(() => undefined)
  }, [])

  const busy = job.status === 'running' || job.status === 'cancelling'
  let status: string | undefined
  if (busy && job.progress) {
    const pct = percent(job)
    const phase = es.progress.phaseShort[job.progress.phase as keyof typeof es.progress.phaseShort] ?? ''
    status = pct === null ? phase : `${formatPercent(pct)} · ${phase}`
  }

  const restart = () => {
    if (job.request) setForm(job.request)
    reset()
  }

  return (
    <div className="app">
      <TitleBar status={status} busy={busy} onCancelJob={cancel}/>
      <div className="app-body">
        {/* Configurar queda montada (oculta) para conservar el árbol y el formulario al volver. */}
        <div className="screen" hidden={job.status !== 'idle'}>
          <ConfigScreen form={form} onChange={setForm} onStarted={(id) => start(id, form)}/>
        </div>
        {busy && <div className="screen"><ProgressScreen job={job} onCancel={cancel}/></div>}
        {!busy && job.status !== 'idle' && <div className="screen"><SummaryScreen job={job} onNew={restart}/></div>}
      </div>
    </div>
  )
}

export default App
