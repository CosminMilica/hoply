import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { QueryClient } from '@tanstack/vue-query'
import { createMemoryHistory, createRouter } from 'vue-router'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { locale } from '@/features/common/i18n'
import { useServices } from '@/test/hoply'
import { sampleProfile } from '@/test/profile-service'
import { installOnboardingGuard, ONBOARDING } from './guard'

const Page = defineComponent({ setup: () => () => h('p') })

function app() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', meta: { layout: 'app' }, component: Page, children: [{ path: 'jobs', name: 'hoply-jobs', component: Page }] },
      { path: '/welcome', name: ONBOARDING, component: Page },
      { path: '/signin', name: 'sign-in', component: Page },
    ],
  })
  installOnboardingGuard(router, new QueryClient({ defaultOptions: { queries: { retry: false } } }))
  return router
}

describe('the onboarding guard', () => {
  it('sends a person with no fields to onboarding, keeping where they were going', async () => {
    useServices(sampleProfile({ domains: [] }))
    const router = app()
    await router.push('/jobs?q=sap')
    expect(router.currentRoute.value.name).toBe(ONBOARDING)
    expect(router.currentRoute.value.query.to).toBe('/jobs?q=sap')
  })

  it('lets a person with fields through, in their language, and leaves pages outside the app alone', async () => {
    useServices(sampleProfile({ app_language: 'ro', domains: ['banking'] }))
    const router = app()
    await router.push('/jobs')
    expect(router.currentRoute.value.name).toBe('hoply-jobs')
    expect(locale.value).toBe('ro')
    await router.push('/signin')
    expect(router.currentRoute.value.name).toBe('sign-in')
    await router.push('/welcome')
    expect(router.currentRoute.value.name).toBe(ONBOARDING)
  })

  it('opens the page when the profile cannot be read', async () => {
    server.use(http.get('*/api/profile/v1/profiles/me', () => HttpResponse.json({ error: { code: 'INTERNAL' } }, { status: 500 })))
    const router = app()
    await router.push('/jobs')
    expect(router.currentRoute.value.name).toBe('hoply-jobs')
  })
})
