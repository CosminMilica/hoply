import { describe, expect, it } from 'vitest'
import { server } from '@/shared/mocks/node'
import { serviceErrorMessage } from '@/features/common/errors'
import { applicationsServiceHandlers } from '@/test/applications-service'
import { seedId } from '@/test/jobs-seed'
import { applicationsApi } from './applicationsApi'
import { applicationSchema, statusSince } from './applications.schema'

describe('applicationsApi', () => {
  it('tracks a job through its statuses, with the time of each', async () => {
    server.use(...applicationsServiceHandlers().handlers)
    expect(await applicationsApi.list()).toEqual([])
    const saved = await applicationsApi.setStatus(seedId(1), 'saved')
    expect(saved).toMatchObject({ status: 'saved', saved_at: '2026-11-25T09:00:01.000Z', cv_ready_at: null, applied_at: null })
    await applicationsApi.setStatus(seedId(1), 'applied')
    const ready = await applicationsApi.setStatus(seedId(1), 'cv_ready')
    expect(ready).toMatchObject({ status: 'applied', saved_at: '2026-11-25T09:00:01.000Z', applied_at: '2026-11-25T09:00:02.000Z', cv_ready_at: '2026-11-25T09:00:03.000Z' })
    await applicationsApi.setStatus(seedId(2), 'saved')
    expect((await applicationsApi.list()).map(item => item.job_id)).toEqual([seedId(2), seedId(1)])
    await applicationsApi.remove(seedId(1))
    await expect(applicationsApi.remove(seedId(1))).rejects.toThrow()
  })

  it('refuses a job id that is not a UUID, and an unknown status', async () => {
    server.use(...applicationsServiceHandlers().handlers)
    const refused = await applicationsApi.setStatus('1', 'saved').catch((error: unknown) => error)
    expect(await serviceErrorMessage(refused)).toBe('job_id must be a UUID')
    const unknown = await applicationsApi.setStatus(seedId(1), 'lost' as never).catch((error: unknown) => error)
    expect(await serviceErrorMessage(unknown)).toMatch(/^status must be one of/)
    await expect(applicationsApi.skip('1')).rejects.toThrow()
    await expect(applicationsApi.unskip('1')).rejects.toThrow()
    await expect(applicationsApi.remove('1')).rejects.toThrow()
  })

  it('remembers skipped jobs until they are brought back', async () => {
    server.use(...applicationsServiceHandlers().handlers)
    await applicationsApi.skip(seedId(1))
    await applicationsApi.skip(seedId(2))
    await applicationsApi.skip(seedId(2))
    expect(await applicationsApi.skips()).toEqual([seedId(2), seedId(1)])
    await applicationsApi.unskip(seedId(1))
    await applicationsApi.unskip(seedId(3))
    expect(await applicationsApi.skips()).toEqual([seedId(2)])
    await applicationsApi.clearSkips()
    expect(await applicationsApi.skips()).toEqual([])
  })
})

describe('applications schema', () => {
  it('reads missing or null times as null', () => {
    expect(applicationSchema.parse({ job_id: 'x', status: 'saved', saved_at: null, applied_at: '' })).toMatchObject({
      saved_at: null, cv_ready_at: null, applied_at: null, created_at: null, updated_at: null,
    })
  })

  it('says since when the job has its status', () => {
    const base = { saved_at: '2026-11-25T09:00:00Z', cv_ready_at: '2026-11-26T09:00:00Z', applied_at: '2026-12-02T09:00:00Z' }
    expect(statusSince({ ...base, status: 'saved' })).toBe('Saved 25 Nov')
    expect(statusSince({ ...base, status: 'cv_ready' })).toBe('CV ready 26 Nov')
    expect(statusSince({ ...base, status: 'applied' })).toBe('Applied 2 Dec')
    expect(statusSince({ ...base, status: 'applied', applied_at: null })).toBe('')
    expect(statusSince({ ...base, status: 'saved', saved_at: 'soon' })).toBe('')
  })
})
