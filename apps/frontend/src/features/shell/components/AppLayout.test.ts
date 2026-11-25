import { afterEach, describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createMemoryHistory, RouterView } from 'vue-router'
import { createApplicationRouter } from '@/shared/application/router'
import { testAuth } from '@/shared/application/testing'
import UApp from '@nuxt/ui/components/App.vue'
import { settle, useServices } from '@/test/hoply'
import AppLayout from './AppLayout.vue'

const Page = defineComponent({ props: { id: { type: String, default: '' } }, setup: props => () => h('p', { 'data-test': 'page' }, props.id) })

async function render(at: string) {
  const router = createApplicationRouter({
    auth: testAuth().auth,
    history: createMemoryHistory(),
    layouts: { app: AppLayout },
    routes: {
      app: [
        { path: 'profile', name: 'hoply-profile', component: Page, meta: { menu: 'profile' } },
        { path: 'jobs', name: 'hoply-jobs', component: Page, meta: { menu: 'jobs' } },
        { path: 'jobs/:id/cv', name: 'hoply-cv', component: Page, props: true, meta: { menu: 'saved' } },
        { path: 'saved', name: 'hoply-saved', component: Page, meta: { menu: 'saved' } },
      ],
    },
  })
  await router.push(at)
  await router.isReady()
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const Root = defineComponent({ setup: () => () => h(UApp, null, () => h(RouterView)) })
  mount(Root, { global: { plugins: [router, createPinia(), [VueQueryPlugin, { queryClient }]] }, attachTo: document.body })
  await flushPromises()
  await settle()
  return router
}

const tabs = () => [...document.querySelectorAll<HTMLAnchorElement>('[data-shell="tabs"] a')]

afterEach(() => {
  document.body.innerHTML = ''
})

describe('AppLayout', () => {
  it('renders the page in the shell with the primary menu as bottom tabs', async () => {
    useServices()
    await render('/profile')
    expect(document.querySelector('[data-test="page"]')).toBeTruthy()
    expect(tabs().map(tab => tab.textContent?.trim())).toEqual(['Profile', 'Find jobs', 'Saved'])
    expect(tabs()[0]!.getAttribute('aria-current')).toBe('page')
    expect(tabs()[1]!.getAttribute('href')).toBe('/jobs')
  })

  it('marks Saved on a CV and shows how many jobs are saved', async () => {
    const { applications } = useServices()
    applications.applications.set('1', { job_id: '1', status: 'saved', updated_at: '2026-11-25T00:00:00Z' })
    applications.applications.set('2', { job_id: '2', status: 'applied', updated_at: '2026-11-25T00:00:00Z' })
    await render('/jobs/1/cv')
    const saved = tabs()[2]!
    expect(saved.getAttribute('aria-current')).toBe('page')
    expect(saved.textContent).toContain('2 saved')
    expect(document.querySelector('[data-shell="header"] a')?.getAttribute('aria-label')).toBe('hoply, go to your profile')
  })

  it('shows no count while nothing is saved', async () => {
    useServices()
    await render('/saved')
    expect(tabs()[2]!.textContent?.trim()).toBe('Saved')
  })
})
