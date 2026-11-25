import { describe, expect, it } from 'vitest'
import { theme, ui } from './nuxt-ui.config'

describe('the Nuxt UI theme', () => {
  it('uses the hoply blue, controls of at least 44px and keeps the error colour', () => {
    expect(ui?.colors).toMatchObject({ primary: 'hoply', error: 'ember' })
    expect(JSON.stringify(ui?.button)).toContain('min-h-11')
    expect(theme?.colors).toContain('error')
  })
})
