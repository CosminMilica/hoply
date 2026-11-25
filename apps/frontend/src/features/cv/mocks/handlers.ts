import { bypass, http, HttpResponse } from 'msw'
import { jobSchema } from '@/features/jobs/api/jobs.schema'
import { profileSchema, type Profile } from '@/features/profile/api/profile.schema'
import { cvRequestSchema, type Cv } from '../api/cv.schema'
import { generateCv } from './engine/cv'
import { toEngineJob, type EngineJob } from './engine/job'
import { prepQuestions } from './engine/questions'

/** How the mocked service reads the profile: the real profile service, past the mocks. */
export type ProfileLoader = (request: Request) => Promise<Profile>

export const loadProfileFromService: ProfileLoader = async (request) => {
  const response = await fetch(bypass(new URL('/api/profile/v1/profiles/me', request.url)))
  if (!response.ok) throw new Error(`The profile service answered ${response.status}`)
  return profileSchema.parse(await response.json())
}

/** How the mocked service reads a job: the real jobs service, past the mocks; null when there is none. */
export type JobLoader = (request: Request, id: string) => Promise<EngineJob | null>

export const loadJobFromService: JobLoader = async (request, id) => {
  const response = await fetch(bypass(new URL(`/api/jobs/v1/jobs/${encodeURIComponent(id)}`, request.url)))
  if (response.status === 404) return null
  if (!response.ok) throw new Error(`The jobs service answered ${response.status}`)
  return toEngineJob(jobSchema.parse(await response.json()))
}

interface CvHandlerOptions {
  loadJob?: JobLoader
  loadProfile?: ProfileLoader
  now?: () => Date
}

const problem = (status: number, message: string, type: string) =>
  HttpResponse.json({ message, type, detail: null }, { status })

/**
 * The future cv service: `/api/cv/v1/...`. It reads the job and the profile from their services when it makes a CV and
 * keeps the whole profile in the CV as a snapshot, so later profile edits do not change a CV already made.
 */
export function createCvHandlers({ loadJob = loadJobFromService, loadProfile = loadProfileFromService, now = () => new Date() }: CvHandlerOptions = {}) {
  const cvs: Cv[] = []
  let sequence = 0

  async function profileOrProblem(request: Request) {
    try {
      return await loadProfile(request)
    }
    catch {
      return null
    }
  }
  async function jobOrProblem(request: Request, id: string) {
    try {
      return await loadJob(request, id)
    }
    catch {
      return undefined
    }
  }

  return [
    http.get('*/api/cv/v1/jobs/:jobId/questions', async ({ params, request }) => {
      const job = await jobOrProblem(request, String(params.jobId))
      if (job === undefined) return problem(502, 'The job could not be read', 'UpstreamError')
      if (!job) return problem(404, 'Job not found', 'NotFound')
      const profile = await profileOrProblem(request)
      if (!profile) return problem(502, 'The profile could not be read', 'UpstreamError')
      return HttpResponse.json(prepQuestions(job, profile))
    }),
    http.post('*/api/cv/v1/cvs', async ({ request }) => {
      const parsed = cvRequestSchema.safeParse(await request.json().catch(() => null))
      if (!parsed.success) return problem(422, parsed.error.issues[0]?.message ?? 'Invalid request', 'ValidationError')
      const job = await jobOrProblem(request, parsed.data.job_id)
      if (job === undefined) return problem(502, 'The job could not be read', 'UpstreamError')
      if (!job) return problem(404, 'Job not found', 'NotFound')
      const profile = await profileOrProblem(request)
      if (!profile) return problem(502, 'The profile could not be read', 'UpstreamError')
      const at = now()
      sequence += 1
      const cv: Cv = { id: `cv-${sequence}`, created_at: at.toISOString(), ...generateCv(profile, job, parsed.data, at) }
      cvs.unshift(cv)
      return HttpResponse.json(cv, { status: 201 })
    }),
    http.get('*/api/cv/v1/cvs', ({ request }) => {
      const jobId = new URL(request.url).searchParams.get('job_id')
      return HttpResponse.json({ items: cvs.filter(cv => !jobId || cv.job_id === jobId) })
    }),
    http.get('*/api/cv/v1/cvs/:id', ({ params }) => {
      const cv = cvs.find(item => item.id === params.id)
      return cv ? HttpResponse.json(cv) : problem(404, 'CV not found', 'NotFound')
    }),
  ]
}
