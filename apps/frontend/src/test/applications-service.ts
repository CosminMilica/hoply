import { http, HttpResponse } from 'msw'

type Status = 'saved' | 'cv_ready' | 'applied'
export interface ApplicationRow {
  job_id: string
  status: Status
  saved_at: string | null
  cv_ready_at: string | null
  applied_at: string | null
  created_at: string
  updated_at: string
}
interface Answer { status: number, body?: unknown }

const STATUSES: Status[] = ['saved', 'cv_ready', 'applied']
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const invalid = (field: string, message: string): Answer => ({
  status: 422,
  body: { error: { code: 'VALIDATION_ERROR', message, type: 'ValidationError', context: { field } } },
})

/**
 * The applications service in memory, by docs/api/applications.md. Each change moves the clock one
 * second on from `start`, so the list order and the timestamps are predictable.
 */
export function createApplicationsBackend(start = '2026-11-25T09:00:00Z') {
  const applications = new Map<string, ApplicationRow>()
  const skips: string[] = []
  let tick = 0
  const now = () => new Date(Date.parse(start) + 1000 * (tick += 1)).toISOString()

  function put(jobId: string, status: string): Answer {
    if (!UUID.test(jobId)) return invalid('job_id', 'job_id must be a UUID')
    if (!STATUSES.includes(status as Status)) return invalid('status', 'status must be one of saved, cv_ready, applied')
    const at = now()
    const current = applications.get(jobId)
    const row: ApplicationRow = current
      ? { ...current }
      : { job_id: jobId, status: status as Status, saved_at: null, cv_ready_at: null, applied_at: null, created_at: at, updated_at: at }
    row.status = status === 'cv_ready' && current?.status === 'applied' ? 'applied' : status as Status
    row[`${status as Status}_at`] = at
    row.updated_at = at
    applications.set(jobId, row)
    return { status: 200, body: row }
  }

  return {
    applications,
    skips,
    /** Puts a job in a status, as the person would have before the test starts. */
    track(jobId: string, ...statuses: Status[]) {
      for (const status of statuses) put(jobId, status)
    },
    list(): Answer {
      return { status: 200, body: { items: [...applications.values()].sort((a, b) => b.updated_at.localeCompare(a.updated_at)) } }
    },
    put,
    remove(jobId: string): Answer {
      if (!UUID.test(jobId)) return invalid('job_id', 'job_id must be a UUID')
      return applications.delete(jobId) ? { status: 204 } : { status: 404, body: { error: { code: 'NOT_FOUND', message: 'Application not found' } } }
    },
    listSkips: (): Answer => ({ status: 200, body: { job_ids: [...skips] } }),
    skip(jobId: string): Answer {
      if (!UUID.test(jobId)) return invalid('job_id', 'job_id must be a UUID')
      if (!skips.includes(jobId)) skips.unshift(jobId)
      return { status: 204 }
    },
    unskip(jobId: string): Answer {
      if (!UUID.test(jobId)) return invalid('job_id', 'job_id must be a UUID')
      if (skips.includes(jobId)) skips.splice(skips.indexOf(jobId), 1)
      return { status: 204 }
    },
    clearSkips(): Answer {
      skips.splice(0)
      return { status: 204 }
    },
  }
}

export type ApplicationsBackend = ReturnType<typeof createApplicationsBackend>

const reply = ({ status, body }: Answer) => (body === undefined ? new HttpResponse(null, { status }) : HttpResponse.json(body as object, { status }))

/** The applications backend behind MSW, for unit and integration tests. */
export function applicationsServiceHandlers(backend: ApplicationsBackend = createApplicationsBackend()) {
  const BASE = '*/api/applications/v1'
  return {
    backend,
    handlers: [
      http.get(`${BASE}/applications`, () => reply(backend.list())),
      http.put(`${BASE}/applications/:jobId`, async ({ params, request }) => {
        const body = await request.json().catch(() => null) as { status?: string } | null
        return reply(backend.put(String(params.jobId), body?.status ?? ''))
      }),
      http.delete(`${BASE}/applications/:jobId`, ({ params }) => reply(backend.remove(String(params.jobId)))),
      http.get(`${BASE}/skips`, () => reply(backend.listSkips())),
      http.put(`${BASE}/skips/:jobId`, ({ params }) => reply(backend.skip(String(params.jobId)))),
      http.delete(`${BASE}/skips/:jobId`, ({ params }) => reply(backend.unskip(String(params.jobId)))),
      http.delete(`${BASE}/skips`, () => reply(backend.clearSkips())),
    ],
  }
}
