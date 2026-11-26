import { describe, expect, it } from 'vitest'
import router, { hoplyRoutes } from './index'

describe('the application router', () => {
  it('composes hoply’s pages into the app layout, starting on the profile', async () => {
    expect(hoplyRoutes.map(route => route.name)).toEqual(['hoply-profile', 'hoply-jobs', 'hoply-job-add', 'hoply-job', 'hoply-prep', 'hoply-cv', 'hoply-saved', 'hoply-skills'])
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
    for (const route of hoplyRoutes) {
      const component = (route as { component: () => Promise<unknown> }).component
      expect(await component()).toBeTruthy()
    }
  })
})
