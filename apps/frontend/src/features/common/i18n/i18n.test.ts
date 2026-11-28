import { describe, expect, it, vi } from 'vitest'
import { areaLabel, countEn, countRo, formatDate, formatTime, locale, setLocale, t } from './index'

describe('the app language', () => {
  it('switches the words, the page language and the remembered language; anything else is ignored', () => {
    expect(t.value.common.actions.save).toBe('Save')
    setLocale('ro')
    expect(locale.value).toBe('ro')
    expect(document.documentElement.lang).toBe('ro')
    expect(localStorage.getItem('hoply.locale')).toBe('ro')
    expect(t.value.common.actions.save).toBe('Salvează')
    setLocale('de')
    setLocale('')
    expect(locale.value).toBe('ro')
  })

  it('switches even where nothing can be remembered', () => {
    const setItem = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    setLocale('ro')
    expect(locale.value).toBe('ro')
    setItem.mockRestore()
  })

  it('names the areas in the app language, and an unknown area as it came', () => {
    expect(areaLabel('Payables')).toBe('Payables (AP)')
    expect(areaLabel('Quantum')).toBe('Quantum')
    setLocale('ro')
    expect(areaLabel('Payables')).toBe('Furnizori (AP)')
    expect(t.value.common.validation.atMost(60)).toBe('Cel mult 60 de caractere')
    expect(t.value.common.validation.atMost(3)).toBe('Cel mult 3 caractere')
  })

  it('counts in English and Romanian', () => {
    expect(countEn(1, 'job', 'jobs')).toBe('1 job')
    expect(countEn(3, 'job', 'jobs')).toBe('3 jobs')
    expect([0, 1, 2, 19, 20, 99, 100, 101, 119, 120].map(n => countRo(n, 'job', 'joburi'))).toEqual([
      '0 joburi', '1 job', '2 joburi', '19 joburi', '20 de joburi', '99 de joburi', '100 de joburi', '101 joburi', '119 joburi', '120 de joburi',
    ])
  })

  it('writes dates and times the way each language does', () => {
    const day = new Date(2026, 11, 3, 9, 5)
    expect(formatDate(day, { day: 'numeric', month: 'short' })).toBe('3 Dec')
    expect(formatTime(day)).toBe('09:05')
    setLocale('ro')
    expect(formatDate(day, { day: 'numeric', month: 'long' })).toBe('3 decembrie')
  })
})
