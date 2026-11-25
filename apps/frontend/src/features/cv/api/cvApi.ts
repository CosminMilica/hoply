import { getApiClient } from '@/shared/api/client'
import { cvListSchema, cvSchema, prepQuestionsSchema, probeSchema, type CvRequest } from './cv.schema'

const BASE = '/api/cv/v1'

export const cvApi = {
  async questions(jobId: string) {
    return prepQuestionsSchema.parse(
      await getApiClient().get(`${BASE}/jobs/${encodeURIComponent(jobId)}/questions`).json(),
    )
  },
  /** Claude's questions for the job, asked in the background: poll while `pending`. */
  async probe(jobId: string) {
    return probeSchema.parse(await getApiClient().get(`${BASE}/jobs/${encodeURIComponent(jobId)}/probe`).json())
  },
  /** Queues a CV; it is written asynchronously (follow it with `get`). */
  async generate(body: CvRequest) {
    return cvSchema.parse(await getApiClient().post(`${BASE}/cvs`, { json: body }).json())
  },
  async get(id: string) {
    return cvSchema.parse(await getApiClient().get(`${BASE}/cvs/${encodeURIComponent(id)}`).json())
  },
  /** The CVs made for a job, newest first. */
  async forJob(jobId: string) {
    return cvListSchema.parse(await getApiClient().get(`${BASE}/cvs`, { searchParams: { job_id: jobId } }).json()).items
  },
}
