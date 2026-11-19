import {useCallback, useEffect, useReducer, useRef} from 'react'
import {CancelJob, GetJobStatus} from '../../wailsjs/go/main/App'
import {extract, jobs} from '../../wailsjs/go/models'
import {EventsOn} from '../../wailsjs/runtime/runtime'
import {initialJob, JobAction, jobReducer, JobOutcome, JobState} from '../lib/job'
import {FormState} from '../lib/request'

// Nombres de los eventos de internal/jobs/manager.go.
const terminalEvents: Record<string, JobOutcome> = {
  'job:done': 'done',
  'job:cancelled': 'cancelled',
  'job:error': 'error',
}

const storageKey = 'jobRequest'

function saveRequest(r: FormState) {
  try {
    sessionStorage.setItem(storageKey, JSON.stringify(r))
  } catch {
    // Sin sessionStorage la recuperación tras recargar muestra el job sin sus parámetros.
  }
}

function loadRequest(): FormState | null {
  try {
    const raw = sessionStorage.getItem(storageKey)
    return raw ? (JSON.parse(raw) as FormState) : null
  } catch {
    return null
  }
}

/** Estado del job alimentado por los eventos job:* y, al montar, por GetJobStatus. */
export function useJob() {
  const [job, dispatch] = useReducer(jobReducer, initialJob)
  const jobIdRef = useRef<string | null>(null)
  // Terminales que llegan antes de que StartJob devuelva su id (un job muy corto).
  const early = useRef(new Map<string, JobAction>())

  useEffect(() => {
    jobIdRef.current = job.jobId
  }, [job.jobId])

  useEffect(() => {
    const offs = [
      EventsOn('job:progress', (p: unknown) => dispatch({type: 'progress', progress: extract.Progress.createFrom(p)})),
      EventsOn('job:activity', (items: unknown) => {
        const list = Array.isArray(items) ? items : []
        dispatch({type: 'activity', items: list.map((i: unknown) => extract.Activity.createFrom(i))})
      }),
      ...Object.entries(terminalEvents).map(([name, outcome]) =>
        EventsOn(name, (t: unknown) => {
          const payload = jobs.Terminal.createFrom(t)
          const action: JobAction = {type: 'terminal', outcome, payload}
          if (jobIdRef.current === null) {
            early.current.set(payload.jobId, action)
            if (early.current.size > 5) early.current.delete(early.current.keys().next().value as string)
          }
          dispatch(action)
        })),
    ]
    GetJobStatus()
      .then((status) => dispatch({type: 'restored', status, request: loadRequest()}))
      .catch(() => undefined)
    return () => offs.forEach((off) => off())
  }, [])

  const start = useCallback((jobId: string, request: FormState) => {
    saveRequest(request)
    jobIdRef.current = jobId
    dispatch({type: 'started', jobId, request})
    const pending = early.current.get(jobId)
    early.current.clear()
    if (pending) dispatch(pending)
  }, [])

  const cancel = useCallback(async () => {
    const id = jobIdRef.current
    if (!id) return
    dispatch({type: 'cancelRequested'})
    try {
      await CancelJob(id)
    } catch {
      // El job ya terminó: su evento terminal llega por su cuenta.
    }
  }, [])

  const reset = useCallback(() => {
    jobIdRef.current = null
    dispatch({type: 'reset'})
  }, [])

  const result: {job: JobState; start: typeof start; cancel: typeof cancel; reset: typeof reset} = {job, start, cancel, reset}
  return result
}
