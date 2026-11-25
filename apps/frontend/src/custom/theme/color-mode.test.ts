import { describe, expect, it } from 'vitest'
import { preferDarkColorMode } from './color-mode'

describe('preferDarkColorMode', () => {
  it('starts dark when the person has not chosen', () => {
    localStorage.removeItem('vueuse-color-scheme')
    preferDarkColorMode()
    expect(localStorage.getItem('vueuse-color-scheme')).toBe('dark')
  })

  it('keeps a choice already made', () => {
    localStorage.setItem('vueuse-color-scheme', 'light')
    preferDarkColorMode()
    expect(localStorage.getItem('vueuse-color-scheme')).toBe('light')
  })
})
