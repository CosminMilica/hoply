import { describe, expect, it } from 'vitest'
import { handlers, mockedServiceHandlers } from './handlers'

describe('the browser mocks', () => {
  it('mock none of the services, which are all real', () => {
    expect(mockedServiceHandlers()).toEqual([])
    expect(handlers().map(handler => String(handler.info.path)).some(path => /\/api\/(profile|jobs|applications|cv)\//.test(path))).toBe(false)
  })
})
