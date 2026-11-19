import {useEffect, useLayoutEffect, useRef} from 'react'
import {extract} from '../../wailsjs/go/models'
import {es} from '../i18n/es'
import {formatDateEs} from '../lib/dates'
import {formatBytes, formatClock, formatDuration, formatInt, formatPercent, formatSeconds} from '../lib/format'
import {etaMs, JobPhase, JobState, PHASES, percent} from '../lib/job'
import {FormState} from '../lib/request'
import {Icon} from './Icon'

interface Props {
  job: JobState
  onCancel: () => Promise<void>
}

/** Parámetros del job en el aside, de solo lectura. */
function JobParams({request}: {request: FormState | null}) {
  if (!request) return <p className="hint">{es.progress.params.unknown}</p>
  const filters: string[] = []
  if (request.subjectContains) filters.push(es.progress.params.subject(request.subjectContains))
  if (request.senderContains) filters.push(es.progress.params.sender(request.senderContains))
  const exts = request.extensions.length ? request.extensions.map((e) => `.${e}`).join(' ') : es.progress.params.allExt
  filters.push([exts, request.skipInline ? es.progress.params.noInline : es.progress.params.withInline].join(' · '))
  if (request.includeSubfolders) filters.push(es.progress.params.subfolders)
  const org = request.organize === 'by_month' ? es.config.dest.byMonth : es.config.dest.flat
  return (
    <dl className="params">
      <div><dt>{es.progress.params.origin}</dt><dd className="mono">{request.folder?.path ?? '—'}</dd></div>
      <div><dt>{es.progress.params.range}</dt><dd className="mono">{formatDateEs(request.range.from)} → {formatDateEs(request.range.to)}</dd></div>
      <div><dt>{es.progress.params.filters}</dt><dd className="mono">{filters.map((f) => <span key={f}>{f}</span>)}</dd></div>
      <div><dt>{es.progress.params.dest}</dt><dd className="mono">{request.destDir}\ · {org.toLowerCase()}</dd></div>
    </dl>
  )
}

function phaseDetail(job: JobState, phase: JobPhase, state: 'done' | 'current' | 'pending'): string {
  if (state === 'pending') return es.progress.pending
  if (state === 'current') return es.progress.inProgress
  const i = PHASES.indexOf(phase)
  const end = job.phaseEndsMs[phase] ?? 0
  const start = i > 0 ? job.phaseEndsMs[PHASES[i - 1]] ?? 0 : 0
  const secs = formatSeconds(Math.max(0, end - start))
  return phase === 'scanning' && job.progress ? `${es.progress.inRange(formatInt(job.progress.inRange))} · ${secs}` : secs
}

function Phases({job}: {job: JobState}) {
  const current = Math.max(0, PHASES.indexOf((job.progress?.phase ?? 'connecting') as JobPhase))
  return (
    <ol className="phases">
      {PHASES.map((p, i) => {
        const state = i < current ? 'done' : i === current ? 'current' : 'pending'
        return (
          <li key={p} className={`phase ${state}`}>
            <span className="phase-dot">
              {state === 'done' && <Icon name="check" size={12} strokeWidth={3}/>}
              {state === 'current' && <span className="phase-dot-inner"/>}
            </span>
            <span className="phase-text">
              <span className="phase-name">{es.progress.phases[p]}</span>
              <span className="mono phase-detail">{phaseDetail(job, p, state)}</span>
            </span>
          </li>
        )
      })}
    </ol>
  )
}

function ActivityRow({a, latest}: {a: extract.Activity; latest: boolean}) {
  const time = formatClock(String(a.at))
  const sub = a.subfolder ? `${a.subfolder}\\` : ''
  switch (a.kind) {
    case 'skipped':
      return <div className="act-row skipped"><span>{time}</span><span>–</span><span className="act-name">{a.name || '—'} · {a.reason}</span><span className="act-size">—</span><span>{es.progress.skipped}</span></div>
    case 'error':
      return <div className="act-row error"><span>{time}</span><span>!</span><span className="act-name">{a.name} · {a.reason}</span><span className="act-size">—</span><span>{es.progress.error}</span></div>
    default:
      return (
        <div className={`act-row saved${latest ? ' latest' : ''}`}>
          <span>{time}</span><span className="act-icon">↓</span>
          <span className="act-name">{a.name}{a.kind === 'renamed' && <span className="act-note"> {es.progress.renamed}</span>}</span>
          <span className="act-size">{formatBytes(a.size)}</span><span>{sub}</span>
        </div>
      )
  }
}

