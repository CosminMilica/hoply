import { getApiClient } from '@/shared/api/client'
import { cvListSchema, cvSchema, prepQuestionsSchema, type CvRequest } from './cv.schema'

const BASE = '/api/cv/v1'

export const cvApi = {
  async questions(jobId: string) {
    return prepQuestionsSchema.parse(
      await getApiClient().get(`${BASE}/jobs/${encodeURIComponent(jobId)}/questions`).json(),
    )
  },
  async generate(body: CvRequest) {
    return cvSchema.parse(await getApiClient().post(`${BASE}/cvs`, { json: body }).json())
  },
  /** The CVs made for a job, newest first. */
  async forJob(jobId: string) {
    return cvListSchema.parse(await getApiClient().get(`${BASE}/cvs`, { searchParams: { job_id: jobId } }).json()).items
  },
}
