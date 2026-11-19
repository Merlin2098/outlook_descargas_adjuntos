import {useState} from 'react'
import {OpenDestination, OpenReport} from '../../wailsjs/go/main/App'
import {es} from '../i18n/es'
import {formatBytes, formatDuration, formatInt, formatPercent} from '../lib/format'
import {JobOutcome, JobState} from '../lib/job'
import {elapsedMs, extRows, ratePerSecond, reasonRows, subfolderRows, summaryTitle} from '../lib/summary'
import {Icon} from './Icon'

interface Props {
  job: JobState
  onNew: () => void
}

const baseName = (p: string) => p.split(/[\\/]/).filter(Boolean).pop() ?? p
const errText = (e: unknown) => (e instanceof Error ? e.message : String(e))
const barColors = ['var(--primary)', 'var(--info)', 'var(--accent)', 'var(--success)', 'var(--warning)']

const resultIcon: Record<JobOutcome, 'check' | 'stop' | 'alert'> = {done: 'check', cancelled: 'stop', error: 'alert'}

/** Pantalla "Resumen": resultado, cifras, desglose y acceso a los archivos. */
export function SummaryScreen({job, onNew}: Props) {
  const [openError, setOpenError] = useState<string | null>(null)
  const t = job.terminal
  const outcome: JobOutcome = job.status === 'cancelled' || job.status === 'error' ? job.status : 'done'
  if (!t) return null
  const s = t.summary
  const size = formatBytes(s.bytes).split(' ')
  const rate = ratePerSecond(s).toFixed(1).replace('.', ',')
  const exts = extRows(s.byExt)
  const reasons = reasonRows(s)
  const subs = subfolderRows(s.bySubfolder)
  const dest = job.request?.destDir ?? ''
  const flat = subs.length === 1 && subs[0].name === ''

  const open = async (fn: () => Promise<void>) => {
    setOpenError(null)
    try {
      await fn()
    } catch (e) {
      setOpenError(errText(e))
    }
  }

  return (
    <>
      <aside className="screen-aside aside-padded">
        <span className="section-title">{es.summary.files}</span>
        <div className="mono files-tree">
          <div className="files-root"><Icon name="folder" size={15} strokeWidth={1.8}/><span>{dest ? `${baseName(dest)}\\` : '—'}</span></div>
          {flat
            ? <div className="files-row"><span>{es.summary.filesInRoot}</span><span className="files-count">{formatInt(subs[0].count)}</span></div>
            : subs.map((r) => <div key={r.name} className="files-row"><span>{r.name}\</span><span className="files-count">{formatInt(r.count)}</span></div>)}
          {t.reportPath && <div className="files-row report"><Icon name="report" size={14} strokeWidth={1.8}/><span className="files-report-name">{baseName(t.reportPath)}</span></div>}
        </div>
        <div className="spacer"/>
        <div className="note-box">
          <span className="field-label">{es.summary.reportIncludes}</span>
          <span>{es.summary.sheetAttachments}<br/>{es.summary.sheetSkipped}</span>
        </div>
      </aside>

      <main className="screen-main">
        <div className="screen-content">
          <div className={`result-banner ${outcome}`}>
            <div className="result-icon"><Icon name={resultIcon[outcome]} size={24} strokeWidth={2.6}/></div>
            <div className="result-text">
              <h1>{summaryTitle(outcome)}</h1>
              <p>
                {es.summary.reviewed(formatInt(s.scanned))} <span className="mono">{formatDuration(elapsedMs(s))}</span>
                {s.errors > 0 && ` · ${es.summary.needReview(s.errors)}`}
              </p>
              {outcome === 'error' && t.message && <p className="result-message">{t.message}</p>}
            </div>
          </div>

          <div className="metrics">
            <div className="metric"><span>{es.summary.metrics.saved}</span><span className="mono metric-value primary">{formatInt(s.saved)}</span></div>
            <div className="metric"><span>{es.summary.metrics.size}</span><span className="mono metric-value">{size[0]}<span className="metric-unit"> {size[1]}</span></span></div>
            <div className="metric"><span>{es.summary.metrics.mails}</span><span className="mono metric-value">{formatInt(s.mailsWithDownloads)}</span></div>
            <div className="metric"><span>{es.summary.metrics.rate}</span><span className="mono metric-value">{rate}<span className="metric-unit"> {es.summary.metrics.rateUnit}</span></span></div>
          </div>

          <div className="summary-grid">
            <section className="card">
              <h2 className="section-title">{es.summary.byType}</h2>
              {exts.length === 0 && <p className="hint">{es.summary.noFiles}</p>}
              <div className="ext-list">
                {exts.map((r, i) => (
                  <div key={r.ext} className="ext-item">
                    <div className="mono ext-line"><span>{r.ext || es.summary.noExt}</span><span className="ext-count">{formatInt(r.count)} · {formatPercent(r.ratio)}</span></div>
                    <div className="ext-track"><div style={{width: `${r.ratio * 100}%`, background: barColors[i % barColors.length]}}/></div>
                  </div>
                ))}
              </div>
            </section>

            <section className="card reasons-card">
              <div className="card-head">
                <h2 className="section-title">{es.summary.reasons}</h2>
                <span className="hint">{es.summary.reasonsHint}</span>
              </div>
              {reasons.length === 0
                ? <p className="hint">{es.summary.noReasons}</p>
                : (
                  <div className="table-wrap">
                    <table className="reasons">
                      <thead><tr><th>{es.summary.table.reason}</th><th className="col-kind">{es.summary.table.kind}</th><th className="col-count">{es.summary.table.count}</th></tr></thead>
                      <tbody>
                        {reasons.map((r) => (
                          <tr key={`${r.kind}:${r.reason}`} className={r.kind}>
                            <td>{r.reason}</td>
                            <td className="kind">{r.kind === 'error' ? es.progress.error : es.progress.skipped}</td>
                            <td className="mono col-count">{formatInt(r.count)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
            </section>
          </div>
        </div>

        <div className="action-bar">
          <button className="btn btn-neutral" onClick={onNew}>{es.summary.newJob}</button>
          {openError && <span className="field-error" role="alert">{openError}</span>}
          <div className="spacer"/>
          <button className="btn btn-outline" disabled={!t.reportPath} onClick={() => open(OpenReport)}>
            <Icon name="report" size={15}/>{es.summary.openReport}
          </button>
          <button className="btn btn-primary" onClick={() => open(OpenDestination)}>
            <Icon name="folder" size={15} strokeWidth={2.2}/>{es.summary.openFolder}
          </button>
        </div>
      </main>
    </>
  )
}
