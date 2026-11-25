import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { getApiClient } from '@/shared/api/client'
import { serviceErrorMessage } from './errors'

const failure = async () => getApiClient().get('/api/test/v1/thing').json().catch((error: unknown) => error)

describe('serviceErrorMessage', () => {
  it('uses the message of the error envelope', async () => {
    server.use(http.get('*/api/test/v1/thing', () => HttpResponse.json({ message: 'name is too long', type: 'ValidationError' }, { status: 422 })))
    expect(await serviceErrorMessage(await failure())).toBe('name is too long')
  })

  it('uses the message of the services’ error envelope', async () => {
    server.use(http.get('*/api/test/v1/thing', () => HttpResponse.json({ error: { code: 'VALIDATION', message: 'title is required', context: { field: 'title' } } }, { status: 422 })))
    expect(await serviceErrorMessage(await failure())).toBe('title is required')
  })

  it('falls back to a generic message without one', async () => {
    server.use(http.get('*/api/test/v1/thing', () => HttpResponse.json({}, { status: 500 })))
    expect(await serviceErrorMessage(await failure())).toMatch(/^Request failed \(500/)
    expect(await serviceErrorMessage(new Error('offline'))).toBe('offline')
  })
})
