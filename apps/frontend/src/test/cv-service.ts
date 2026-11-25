import { http, HttpResponse } from 'msw'
import { cvRequestSchema, type Cv, type Letter, type Probe } from '@/features/cv/api/cv.schema'
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
  /** Claude's questions for every job; unavailable by default. */
  probe?: Probe
  /** How many reads of a letter asked for answer that it is still being written. */
  letterWriting?: number
  /** Why every letter fails, instead of being written. */
  letterFailure?: Letter['reason']
}

const problem = (status: number, code: string, message: string, field?: string): Answer =>
  ({ status, body: { error: { code, message, type: 'Error', context: field ? { field } : {} } } })

const unwritten = { highlights: null, profile: null, tailored: null, ats: null, preview: null, tailoring: null, written_by: null, model: '', refused: 0, rules_reason: '' as const, limit_until: null }

/**
 * The cv service in memory, by docs/api/cv.md, writing with the web app's former rules engine (src/test/cv-engine).
 * A CV is queued, then answers `writing` for `writing` reads, then `ready` (or `failed`).
 */
export function createCvBackend(sources: CvSources, {
  now = () => new Date('2026-11-25T10:00:00Z'),
  writing = 0,
  failing = false,
  probe = { status: 'unavailable', questions: [] },
  letterWriting = 0,
  letterFailure = '',
}: CvBackendOptions = {}) {
  const cvs: { cv: Cv, pending: number }[] = []
  const letters = new Map<string, { recruiter: string, pending: number }>()
  const letterFiles: { id: string, format: string, template: string, text: string }[] = []
  const none: Letter = { status: 'none', recruiter: '', texts: null, model: '', refused: 0, reason: '', limit_until: null }
  const readyCv = (id: string) => {
    const entry = cvs.find(item => item.cv.id === id)
    return entry && view(entry).status === 'ready' ? entry.cv : null
  }
  const written = (cv: Cv, recruiter: string): Letter => {
    const name = cv.tailored?.name ?? ''
    const first = recruiter.split(' ')[0]
    const hello = first ? `Hi ${first},` : 'Hello,'
    const signed = (closing: string) => (name ? `${closing}\n${name}` : closing)
    return {
      ...none,
      status: 'ready',
      recruiter,
      model: 'claude-sonnet-5-5',
      texts: {
        cover_letter: [`Dear ${recruiter || `${cv.company} hiring team`},`, `You need someone for the ${cv.job_title} role, and I have done this work.`,
          'I would welcome a conversation.', signed('Kind regards,')].join('\n\n'),
        connection_note: `${first ? `Hi ${first}, ` : ''}I applied for the ${cv.job_title} role and would like to connect.`,
        message: { subject: `Application for ${cv.job_title}`, body: `${hello}\n\nI applied for the ${cv.job_title} role today.\n\n${signed('Best regards,')}` },
        follow_up: { subject: `Following up on my application for ${cv.job_title}`, body: `${hello}\n\nI applied last week.\n\n${signed('Best regards,')}` },
      },
    }
  }
  const analyses: string[] = []
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
    analyses,
    analyze(jobId: string): Answer {
      if (!readJob(jobId)) return problem(404, 'NOT_FOUND', 'Job not found')
      analyses.push(jobId)
      return { status: 202, body: { status: probe.status === 'unavailable' ? 'unavailable' : 'pending' } }
    },
    probe(jobId: string): Answer {
      return readJob(jobId) ? { status: 200, body: probe } : problem(404, 'NOT_FOUND', 'Job not found')
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
    letters,
    letterFiles,
    letter(id: string): Answer {
      const cv = cvs.find(item => item.cv.id === id)?.cv
      if (!cv) return problem(404, 'NOT_FOUND', 'CV not found')
      const entry = letters.get(id)
      if (!entry) return { status: 200, body: none }
      if (entry.pending > 0) {
        entry.pending -= 1
        return { status: 200, body: { ...none, status: 'writing', recruiter: entry.recruiter } }
      }
      if (letterFailure) return { status: 200, body: { ...none, status: 'failed', recruiter: entry.recruiter, reason: letterFailure } }
      return { status: 200, body: written(cv, entry.recruiter) }
    },
    writeLetter(id: string, body: unknown): Answer {
      if (!cvs.some(item => item.cv.id === id)) return problem(404, 'NOT_FOUND', 'CV not found')
      if (!readyCv(id)) return problem(409, 'CONSTRAINT_VIOLATION', 'the CV is not ready yet')
      const recruiter = String((body as { recruiter?: unknown } | null)?.recruiter ?? '').trim().split(/\s+/).filter(Boolean).join(' ')
      if (recruiter.length > 80) return problem(422, 'VALIDATION_FAILED', "'recruiter' must be at most 80 characters", 'recruiter')
      letters.set(id, { recruiter, pending: letterWriting })
      return { status: 202, body: { ...none, status: 'writing', recruiter } }
    },
    letterFile(id: string, format: string, template: string, body: unknown): Answer {
      const cv = readyCv(id)
      if (!cv) return problem(404, 'NOT_FOUND', 'CV not found')
      const text = String((body as { text?: unknown } | null)?.text ?? '').trim()
      if (!text) return problem(422, 'VALIDATION_FAILED', "'text' must be 1 to 8000 characters", 'text')
      letterFiles.push({ id, format, template, text })
      return { status: 200, body: { name: cv.file_name.replace('_CV_', '_Cover_Letter_').replace(/\.pdf$/, `.${format}`) } }
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
      http.post('*/api/cv/v1/jobs/:jobId/analysis', ({ params }) => reply(backend.analyze(String(params.jobId)))),
      http.get('*/api/cv/v1/jobs/:jobId/probe', ({ params }) => reply(backend.probe(String(params.jobId)))),
      http.post('*/api/cv/v1/cvs', async ({ request }) => reply(backend.create(await request.json().catch(() => null)))),
      http.get('*/api/cv/v1/cvs', ({ request }) => reply(backend.list(new URL(request.url).searchParams.get('job_id')))),
      http.get('*/api/cv/v1/cvs/:id', ({ params }) => reply(backend.get(String(params.id)))),
      http.delete('*/api/cv/v1/cvs/:id', ({ params }) => reply(backend.remove(String(params.id)))),
      http.get('*/api/cv/v1/cvs/:id/letter', ({ params }) => reply(backend.letter(String(params.id)))),
      http.post('*/api/cv/v1/cvs/:id/letter', async ({ params, request }) => reply(backend.writeLetter(String(params.id), await request.json().catch(() => null)))),
      http.post('*/api/cv/v1/cvs/:id/letter/file', async ({ params, request }) => {
        const query = new URL(request.url).searchParams
        const answer = backend.letterFile(String(params.id), query.get('format') ?? '', query.get('template') ?? '', await request.json().catch(() => null))
        if (answer.status !== 200) return reply(answer)
        return new HttpResponse('%PDF-letter', { status: 200, headers: { 'content-disposition': `attachment; filename=${(answer.body as { name: string }).name}` } })
      }),
    ],
  }
}
