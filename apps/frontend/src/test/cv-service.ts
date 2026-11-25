import { http, HttpResponse } from 'msw'
import { cvRequestSchema, type Cv } from '@/features/cv/api/cv.schema'
import { jobSchema } from '@/features/jobs/api/jobs.schema'
import type { Profile } from '@/features/profile/api/profile.schema'
import { generateCv } from './cv-engine/cv'
import { toEngineJob } from './cv-engine/job'
import { prepQuestions } from './cv-engine/questions'

interface Answer { status: number, body?: unknown }

/** Where the cv service reads the profile and a job; undefined when the service cannot answer or knows no such job. */
export interface CvSources {
  profile: () => Profile | undefined
  job: (id: string) => unknown
}

interface CvBackendOptions {
  now?: () => Date
  /** How many reads of a new CV answer that it is still being written. */
  writing?: number
  /** Every CV fails instead of being written. */
  failing?: boolean
}

const problem = (status: number, code: string, message: string, field?: string): Answer =>
  ({ status, body: { error: { code, message, type: 'Error', context: field ? { field } : {} } } })

const unwritten = { highlights: null, profile: null, tailored: null, ats: null, preview: null, tailoring: null, written_by: null, model: '', refused: 0, rules_reason: '' as const, limit_until: null }

/**
 * The cv service in memory, by docs/api/cv.md, writing with the web app's former rules engine (src/test/cv-engine).
 * A CV is queued, then answers `writing` for `writing` reads, then `ready` (or `failed`).
 */
export function createCvBackend(sources: CvSources, { now = () => new Date('2026-11-25T10:00:00Z'), writing = 0, failing = false }: CvBackendOptions = {}) {
  const cvs: { cv: Cv, pending: number }[] = []
  let sequence = 0
  const view = (entry: { cv: Cv, pending: number }): Cv => {
    if (entry.pending > 0) return { ...entry.cv, ...unwritten, status: 'writing' }
    return failing ? { ...entry.cv, ...unwritten, status: 'failed' } : entry.cv
  }
  const readJob = (id: string) => {
    const job = sources.job(id)
    return job ? toEngineJob(jobSchema.parse(job)) : null
  }

  return {
    cvs,
    questions(jobId: string): Answer {
      const job = readJob(jobId)
      if (!job) return problem(404, 'NOT_FOUND', 'Job not found')
      const profile = sources.profile()
      if (!profile) return problem(503, 'DEPENDENCY_UNAVAILABLE', 'profile is unavailable')
      return { status: 200, body: prepQuestions(job, profile) }
    },
    create(body: unknown): Answer {
      const parsed = cvRequestSchema.safeParse(body)
      if (!parsed.success) return problem(422, 'VALIDATION_FAILED', parsed.error.issues[0]?.message ?? 'Invalid request', String(parsed.error.issues[0]?.path[0] ?? ''))
      const job = readJob(parsed.data.job_id)
      if (!job) return problem(404, 'NOT_FOUND', 'Job not found')
      const profile = sources.profile()
      if (!profile) return problem(503, 'DEPENDENCY_UNAVAILABLE', 'profile is unavailable')
      const at = now().toISOString()
      sequence += 1
      const id = `00000000-0000-4000-9000-${String(sequence).padStart(12, '0')}`
      const cv: Cv = { id, created_at: at, updated_at: at, ...generateCv(structuredClone(profile), job, parsed.data, now()) }
      cvs.unshift({ cv, pending: writing })
      return { status: 202, body: { ...cv, ...unwritten, status: 'queued' } }
    },
    get(id: string): Answer {
      const entry = cvs.find(item => item.cv.id === id)
      if (!entry) return problem(404, 'NOT_FOUND', 'CV not found')
      const answer = view(entry)
      entry.pending = Math.max(0, entry.pending - 1)
      return { status: 200, body: answer }
    },
    list(jobId: string | null): Answer {
      return { status: 200, body: { items: cvs.filter(entry => !jobId || entry.cv.job_id === jobId).map(view) } }
    },
    remove(id: string): Answer {
      const at = cvs.findIndex(item => item.cv.id === id)
      if (at < 0) return problem(404, 'NOT_FOUND', 'CV not found')
      cvs.splice(at, 1)
      return { status: 204 }
    },
  }
}

export type CvBackend = ReturnType<typeof createCvBackend>

const reply = (answer: Answer) =>
  answer.body === undefined ? new HttpResponse(null, { status: answer.status }) : HttpResponse.json(answer.body as object, { status: answer.status })

/** MSW handlers for the in-memory cv service. */
export function cvServiceHandlers(sources: CvSources, options: CvBackendOptions = {}) {
  const backend = createCvBackend(sources, options)
  return {
    backend,
    handlers: [
      http.get('*/api/cv/v1/jobs/:jobId/questions', ({ params }) => reply(backend.questions(String(params.jobId)))),
      http.post('*/api/cv/v1/cvs', async ({ request }) => reply(backend.create(await request.json().catch(() => null)))),
      http.get('*/api/cv/v1/cvs', ({ request }) => reply(backend.list(new URL(request.url).searchParams.get('job_id')))),
      http.get('*/api/cv/v1/cvs/:id', ({ params }) => reply(backend.get(String(params.id)))),
      http.delete('*/api/cv/v1/cvs/:id', ({ params }) => reply(backend.remove(String(params.id)))),
    ],
  }
}
