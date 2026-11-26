import { getApiClient } from '@/shared/api/client'
import { cvListSchema, cvSchema, prepQuestionsSchema, probeSchema, analysisStartSchema, letterSchema, type CvRequest, type CvTemplate } from './cv.schema'

const BASE = '/api/cv/v1'

export const cvApi = {
  async questions(jobId: string) {
    return prepQuestionsSchema.parse(
      await getApiClient().get(`${BASE}/jobs/${encodeURIComponent(jobId)}/questions`).json(),
    )
  },
  /** The questions that make every CV stronger, asked once for the whole profile. */
  async profileProbe() {
    return probeSchema.parse(await getApiClient().get(`${BASE}/probe`).json())
  },
  /** Claude's questions for the job, asked in the background: poll while `pending`. */
  async probe(jobId: string) {
    return probeSchema.parse(await getApiClient().get(`${BASE}/jobs/${encodeURIComponent(jobId)}/probe`).json())
  },
  /** Starts Claude's reading of the job's ad in the background; its requirements replace the rules' ones. */
  async analyze(jobId: string) {
    return analysisStartSchema.parse(await getApiClient().post(`${BASE}/jobs/${encodeURIComponent(jobId)}/analysis`).json())
  },
  /** Queues a CV; it is written asynchronously (follow it with `get`). */
  async generate(body: CvRequest) {
    return cvSchema.parse(await getApiClient().post(`${BASE}/cvs`, { json: body }).json())
  },
  async get(id: string) {
    return cvSchema.parse(await getApiClient().get(`${BASE}/cvs/${encodeURIComponent(id)}`).json())
  },
  /** The CVs made for a job, newest first. */
  /** Every CV of the person, newest first. */
  async all() {
    return cvListSchema.parse(await getApiClient().get(`${BASE}/cvs`).json()).items
  },
  async forJob(jobId: string) {
    return cvListSchema.parse(await getApiClient().get(`${BASE}/cvs`, { searchParams: { job_id: jobId } }).json()).items
  },
  async letter(cvId: string) {
    return letterSchema.parse(await getApiClient().get(`${BASE}/cvs/${encodeURIComponent(cvId)}/letter`).json())
  },
  /** Starts writing the CV's cover letter and messages; follow it with `letter`. */
  async writeLetter(cvId: string, recruiter: string) {
    return letterSchema.parse(await getApiClient().post(`${BASE}/cvs/${encodeURIComponent(cvId)}/letter`, { json: { recruiter } }).json())
  },
  /** The cover letter as the person edited it, as a file with the CV's header. */
  async letterFile(cvId: string, format: 'pdf' | 'docx', template: CvTemplate, text: string) {
    const response = await getApiClient().post(`${BASE}/cvs/${encodeURIComponent(cvId)}/letter/file`, { json: { text }, searchParams: { format, template } })
    const name = /filename="?([^";]+)"?/.exec(response.headers.get('content-disposition') ?? '')?.[1] ?? `Cover_Letter.${format}`
    return { name, blob: await response.blob() }
  },
}
