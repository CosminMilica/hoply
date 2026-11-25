import { getApiClient } from '@/shared/api/client'
import { applicationListSchema, applicationSchema, skipListSchema, type ApplicationStatus } from './applications.schema'

const BASE = '/api/applications/v1'

export const applicationsApi = {
  async list() {
    return applicationListSchema.parse(await getApiClient().get(`${BASE}/applications`).json()).items
  },
  async setStatus(jobId: string, status: ApplicationStatus) {
    return applicationSchema.parse(
      await getApiClient().put(`${BASE}/applications/${encodeURIComponent(jobId)}`, { json: { status } }).json(),
    )
  },
  async remove(jobId: string) {
    await getApiClient().delete(`${BASE}/applications/${encodeURIComponent(jobId)}`)
  },
  async skips() {
    return skipListSchema.parse(await getApiClient().get(`${BASE}/skips`).json()).job_ids
  },
  async skip(jobId: string) {
    await getApiClient().put(`${BASE}/skips/${encodeURIComponent(jobId)}`)
  },
  async unskip(jobId: string) {
    await getApiClient().delete(`${BASE}/skips/${encodeURIComponent(jobId)}`)
  },
  async clearSkips() {
    await getApiClient().delete(`${BASE}/skips`)
  },
}
