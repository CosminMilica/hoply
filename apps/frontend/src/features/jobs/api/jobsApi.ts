import { HTTPError } from 'ky'
import { getApiClient } from '@/shared/api/client'
import { jobDraftSchema, jobListSchema, jobSchema, UNREADABLE_REASONS, type JobFilters, type JobInput, type ReadRequest, type UnreadableReason } from './jobs.schema'

const BASE = '/api/jobs/v1/jobs'

export function jobSearchParams(filters: JobFilters): URLSearchParams {
  const params = new URLSearchParams()
  if (filters.q?.trim()) params.set('q', filters.q.trim())
  if (filters.city?.trim()) params.set('city', filters.city.trim())
  if (filters.country?.trim()) params.set('country', filters.country.trim())
  if (filters.includeRemote !== undefined) params.set('include_remote', String(filters.includeRemote))
  for (const area of filters.areas ?? []) params.append('area', area)
  if (filters.level && filters.level !== 'Any') params.set('level', filters.level.toLowerCase())
  return params
}

/** Why the service could not read a link (`422 PAGE_UNREADABLE`), or null for any other error. */
export function unreadableReason(error: unknown): UnreadableReason | 'unknown' | null {
  if (!(error instanceof HTTPError) || error.response.status !== 422) return null
  const body = (error.data as { error?: { code?: string, context?: { reason?: string } } } | undefined)?.error
  if (body?.code !== 'PAGE_UNREADABLE') return null
  const reason = body.context?.reason as UnreadableReason | undefined
  return reason && UNREADABLE_REASONS.includes(reason) ? reason : 'unknown'
}

export const jobsApi = {
  async list(filters: JobFilters = {}) {
    return jobListSchema.parse(await getApiClient().get(BASE, { searchParams: jobSearchParams(filters) }).json())
  },
  async get(id: string) {
    return jobSchema.parse(await getApiClient().get(`${BASE}/${encodeURIComponent(id)}`).json())
  },
  /** Reads a link or pasted text with the service's rules; saves nothing. */
  async read(body: ReadRequest) {
    return jobDraftSchema.parse(await getApiClient().post(`${BASE}/read`, { json: body, timeout: 20_000 }).json())
  },
  async create(body: JobInput) {
    return jobSchema.parse(await getApiClient().post(BASE, { json: body }).json())
  },
  async remove(id: string) {
    await getApiClient().delete(`${BASE}/${encodeURIComponent(id)}`)
  },
}
