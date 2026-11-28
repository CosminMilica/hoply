import { afterEach, describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { locale } from '@/features/common/i18n'
import { button, click, mountPage, useServices } from '@/test/hoply'
import { sampleProfile } from '@/test/profile-service'
import OnboardingPage from './OnboardingPage.vue'

const text = () => document.body.textContent ?? ''
const language = (code: string) => document.querySelector<HTMLButtonElement>(`[role="radio"][lang="${code}"]`)!
const newcomer = () => useServices(sampleProfile({ app_language: '', domains: [] }))

afterEach(() => {
  document.body.innerHTML = ''
})

describe('OnboardingPage', () => {
  it('asks for the language, switching the app to it at once, then the fields, and saves both', async () => {
    const { profile } = newcomer()
    const { router } = await mountPage(OnboardingPage, { at: '/welcome?to=/saved' })
    expect(text()).toContain('Step 1 of 2')
    expect(text()).toContain('Welcome to Hoply')
    expect(language('en').getAttribute('aria-checked')).toBe('true')

    await click(language('ro'))
    expect(locale.value).toBe('ro')
    expect(text()).toContain('Bine ai venit în Hoply')
    await click(button('Continuă'))

    expect(text()).toContain('Pasul 2 din 2')
    expect(text()).toContain('Ce domenii te interesează?')
    expect(text()).toContain('Alege cel puțin un domeniu')
    expect(button('Gata, arată-mi joburile').disabled).toBe(true)
    await click(document.querySelector<HTMLButtonElement>('[data-test="domain-accounting"]')!)
    await click(document.querySelector<HTMLButtonElement>('[data-test="domain-banking"]')!)
    expect(text()).toContain('2 domenii alese')
    expect(text()).toContain('Filtre: Contabilitate generală · Furnizori (AP)')
    await click(document.querySelector<HTMLButtonElement>('[data-test="domain-accounting"]')!)
    await click(document.querySelector<HTMLButtonElement>('[data-test="domain-accounting"]')!)

    await click(button('Gata, arată-mi joburile'))
    expect(profile.app_language).toBe('ro')
    expect(profile.domains).toEqual(['banking', 'accounting'])
    expect(router.currentRoute.value.fullPath).toBe('/saved')
  })

  it('starts from what the person picked before, goes back a step, and opens Find jobs after', async () => {
    useServices(sampleProfile({ app_language: 'ro', domains: ['hr'] }))
    const { router } = await mountPage(OnboardingPage, { at: '/welcome?to=//elsewhere.example' })
    expect(language('ro').getAttribute('aria-checked')).toBe('true')
    await click(button('Continuă'))
    expect(document.querySelector('[data-test="domain-hr"]')!.getAttribute('aria-pressed')).toBe('true')
    expect(text()).toContain('1 domeniu ales')
    await click(button('Înapoi'))
    expect(text()).toContain('Pasul 1 din 2')
    await click(language('en'))
    await click(button('Continue'))
    await click(button('Done, show me the jobs'))
    expect(router.currentRoute.value.name).toBe('hoply-jobs')
  })

  it('says when the choices could not be saved, and when the profile could not be read', async () => {
    newcomer()
    server.use(http.patch('*/api/profile/v1/profiles/me', () => HttpResponse.json({ error: { code: 'INTERNAL' } }, { status: 500 })))
    const { router } = await mountPage(OnboardingPage, { at: '/welcome' })
    await click(button('Continue'))
    await click(document.querySelector<HTMLButtonElement>('[data-test="domain-it"]')!)
    await click(button('Done, show me the jobs'))
    expect(text()).toContain('We could not save your choices. Try again.')
    expect(router.currentRoute.value.name).toBe('hoply-onboarding')

    document.body.innerHTML = ''
    server.use(http.get('*/api/profile/v1/profiles/me', () => HttpResponse.json({ error: { code: 'INTERNAL' } }, { status: 500 })))
    await mountPage(OnboardingPage, { at: '/welcome' })
    expect(text()).toContain('We could not load your profile.')
  })
})
