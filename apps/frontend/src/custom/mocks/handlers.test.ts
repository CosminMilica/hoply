import { describe, expect, it } from 'vitest'
import { handlers, mockedServiceHandlers } from './handlers'

describe('the browser mocks', () => {
  it('mock the services that do not exist yet, and never the profile or the jobs', () => {
    const paths = mockedServiceHandlers().map(handler => String(handler.info.path))
    expect(paths.some(path => path.includes('/api/applications/v1/'))).toBe(true)
    expect(paths.some(path => path.includes('/api/cv/v1/'))).toBe(true)
    expect(handlers().map(handler => String(handler.info.path)).some(path => /\/api\/(profile|jobs)\//.test(path))).toBe(false)
  })
})
