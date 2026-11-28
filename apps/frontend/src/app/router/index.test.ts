import { beforeAll, describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { setLocale } from '@/features/common/i18n'
import { sampleProfile } from '@/test/profile-service'
import { auth } from '../auth'
import { queryClient } from '../providers/query-client'
import router, { hoplyRoutes, onboardingRoute } from './index'

describe('the application router', () => {
  beforeAll(async () => {
    server.use(http.get('*/auth/userinfo', () => HttpResponse.json({ userId: 'u1', email: 'ana@example.com', preferredUsername: 'ana', roles: ['user'], customerId: 't1' })))
    await auth.init?.()
  })
  const profile = (domains: string[]) => {
    queryClient.clear()
    server.use(http.get('*/api/profile/v1/profiles/me', () => HttpResponse.json(sampleProfile({ domains }))))
  }

  it('composes hoply’s pages into the app layout, starting on the profile', async () => {
    profile(['it'])
    expect(hoplyRoutes.map(route => route.name)).toEqual(['hoply-profile', 'hoply-jobs', 'hoply-job-add', 'hoply-job', 'hoply-prep', 'hoply-cv', 'hoply-saved', 'hoply-applications', 'hoply-skills', 'hoply-preferences'])
    expect(router.resolve({ name: 'hoply-prep', params: { id: '3' } }).path).toBe('/jobs/3/prepare')
    expect(router.resolve({ name: 'hoply-cv', params: { id: '3' } }).path).toBe('/jobs/3/cv')
    await router.push('/')
    expect(router.currentRoute.value.name).toBe('hoply-profile')
    await router.push('/jobs/add')
    expect(router.currentRoute.value.name).toBe('hoply-job-add')
    const add = router.currentRoute.value.matched.at(-1)!
    expect(add.meta.menu).toBe('jobs')
    await router.push('/jobs/add?from=saved')
    expect(add.meta.menu).toBe('saved')
    await router.push('/jobs/1')
    expect(router.currentRoute.value.matched.at(-1)!.meta.menu).toBe('saved')
    await router.push('/jobs/1/cv')
    expect(router.currentRoute.value.matched.at(-1)!.meta.menu).toBe('applications')
    expect(router.resolve({ name: 'hoply-skills' }).path).toBe('/skills')
    for (const route of [...hoplyRoutes, onboardingRoute]) {
      const component = (route as { component: () => Promise<unknown> }).component
      expect(await component()).toBeTruthy()
    }
  })

  it('titles the pages in the app language, and sends a person with no fields to onboarding first', async () => {
    profile([])
    await router.push('/saved')
    expect(router.currentRoute.value.name).toBe('hoply-onboarding')
    expect(router.currentRoute.value.query.to).toBe('/saved')
    expect(router.currentRoute.value.meta.requiresAuth).toBe(true)
    const saved = router.resolve({ name: 'hoply-saved' }).matched.at(-1)!
    expect(saved.meta.title).toBe('Saved jobs')
    setLocale('ro')
    expect(saved.meta.title).toBe('Joburi salvate')
    expect(router.resolve({ name: 'hoply-preferences' }).path).toBe('/preferences')
  })
})
