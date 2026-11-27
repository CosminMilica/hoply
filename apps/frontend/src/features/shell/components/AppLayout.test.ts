import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createMemoryHistory, RouterView } from 'vue-router'
import { createApplicationRouter } from '@/shared/application/router'
import { testAuth } from '@/shared/application/testing'
import UApp from '@nuxt/ui/components/App.vue'
import { button, settle, useServices } from '@/test/hoply'
import AppLayout from './AppLayout.vue'
import { seedId } from '@/test/jobs-seed'

const Page = defineComponent({ props: { id: { type: String, default: '' } }, setup: props => () => h('p', { 'data-test': 'page' }, props.id) })
const narrowScreen = window.matchMedia

async function render(at: string) {
  const router = createApplicationRouter({
    auth: testAuth().auth,
    history: createMemoryHistory(),
    layouts: { app: AppLayout },
    routes: {
      app: [
        { path: 'profile', name: 'hoply-profile', component: Page, meta: { menu: 'profile' } },
        { path: 'jobs', name: 'hoply-jobs', component: Page, meta: { menu: 'jobs' } },
        { path: 'jobs/add', name: 'hoply-job-add', component: Page, meta: { menu: 'jobs' } },
        { path: 'jobs/:id/cv', name: 'hoply-cv', component: Page, props: true, meta: { menu: 'applications' } },
        { path: 'saved', name: 'hoply-saved', component: Page, meta: { menu: 'saved' } },
        { path: 'applications', name: 'hoply-applications', component: Page, meta: { menu: 'applications' } },
        { path: 'skills', name: 'hoply-skills', component: Page, meta: { menu: 'skills' } },
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

/** A screen from 900px wide: the header holds the menu. */
function wideScreen() {
  window.matchMedia = vi.fn().mockImplementation((query: string) => ({
    matches: query === '(min-width: 900px)', media: query, onchange: null,
    addListener: vi.fn(), removeListener: vi.fn(), addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn(),
  }))
}

const links = (shell: string) => [...document.querySelectorAll<HTMLAnchorElement>(`[data-shell="${shell}"] a`)]
const tabs = () => links('tabs')

afterEach(() => {
  window.matchMedia = narrowScreen
  document.body.innerHTML = ''
})

describe('AppLayout', () => {
  it('shows the primary menu as bottom tabs on a phone, with Add a job and the avatar in the header', async () => {
    useServices()
    await render('/profile')
    expect(document.querySelector('[data-test="page"]')).toBeTruthy()
    expect(tabs().map(tab => tab.textContent?.trim())).toEqual(['Find jobs', 'Saved', 'Applications'])
    expect(tabs().some(tab => tab.getAttribute('aria-current'))).toBe(false)
    expect(tabs()[0]!.getAttribute('href')).toBe('/jobs')
    expect(document.querySelector('[data-shell="top-nav"]')).toBeNull()
    expect(document.querySelector('[data-shell="header"] a')?.getAttribute('aria-label')).toBe('hoply, go to your profile')
    expect(button('Add a job').getAttribute('href')).toBe('/jobs/add')
    expect(document.querySelector('[data-test="account-avatar"]')!.textContent?.trim()).toBe('AP')
  })

  it('counts saved jobs and applications apart, and marks Applications on a CV', async () => {
    const { applications } = useServices()
    applications.track(seedId(1), 'saved')
    applications.track(seedId(2), 'cv_ready')
    applications.track(seedId(3), 'applied')
    await render(`/jobs/${seedId(2)}/cv`)
    const [, saved, sent] = tabs()
    expect(sent!.getAttribute('aria-current')).toBe('page')
    expect(saved!.textContent).toContain('1 saved')
    expect(sent!.textContent).toContain('2 applications')
  })

  it('shows no count while nothing is tracked, and adds a job to Saved from Saved', async () => {
    useServices()
    await render('/saved')
    expect(tabs().map(tab => tab.textContent?.trim())).toEqual(['Find jobs', 'Saved', 'Applications'])
    expect(button('Add a job').getAttribute('href')).toBe('/jobs/add?from=saved')
  })

  it('puts the menu in the header on a wide screen', async () => {
    wideScreen()
    const { applications } = useServices()
    applications.track(seedId(1), 'saved')
    await render('/saved')
    const top = links('top-nav')
    expect(top.map(tab => tab.textContent?.replace(/\s+/g, ' ').trim())).toEqual(['Find jobs', 'Saved 1 saved', 'Applications'])
    expect(top[1]!.getAttribute('aria-current')).toBe('page')
    expect(button('Add a job').textContent?.trim()).toBe('Add a job')
  })

  it('opens the account menu from the avatar: who you are, how complete the profile is, and its links', async () => {
    useServices()
    const router = await render('/jobs')
    const avatar = button('Your profile and menu')
    avatar.focus()
    avatar.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    await settle()
    const menu = document.querySelector('[role="menu"]')!
    expect(menu.textContent).toContain('Andrei Popescu')
    expect(menu.textContent).toContain('Profile complete')
    expect(document.querySelector('[data-test="account-readiness"]')!.textContent).toBe('100%')
    const items = [...menu.querySelectorAll('a')]
    expect(items.map(item => [item.textContent?.trim(), item.getAttribute('href')])).toEqual([['Your profile', '/profile'], ['Skill gaps', '/skills']])
    items[1]!.click()
    await settle()
    expect(router.currentRoute.value.name).toBe('hoply-skills')
  })
})
