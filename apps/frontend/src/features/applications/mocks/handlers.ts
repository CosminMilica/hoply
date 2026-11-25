import { http, HttpResponse } from 'msw'
import { APPLICATION_STATUSES, type Application, type ApplicationStatus } from '../api/applications.schema'

export interface ApplicationsStore {
  applications: Map<string, Application>
  skips: Set<string>
}

export function createApplicationsStore(): ApplicationsStore {
  return { applications: new Map(), skips: new Set() }
}

const notFound = () => HttpResponse.json({ message: 'Not found', type: 'NotFound', detail: null }, { status: 404 })

/** The future applications service: saved jobs, their status, and the jobs skipped in the deck. */
export function createApplicationsHandlers(store: ApplicationsStore = createApplicationsStore()) {
  return [
    http.get('*/api/applications/v1/applications', () =>
      HttpResponse.json({
        items: [...store.applications.values()].sort((a, b) => b.updated_at.localeCompare(a.updated_at)),
      })),
    http.put('*/api/applications/v1/applications/:jobId', async ({ params, request }) => {
      const body = (await request.json().catch(() => null)) as { status?: string } | null
      const status = body?.status as ApplicationStatus | undefined
      if (!status || !APPLICATION_STATUSES.includes(status)) {
        return HttpResponse.json(
          { message: `status must be one of ${APPLICATION_STATUSES.join(', ')}`, type: 'ValidationError', detail: null },
          { status: 422 },
        )
      }
      const application: Application = { job_id: String(params.jobId), status, updated_at: new Date().toISOString() }
      store.applications.set(application.job_id, application)
      return HttpResponse.json(application)
    }),
    http.delete('*/api/applications/v1/applications/:jobId', ({ params }) => {
      if (!store.applications.delete(String(params.jobId))) return notFound()
      return new HttpResponse(null, { status: 204 })
    }),
    http.get('*/api/applications/v1/skips', () => HttpResponse.json({ job_ids: [...store.skips] })),
    http.put('*/api/applications/v1/skips/:jobId', ({ params }) => {
      store.skips.add(String(params.jobId))
      return new HttpResponse(null, { status: 204 })
    }),
    http.delete('*/api/applications/v1/skips/:jobId', ({ params }) => {
      store.skips.delete(String(params.jobId))
      return new HttpResponse(null, { status: 204 })
    }),
    http.delete('*/api/applications/v1/skips', () => {
      store.skips.clear()
      return new HttpResponse(null, { status: 204 })
    }),
  ]
}
