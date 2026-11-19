import {describe, expect, it} from 'vitest'
import {domain, extract, jobs} from '../../wailsjs/go/models'
import {ACTIVITY_MAX, etaMs, initialJob, JobAction, jobReducer, JobState, percent, restoreActions} from './job'
import {FormState} from './request'

const form: FormState = {
  folder: {storeId: 's', entryId: 'e', path: 'p'}, range: {from: '2026-08-01', to: '2026-10-31'},
  destDir: 'C:\\d', organize: 'by_month', subjectContains: '', senderContains: '', extensions: [],
  skipInline: true, includeSubfolders: false,
}

const prog = (p: Partial<extract.Progress>) => extract.Progress.createFrom({
  phase: 'scanning', scanned: 0, inRange: 0, mailsDone: 0, mailsTotal: 0, saved: 0, skipped: 0,
  errors: 0, bytes: 0, elapsedMs: 0, ...p,
})
const act = (name: string) => extract.Activity.createFrom({at: '2026-11-12T10:42:11Z', kind: 'saved', name, subfolder: '', reason: '', size: 1})
const term = (jobId: string, extra: Partial<jobs.Terminal> = {}) => jobs.Terminal.createFrom({
  jobId, reportPath: 'r.xlsx', message: '', summary: domain.Summary.createFrom({cancelled: false}), ...extra,
})

const running = (id = 'b'): JobState => jobReducer(initialJob, {type: 'started', jobId: id, request: form})

describe('jobReducer', () => {
  it('started limpia el estado anterior', () => {
    const dirty: JobState = {...initialJob, activity: [act('x')], progress: prog({}), terminal: term('a'), phaseEndsMs: {connecting: 3}}
    const s = jobReducer(dirty, {type: 'started', jobId: 'b', request: form})
    expect(s.status).toBe('running')
    expect(s.jobId).toBe('b')
    expect(s.request).toBe(form)
    expect(s.activity).toEqual([])
    expect(s.progress).toBeNull()
    expect(s.terminal).toBeNull()
    expect(s.phaseEndsMs).toEqual({})
  })

  it('registra el fin de cada fase', () => {
    let s = jobReducer(running(), {type: 'progress', progress: prog({phase: 'scanning', elapsedMs: 400})})
    expect(s.phaseEndsMs.connecting).toBe(400)
    s = jobReducer(s, {type: 'progress', progress: prog({phase: 'downloading', elapsedMs: 6500})})
    expect(s.phaseEndsMs.scanning).toBe(6500)
    expect(s.downloadStartMs).toBe(6500)
  })

  it('recorta la actividad conservando las más nuevas', () => {
    const items = Array.from({length: ACTIVITY_MAX + 5}, (_, i) => act(`f${i}`))
    const s = jobReducer(running(), {type: 'activity', items})
    expect(s.activity).toHaveLength(ACTIVITY_MAX)
    expect(s.activity[ACTIVITY_MAX - 1].name).toBe(`f${ACTIVITY_MAX + 4}`)
    expect(s.activity[0].name).toBe('f5')
  })

  it('ignores terminal of another job', () => {
    const s = running('b')
    expect(jobReducer(s, {type: 'terminal', outcome: 'done', payload: term('a')})).toBe(s)
  })

  it('aplica el terminal del job vigente', () => {
    const t = term('b')
    const s = jobReducer(running('b'), {type: 'terminal', outcome: 'cancelled', payload: t})
    expect(s.status).toBe('cancelled')
    expect(s.terminal).toBe(t)
  })

  it('cancelRequested solo actúa sobre un job en curso', () => {
    expect(jobReducer(running(), {type: 'cancelRequested'}).status).toBe('cancelling')
    expect(jobReducer(initialJob, {type: 'cancelRequested'})).toBe(initialJob)
  })

  it('restored recupera un job en curso', () => {
    const st = jobs.Status.createFrom({jobId: 'z', running: true, destDir: 'C:\\d', progress: prog({scanned: 9}), recent: [act('r')]})
    const s = jobReducer(initialJob, {type: 'restored', status: st, request: form})
    expect(s.status).toBe('running')
    expect(s.jobId).toBe('z')
    expect(s.progress?.scanned).toBe(9)
    expect(s.activity.map((a) => a.name)).toEqual(['r'])
    expect(s.request).toBe(form)
  })

  it('restored recupera el resultado del último job', () => {
    const base = {jobId: 'z', running: false, destDir: 'C:\\d', progress: prog({}), recent: null}
    const done = jobReducer(initialJob, {type: 'restored', status: jobs.Status.createFrom({...base, last: term('z')}), request: null})
    expect(done.status).toBe('done')
    expect(done.terminal?.jobId).toBe('z')
    const cancelled = jobs.Status.createFrom({...base, last: term('z', {summary: domain.Summary.createFrom({cancelled: true})})})
    expect(jobReducer(initialJob, {type: 'restored', status: cancelled, request: null}).status).toBe('cancelled')
    const failed = jobs.Status.createFrom({...base, last: term('z', {message: 'falló'})})
    expect(jobReducer(initialJob, {type: 'restored', status: failed, request: null}).status).toBe('error')
  })

  it('restored usa el outcome explícito del terminal', () => {
    const base = {jobId: 'z', running: false, destDir: 'C:\\d', progress: prog({}), recent: null}
    // Cancelado durante el reporte: summary.cancelled queda en false pero el evento fue job:cancelled.
    const st = jobs.Status.createFrom({...base, last: term('z', {outcome: 'cancelled'})})
    expect(jobReducer(initialJob, {type: 'restored', status: st, request: null}).status).toBe('cancelled')
  })

  it('restoreActions aplica el terminal que llegó antes que GetJobStatus', () => {
    const st = jobs.Status.createFrom({jobId: 'z', running: true, destDir: 'C:\\d', progress: prog({phase: 'reporting'}), recent: []})
    const early = new Map<string, JobAction>([['z', {type: 'terminal', outcome: 'done', payload: term('z')}]])
    const s = restoreActions(st, form, early).reduce(jobReducer, initialJob)
    expect(s.status).toBe('done')
    expect(s.terminal?.jobId).toBe('z')
    expect(restoreActions(st, form, new Map())).toHaveLength(1)
  })

  it('restored sin job vuelve al estado inicial', () => {
    const st = jobs.Status.createFrom({jobId: '', running: false})
    expect(jobReducer(initialJob, {type: 'restored', status: st, request: null})).toEqual(initialJob)
  })

  it('reset vuelve al estado inicial', () => {
    expect(jobReducer(running(), {type: 'reset'})).toEqual(initialJob)
  })
})

describe('percent y etaMs', () => {
  it('percent solo existe al descargar', () => {
    expect(percent({...running(), progress: prog({phase: 'scanning'})})).toBeNull()
    expect(percent({...running(), progress: prog({phase: 'downloading', mailsDone: 156, mailsTotal: 312})})).toBe(0.5)
    expect(percent({...running(), progress: prog({phase: 'downloading', mailsTotal: 0})})).toBeNull()
  })

  it('etaMs extrapola el ritmo de descarga', () => {
    const s: JobState = {...running(), downloadStartMs: 6000,
      progress: prog({phase: 'downloading', elapsedMs: 16000, mailsDone: 100, mailsTotal: 300})}
    expect(etaMs(s)).toBe(20000)
    expect(etaMs({...s, progress: prog({phase: 'downloading', mailsDone: 0, mailsTotal: 300})})).toBeNull()
    expect(etaMs({...s, downloadStartMs: null})).toBeNull()
  })
})
