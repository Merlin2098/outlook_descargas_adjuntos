import {extract, jobs} from '../../wailsjs/go/models'
import {FormState} from './request'

export type JobPhase = 'connecting' | 'scanning' | 'downloading' | 'reporting'
export type JobOutcome = 'done' | 'cancelled' | 'error'

export const PHASES: readonly JobPhase[] = ['connecting', 'scanning', 'downloading', 'reporting']

/** Estado del job tal como lo ve la UI. Solo cambia con jobReducer. */
export interface JobState {
  jobId: string | null
  request: FormState | null
  status: 'idle' | 'running' | 'cancelling' | JobOutcome
  progress: extract.Progress | null
  /** elapsedMs del job en el momento en que terminó cada fase. */
  phaseEndsMs: Partial<Record<JobPhase, number>>
  downloadStartMs: number | null
  /** Últimas ACTIVITY_MAX líneas, la más nueva al final. */
  activity: extract.Activity[]
  terminal: jobs.Terminal | null
}

export const ACTIVITY_MAX = 200

export type JobAction =
  | {type: 'started'; jobId: string; request: FormState}
  | {type: 'cancelRequested'}
  | {type: 'progress'; progress: extract.Progress}
  | {type: 'activity'; items: extract.Activity[]}
  | {type: 'terminal'; outcome: JobOutcome; payload: jobs.Terminal}
  | {type: 'restored'; status: jobs.Status; request: FormState | null}
  | {type: 'reset'}

export const initialJob: JobState = {
  jobId: null,
  request: null,
  status: 'idle',
  progress: null,
  phaseEndsMs: {},
  downloadStartMs: null,
  activity: [],
  terminal: null,
}

const isActive = (s: JobState) => s.status === 'running' || s.status === 'cancelling'

function applyProgress(s: JobState, p: extract.Progress): JobState {
  const idx = PHASES.indexOf(p.phase as JobPhase)
  const phaseEndsMs = {...s.phaseEndsMs}
  for (let i = 0; i < idx; i++) {
    if (phaseEndsMs[PHASES[i]] === undefined) phaseEndsMs[PHASES[i]] = p.elapsedMs
  }
  const downloadStartMs = s.downloadStartMs ?? (p.phase === 'downloading' ? p.elapsedMs : null)
  return {...s, progress: p, phaseEndsMs, downloadStartMs}
}

/** Resultado de un terminal recuperado con GetJobStatus. */
function outcomeOf(t: jobs.Terminal): JobOutcome {
  // El backend repite en Outcome el evento que emitió; lo demás es respaldo.
  if (t.outcome === 'done' || t.outcome === 'cancelled' || t.outcome === 'error') return t.outcome
  if (t.message) return 'error'
  if (t.summary?.cancelled) return 'cancelled'
  return 'done'
}

export function jobReducer(s: JobState, a: JobAction): JobState {
  switch (a.type) {
    case 'started':
      return {...initialJob, status: 'running', jobId: a.jobId, request: a.request}
    case 'cancelRequested':
      return s.status === 'running' ? {...s, status: 'cancelling'} : s
    case 'progress':
      return isActive(s) ? applyProgress(s, a.progress) : s
    case 'activity': {
      if (!isActive(s)) return s
      const all = s.activity.concat(a.items)
      return {...s, activity: all.length > ACTIVITY_MAX ? all.slice(all.length - ACTIVITY_MAX) : all}
    }
    case 'terminal':
      // Un terminal de otro job (por ejemplo, el cancelado antes de relanzar) se ignora.
      if (a.payload.jobId !== s.jobId) return s
      return {...s, status: a.outcome, terminal: a.payload}
    case 'restored': {
      const st = a.status
      if (!st.jobId) return initialJob
      const base: JobState = {...initialJob, jobId: st.jobId, request: a.request, activity: st.recent ?? []}
      if (st.running) return applyProgress({...base, status: 'running'}, st.progress)
      if (!st.last) return initialJob
      return {...base, status: outcomeOf(st.last), progress: st.progress, terminal: st.last}
    }
    case 'reset':
      return initialJob
  }
}

/**
 * Acciones para recuperar el estado con GetJobStatus. Si el terminal del job restaurado llegó antes que la
 * respuesta (se guardó en early porque aún no se conocía el id), se aplica después de restaurar.
 */
export function restoreActions(status: jobs.Status, request: FormState | null,
                               early: ReadonlyMap<string, JobAction>): JobAction[] {
  const out: JobAction[] = [{type: 'restored', status, request}]
  const pending = status.jobId ? early.get(status.jobId) : undefined
  if (pending) out.push(pending)
  return out
}

/** Avance 0..1 durante la descarga; null cuando no se puede calcular (barra indeterminada). */
export function percent(s: JobState): number | null {
  const p = s.progress
  if (!p || p.phase !== 'downloading' || p.mailsTotal <= 0) return null
  return p.mailsDone / p.mailsTotal
}

/** Tiempo restante estimado con el ritmo de la descarga; null sin datos suficientes. */
export function etaMs(s: JobState): number | null {
  const p = s.progress
  if (!p || p.phase !== 'downloading' || s.downloadStartMs === null || p.mailsDone <= 0) return null
  const perMail = (p.elapsedMs - s.downloadStartMs) / p.mailsDone
  return Math.max(0, Math.round(perMail * (p.mailsTotal - p.mailsDone)))
}
