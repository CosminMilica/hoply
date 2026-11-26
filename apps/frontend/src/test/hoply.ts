import { defineComponent, h, type Component } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createMemoryHistory, createRouter, type RouteRecordRaw } from 'vue-router'
import UApp from '@nuxt/ui/components/App.vue'
import { server } from '@/shared/mocks/node'
import { applicationsServiceHandlers } from './applications-service'
import { cvServiceHandlers } from './cv-service'
import { jobsServiceHandlers } from './jobs-service'
import type { Profile } from '@/features/profile/api/profile.schema'
import { profileServiceHandlers, sampleProfile } from './profile-service'

const Blank = defineComponent({ name: 'BlankRoute', setup: () => () => h('div', { 'data-test': 'route' }) })

export const ROUTE_PATHS: Record<string, string> = {
  'hoply-profile': '/profile',
  'hoply-jobs': '/jobs',
  'hoply-job-add': '/jobs/add',
  'hoply-job': '/jobs/:id',
  'hoply-prep': '/jobs/:id/prepare',
  'hoply-cv': '/jobs/:id/cv',
  'hoply-saved': '/saved',
  'hoply-skills': '/saved/skills',
}

export function testRouter(pages: Partial<Record<string, Component>> = {}) {
  const routes: RouteRecordRaw[] = Object.entries(ROUTE_PATHS).map(([name, path]) => ({ name, path, component: pages[name] ?? Blank, props: true }))
  routes.push({ path: '/:any(.*)*', component: Blank })
  return createRouter({ history: createMemoryHistory(), routes })
}

/** Every service the app talks to, in memory; the cv service reads the profile and the jobs from the other two. */
export function useServices(profile: Profile = sampleProfile(), cvOptions: Parameters<typeof cvServiceHandlers>[1] = {}) {
  const profileService = profileServiceHandlers(profile)
  const jobsService = jobsServiceHandlers()
  const applicationsService = applicationsServiceHandlers()
  const cvService = cvServiceHandlers({
    profile: () => profileService.profile,
    job: id => jobsService.backend.jobs.find(item => item.id === id),
  }, cvOptions)
  server.use(...profileService.handlers, ...jobsService.handlers, ...applicationsService.handlers, ...cvService.handlers)
  return { profile: profileService.profile, applications: applicationsService.backend, jobs: jobsService.backend, cvs: cvService.backend }
}

interface MountOptions {
  at?: string
  props?: Record<string, unknown>
  pages?: Partial<Record<string, Component>>
}

/** A page as the app renders it: inside UApp, with a router, Pinia and a query client of its own. */
export async function mountPage(component: Component, { at = '/', props = {}, pages }: MountOptions = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 0 }, mutations: { retry: false } } })
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = testRouter(pages)
  await router.push(at)
  await router.isReady()
  const Root = defineComponent({ setup: () => () => h(UApp, null, () => h(component, props)) })
  const wrapper = mount(Root, { global: { plugins: [router, pinia, [VueQueryPlugin, { queryClient }]] }, attachTo: document.body })
  await settle()
  return { wrapper, router, queryClient }
}

/** Lets requests, their answers and the renders after them finish. */
export async function settle(rounds = 4) {
  for (let i = 0; i < rounds; i += 1) {
    await flushPromises()
    await new Promise(resolve => setTimeout(resolve, 0))
  }
}

/** The button whose accessible text or label is `name`. */
export function button(name: string | RegExp): HTMLButtonElement {
  const candidates = [...document.querySelectorAll<HTMLButtonElement>('button, a')]
  const match = candidates.find((element) => {
    const label = element.getAttribute('aria-label') ?? element.textContent?.trim() ?? ''
    return typeof name === 'string' ? label === name : name.test(label)
  })
  if (!match) throw new Error(`No button named ${String(name)}`)
  return match
}

export function input(label: string): HTMLInputElement | HTMLTextAreaElement {
  const byAria = document.querySelector<HTMLInputElement>(`[aria-label="${label}"]`)
  if (byAria) return byAria
  const labelElement = [...document.querySelectorAll('label')].find(element => element.textContent?.trim() === label)
  const control = labelElement?.htmlFor ? document.getElementById(labelElement.htmlFor) : null
  if (!control) throw new Error(`No field labelled ${label}`)
  return control as HTMLInputElement
}

export async function type(field: HTMLInputElement | HTMLTextAreaElement, value: string) {
  field.value = value
  field.dispatchEvent(new Event('input', { bubbles: true }))
  field.dispatchEvent(new Event('change', { bubbles: true }))
  await settle()
}

export async function click(element: HTMLElement) {
  element.click()
  await settle()
}

/** Picks `option` in the Nuxt UI select labelled `label`, with the keyboard as jsdom has no pointer capture. */
export async function choose(label: string, option: string) {
  const trigger = input(label) as unknown as HTMLElement
  trigger.focus()
  trigger.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
  await settle()
  const item = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find(element => element.textContent?.trim() === option)
  if (!item) throw new Error(`No option ${option} in ${label}`)
  item.focus()
  item.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
  await settle()
}

export const sectionToggle = (title: string) =>
  [...document.querySelectorAll('h2')].find(element => element.textContent === title)!.closest('button')!