/** Pantalla "En curso": fases, contadores, barra de progreso y actividad en vivo. */
export function ProgressScreen({job, onCancel}: Props) {
  const p = job.progress
  const pct = percent(job)
  const eta = etaMs(job)
  const logRef = useRef<HTMLDivElement>(null)
  const stick = useRef(true)

  useLayoutEffect(() => {
    const el = logRef.current
    if (el && stick.current) el.scrollTop = el.scrollHeight
  }, [job.activity])

  useEffect(() => {
    stick.current = true
  }, [job.jobId])

  const cancelling = job.status === 'cancelling'
  const done = p?.phase === 'downloading' || p?.phase === 'reporting' ? p.mailsDone : 0
  const total = p?.phase === 'downloading' || p?.phase === 'reporting' ? p.mailsTotal : p?.inRange ?? 0
  const bytes = formatBytes(p?.bytes ?? 0).split(' ')

  return (
    <>
      <aside className="screen-aside aside-padded">
        <div className="aside-heading"><Icon name="lock" size={15}/><span className="section-title">{es.progress.params.title}</span></div>
        <JobParams request={job.request}/>
        <div className="spacer"/>
        <div className="info-note"><Icon name="moon" size={16}/><span>{es.progress.keepAwake}</span></div>
      </aside>

      <main className="screen-main">
        <div className="screen-content progress-content">
          <Phases job={job}/>

          <section className="card">
            <div className="counter-row">
              <span className="mono big-counter">{formatInt(done)}<span className="big-counter-total">/{formatInt(total)}</span></span>
              <span className="counter-label">{p?.phase === 'scanning' || !p ? es.progress.scannedLabel(formatInt(p?.scanned ?? 0)) : es.progress.mailsLabel}</span>
              <div className="spacer"/>
              <div className="time-box">
                <span className="mono elapsed">{formatDuration(p?.elapsedMs ?? 0)}</span>
                {eta !== null && <span className="hint">{es.progress.remaining} <span className="mono">{formatDuration(eta)}</span></span>}
              </div>
            </div>
            <div className={`progress-bar${pct === null ? ' indeterminate' : ''}`} role="progressbar" aria-label={es.progress.barLabel}
                 aria-valuemin={0} aria-valuemax={100} aria-valuenow={pct === null ? undefined : Math.round(pct * 100)}>
              <div className="progress-fill" style={pct === null ? undefined : {width: `${pct * 100}%`}}><div className="progress-sweep"/></div>
              {pct !== null && <span className="progress-pct">{formatPercent(pct)}</span>}
            </div>
            <div className="stats">
              <div className="stat"><span>{es.progress.stats.saved}</span><span className="mono stat-value">{formatInt(p?.saved ?? 0)}</span></div>
              <div className="stat"><span>{es.progress.stats.size}</span><span className="mono stat-value">{bytes[0]} <span className="stat-unit">{bytes[1]}</span></span></div>
              <div className="stat"><span>{es.progress.stats.skipped}</span><span className="mono stat-value warning">{formatInt(p?.skipped ?? 0)}</span></div>
              <div className="stat"><span>{es.progress.stats.errors}</span><span className="mono stat-value error">{formatInt(p?.errors ?? 0)}</span></div>
            </div>
          </section>

          <section className="activity">
            <div className="activity-head"><span className="live-dot"/><h2>{es.progress.activity}</h2></div>
            <div className="mono activity-log" ref={logRef} aria-live="off"
                 onScroll={(e) => {
                   const el = e.currentTarget
                   stick.current = el.scrollHeight - el.scrollTop - el.clientHeight < 24
                 }}>
              {job.activity.length === 0 && <span className="hint">{es.progress.waiting}</span>}
              {job.activity.map((a, i) => <ActivityRow key={job.activityBase + i} a={a} latest={i === job.activity.length - 1}/>)}
            </div>
          </section>
        </div>

        <div className="action-bar">
          <span className="page-subtitle">{es.progress.minimizeHint}</span>
          <div className="spacer"/>
          <button className="btn btn-danger" disabled={cancelling} onClick={onCancel}>
            <Icon name="stop" size={13}/>
            {cancelling ? es.progress.cancelling : es.progress.cancel}
          </button>
        </div>
      </main>
    </>
  )
}
