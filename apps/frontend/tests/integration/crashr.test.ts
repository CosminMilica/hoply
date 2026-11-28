import { afterEach, beforeAll, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import type { Router } from 'vue-router'
import App from '../../src/app/App.vue'
import { auth } from '../../src/app/auth'
import { button, click, settle, useServices } from '../../src/test/crashr'

// The app as it ships: its root, its router and crashr's pages, against the services in memory.
let router: Router
let wrapper: VueWrapper | undefined

beforeAll(async () => {
  vi.stubEnv('VITE_AUTH_DISABLED', 'true')
  await auth.init?.()
  router = (await import('../../src/app/router')).default
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
})

async function open(path: string) {
  await router.push(path)
  await router.isReady()
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  wrapper = mount(App, { global: { plugins: [router, createPinia(), [VueQueryPlugin, { queryClient }]] }, attachTo: document.body })
  await flushPromises()
  await settle(8)
}

const text = () => document.body.textContent ?? ''

it('starts on the profile, read from the profile service', async () => {
  useServices()
  await open('/')
  expect(router.currentRoute.value.name).toBe('crashr-profile')
  expect(text()).toContain('Ready to tailor')
  expect(text()).toContain('100%')
  expect(text()).toContain('Andrei Popescu')
})

it('opens Find jobs on the grid with its filters, which also leads to adding a job', async () => {
  useServices()
  await open('/jobs')
  await vi.waitFor(() => expect(document.querySelector('[data-test="filter-bar"]')).toBeTruthy())
  await click(button('Add a job'))
  await vi.waitFor(() => expect(router.currentRoute.value.name).toBe('crashr-job-add'))
})

it('adds a job from a link and lists it under Saved', async () => {
  const { jobs } = useServices()
  await open('/jobs/add?from=saved')
  const field = [...document.querySelectorAll('label')].find(label => label.textContent?.trim() === 'Link to the job ad')!
  const url = document.getElementById(field.htmlFor) as HTMLInputElement
  url.value = 'https://careers.northwind.example/jobs/platform-engineer'
  url.dispatchEvent(new Event('input', { bubbles: true }))
  await click(button('Read the job'))
  await settle(8)
  expect(text()).toContain('Check what we found')
  await click(button('Save to my jobs'))
  await settle(8)
  await vi.waitFor(() => expect(router.currentRoute.value.name).toBe('crashr-saved'))
  await settle(4)
  expect(text()).toContain('Platform Engineer')
  expect(jobs.jobs[0]).toMatchObject({ title: 'Platform Engineer', added_by_me: true })
})
