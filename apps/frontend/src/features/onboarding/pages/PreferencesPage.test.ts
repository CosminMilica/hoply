import { afterEach, describe, expect, it } from 'vitest'
import { locale } from '@/features/common/i18n'
import { click, mountPage, settle, useServices } from '@/test/hoply'
import { sampleProfile } from '@/test/profile-service'
import PreferencesPage from './PreferencesPage.vue'

const text = () => document.body.textContent ?? ''
const language = (code: string) => document.querySelector<HTMLButtonElement>(`[role="radio"][lang="${code}"]`)!
const domain = (key: string) => document.querySelector<HTMLButtonElement>(`[data-test="domain-${key}"]`)!

afterEach(() => {
  document.body.innerHTML = ''
})

describe('PreferencesPage', () => {
  it('changes the language and the fields, saving each change, and keeps the last field', async () => {
    const { profile } = useServices(sampleProfile({ app_language: 'en', domains: ['it', 'finance'] }))
    await mountPage(PreferencesPage, { at: '/preferences' })
    expect(text()).toContain('Preferences')
    expect(domain('finance').getAttribute('aria-pressed')).toBe('true')
    expect(text()).toContain('2 fields picked')

    await click(domain('finance'))
    await settle()
    expect(profile.domains).toEqual(['it'])
    expect(text()).toContain('Keep at least one field.')
    await click(domain('it'))
    await settle()
    expect(profile.domains).toEqual(['it'])
    expect(domain('it').getAttribute('aria-pressed')).toBe('true')

    await click(language('ro'))
    await settle()
    expect(locale.value).toBe('ro')
    expect(profile.app_language).toBe('ro')
    expect(text()).toContain('Preferințe')
    expect(text()).toContain('Păstrează cel puțin un domeniu.')
  })
})
