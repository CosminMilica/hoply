import { describe, expect, it } from 'vitest'
import { server } from '@/shared/mocks/node'
import { applicationsApi } from '../api/applicationsApi'
import { createApplicationsHandlers } from './handlers'

describe('the mocked applications service', () => {
  it('tracks the status of saved jobs', async () => {
    server.use(...createApplicationsHandlers())
    expect(await applicationsApi.list()).toEqual([])
    await applicationsApi.setStatus('1', 'saved')
    const updated = await applicationsApi.setStatus('1', 'cv_ready')
    expect(updated).toMatchObject({ job_id: '1', status: 'cv_ready' })
    await applicationsApi.setStatus('2', 'saved')
    expect((await applicationsApi.list()).map(item => item.job_id).sort()).toEqual(['1', '2'])
    await applicationsApi.remove('1')
    expect((await applicationsApi.list()).map(item => item.job_id)).toEqual(['2'])
    await expect(applicationsApi.remove('1')).rejects.toThrow()
  })

  it('refuses an unknown status', async () => {
    server.use(...createApplicationsHandlers())
    const response = await fetch(`${window.location.origin}/api/applications/v1/applications/1`, { method: 'PUT', body: '{"status":"lost"}' })
    expect(response.status).toBe(422)
    expect((await fetch(`${window.location.origin}/api/applications/v1/applications/1`, { method: 'PUT', body: 'nope' })).status).toBe(422)
  })

  it('remembers skipped jobs until they are brought back', async () => {
    server.use(...createApplicationsHandlers())
    await applicationsApi.skip('1')
    await applicationsApi.skip('2')
    await applicationsApi.unskip('1')
    expect(await applicationsApi.skips()).toEqual(['2'])
    await applicationsApi.clearSkips()
    expect(await applicationsApi.skips()).toEqual([])
  })
})
