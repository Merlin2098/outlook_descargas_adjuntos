import {KeyboardEvent, useState} from 'react'
import {PickDirectory, StartJob} from '../../wailsjs/go/main/App'
import {domain} from '../../wailsjs/go/models'
import {es} from '../i18n/es'
import {DateRange, daysInclusive, monthsInRange, presets} from '../lib/dates'
import {addExtensions} from '../lib/extensions'
import {formatInt} from '../lib/format'
import {buildRequest, FormState, validateForm} from '../lib/request'
import {FolderTree} from './FolderTree'
import {Icon} from './Icon'

interface Props {
  form: FormState
  onChange: (f: FormState) => void
  onStarted: (jobId: string) => void
}

const errText = (e: unknown) => (e instanceof Error ? e.message : String(e))
const sameRange = (a: DateRange, b: DateRange) => a.from === b.from && a.to === b.to

/** Pantalla "Configurar": carpeta de origen, rango, destino y filtros. */
export function ConfigScreen({form, onChange, onStarted}: Props) {
  const [selected, setSelected] = useState<domain.Folder | null>(null)
  const [extDraft, setExtDraft] = useState('')
  const [starting, setStarting] = useState(false)
  const [startError, setStartError] = useState<string | null>(null)
  const [pickError, setPickError] = useState<string | null>(null)

  const set = <K extends keyof FormState>(k: K, v: FormState[K]) => {
    setStartError(null)
    onChange({...form, [k]: v})
  }
  // Se recalculan en cada render: si la app queda abierta pasada la medianoche, siguen vigentes.
  const pre = presets(new Date())
  const days = daysInclusive(form.range)
  const months = monthsInRange(form.range, 4)
  const invalid = validateForm(form)

  const commitExt = () => {
    if (!extDraft.trim()) return
    set('extensions', addExtensions(form.extensions, extDraft))
    setExtDraft('')
  }
  const onExtKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',' || e.key === ';') {
      e.preventDefault()
      commitExt()
    } else if (e.key === 'Backspace' && !extDraft && form.extensions.length > 0) {
      set('extensions', form.extensions.slice(0, -1))
    }
  }

  const pick = async () => {
    setPickError(null)
    try {
      const dir = await PickDirectory()
      if (dir) set('destDir', dir)
    } catch (e) {
      setPickError(errText(e))
    }
  }

  const start = async () => {
    if (invalid || starting) return
    setStarting(true)
    setStartError(null)
    try {
      onStarted(await StartJob(buildRequest(form)))
    } catch (e) {
      setStartError(errText(e))
    } finally {
      setStarting(false)
    }
  }

  const presetButton = (label: string, r: DateRange) => {
    const active = sameRange(form.range, r)
    return <button className={`chip-btn${active ? ' active' : ''}`} aria-pressed={active} onClick={() => set('range', r)}>{label}</button>
  }

  const selectedCount = selected && form.folder && selected.ref.entryId === form.folder.entryId
    && selected.ref.storeId === form.folder.storeId ? selected.itemCount : null

  return (
    <>
      <aside className="screen-aside">
        <FolderTree selected={form.folder} onSelect={(f) => { setSelected(f); set('folder', f.ref) }}/>
        <div className="mono selected-path">
          {form.folder ? `${es.config.selectedPath} ${form.folder.path}` : es.config.noSelection}
        </div>
      </aside>

      <main className="screen-main">
        <div className="screen-content">
          <div className="page-head">
            <h1 className="page-title">{es.config.title}</h1>
            <p className="page-subtitle">{es.config.subtitle}</p>
          </div>

          <div className="config-grid">
            <section className="card">
              <div className="card-head">
                <h2 className="section-title">{es.config.range.title}</h2>
                <span className="mono hint">{es.config.range.days(days)}</span>
              </div>
              <div className="two-cols">
                <label className="field">{es.config.range.from}
                  <input type="date" className="input" value={form.range.from} max={form.range.to || undefined}
                         onChange={(e) => set('range', {...form.range, from: e.target.value})}/>
                </label>
                <label className="field">{es.config.range.to}
                  <input type="date" className="input" value={form.range.to} min={form.range.from || undefined}
                         onChange={(e) => set('range', {...form.range, to: e.target.value})}/>
                </label>
              </div>
              <div className="chip-row">
                {presetButton(es.config.range.last30, pre.last30)}
                {presetButton(es.config.range.lastQuarter, pre.lastQuarter)}
                {presetButton(es.config.range.thisYear, pre.thisYear)}
              </div>
            </section>

            <section className="card">
              <h2 className="section-title">{es.config.dest.title}</h2>
              <div className="dest-row">
                <label className="dest-field">
                  <span className="sr-only">{es.config.dest.title}</span>
                  <input className="input mono" value={form.destDir} placeholder={es.config.dest.placeholder}
                         spellCheck={false} onChange={(e) => set('destDir', e.target.value)}/>
                </label>
                <button className="btn btn-outline btn-sm" onClick={pick}>{es.config.dest.pick}</button>
              </div>
              <div className="organize-row" role="radiogroup" aria-label={es.config.dest.organize}>
                <span className="field-label">{es.config.dest.organize}</span>
                <label className="check"><input type="radio" name="organize" checked={form.organize === 'flat'}
                                                onChange={() => set('organize', 'flat')}/>{es.config.dest.flat}</label>
                <label className="check"><input type="radio" name="organize" checked={form.organize === 'by_month'}
                                                onChange={() => set('organize', 'by_month')}/>{es.config.dest.byMonth}</label>
                {form.organize === 'by_month' && months.length > 0 && (
                  <span className="mono hint">
                    {months.slice(0, 3).map((m) => `${m}\\`).join(' ')}{months.length > 3 ? ' …' : ''}
                  </span>
                )}
              </div>
              {pickError && <p className="field-error">{pickError}</p>}
            </section>

            <section className="card span-2">
              <div className="card-head start">
                <h2 className="section-title">{es.config.filters.title}</h2>
                <span className="hint">{es.config.filters.hint}</span>
              </div>
              <div className="filters-grid">
                <label className="field">{es.config.filters.subject}
                  <input className="input" value={form.subjectContains} placeholder={es.config.filters.subjectPlaceholder}
                         onChange={(e) => set('subjectContains', e.target.value)}/>
                </label>
                <label className="field">{es.config.filters.sender}
                  <input className="input" value={form.senderContains} placeholder={es.config.filters.senderPlaceholder}
                         onChange={(e) => set('senderContains', e.target.value)}/>
                </label>
                <div className="field">
                  <label htmlFor="ext-input">{es.config.filters.extensions}</label>
                  <div className="chips-input">
                    {form.extensions.map((ext) => (
                      <span key={ext} className="mono ext-chip">
                        .{ext}
                        <button aria-label={es.config.filters.removeExt(ext)}
                                onClick={() => set('extensions', form.extensions.filter((x) => x !== ext))}>
                          <Icon name="x" size={10} strokeWidth={2.4}/>
                        </button>
                      </span>
                    ))}
                    <input id="ext-input" className="chips-draft" value={extDraft} placeholder={es.config.filters.addExt}
                           onChange={(e) => setExtDraft(e.target.value)} onKeyDown={onExtKey} onBlur={commitExt}/>
                  </div>
                </div>
              </div>
              <div className="checks-row">
                <label className="check"><input type="checkbox" checked={form.skipInline}
                                                onChange={(e) => set('skipInline', e.target.checked)}/>{es.config.filters.skipInline}</label>
                <label className="check"><input type="checkbox" checked={form.includeSubfolders}
                                                onChange={(e) => set('includeSubfolders', e.target.checked)}/>{es.config.filters.includeSubfolders}</label>
              </div>
            </section>
          </div>
        </div>

        <div className="action-bar">
          <div className="action-info">
            {startError
              ? <span className="field-error" role="alert">{startError}</span>
              : selectedCount !== null
                ? <span><span className="mono strong-number">{formatInt(selectedCount)}</span> {es.config.footer(selectedCount, form.includeSubfolders)}</span>
                : <span className="hint">{invalid ?? form.folder?.path ?? ''}</span>}
            {!startError && selectedCount !== null && invalid && <span className="hint">{invalid}</span>}
            {!startError && !invalid && <span className="hint">{es.config.keepAwake}</span>}
          </div>
          <div className="spacer"/>
          <button className="btn btn-primary" disabled={invalid !== null || starting} onClick={start}
                  title={invalid ?? undefined}>
            <Icon name="download" size={15} strokeWidth={2.4}/>
            {starting ? es.config.starting : es.config.start}
          </button>
        </div>
      </main>
    </>
  )
}
