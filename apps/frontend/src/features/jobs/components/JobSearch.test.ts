import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { button, click, input, mountPage, settle, type, useServices as useAllServices } from '@/test/hoply'
import { sampleProfile } from '@/test/profile-service'
import { SEARCH_DELAY_MS } from '../composables/useDeck'
import { useJobSearchStore } from '../stores/search'
import JobSearch from './JobSearch.vue'
import { seedId } from '@/test/jobs-seed'

const text = () => document.body.textContent ?? ''
/** A person without work preferences, so the filters start empty. */
const useServices = () => useAllServices(sampleProfile({ city: '', work_modes: [], target_seniority: [] }))
const cardTitle = () => document.querySelector('[data-test="job-card"] h2')?.textContent
const wait = async (ms: number) => {
  await new Promise(resolve => setTimeout(resolve, ms))
  await settle()
}

/** Shows the jobs, then the swipe deck. */
const showDeck = async (label: string) => {
  await click(button(label))
  await click(button('Swipe'))
}
const rows = () => [...document.querySelectorAll('[data-test="job-tile"]')].map(row => row.getAttribute('aria-label'))

afterEach(() => {
  document.body.innerHTML = ''
  localStorage.clear()
  vi.useRealTimers()
})

describe('JobSearch in swipe view', () => {
  beforeEach(() => localStorage.setItem('hoply.jobs.view', 'swipe'))

  it('filters the jobs, then shows the deck best match first', async () => {
    useServices()
    await mountPage(JobSearch)
    expect(button('Show 4 jobs')).toBeTruthy()
    await click(button('Bucharest'))
    expect(button('Bucharest').getAttribute('aria-pressed')).toBe('true')
    expect(button('Show 2 jobs')).toBeTruthy()
    await click(button('Backend'))
    await click(button('Senior'))
    expect(button('No jobs match yet').disabled).toBe(true)
    await click(button('Reset filters'))
    await type(input('What'), 'devops')
    await wait(SEARCH_DELAY_MS + 50)
    await showDeck('Show 2 jobs')
    expect(text()).toContain('“devops” · All areas · Anywhere')
    expect(text()).toContain('2 jobs left')
    expect(cardTitle()).toBe('DevOps Engineer')
    await click(button('Filters'))
    expect(text()).toContain('Position area')
  })

  it('shows a job a company posts in several cities once, and leaves it once one is decided', async () => {
    const { applications, jobs } = useServices()
    const devops = jobs.jobs.find(job => job.id === seedId(2))!
    jobs.jobs.push({ ...structuredClone(devops), id: '22222222-0000-4000-8000-000000000001', city: 'Iași' })
    await mountPage(JobSearch)
    await showDeck('Show 4 jobs')
    expect(text()).toContain('4 jobs left')
    await click(button('Save this job'))
    await wait(260)
    expect(applications.applications.has(seedId(2)) || applications.applications.has('22222222-0000-4000-8000-000000000001')).toBe(true)
    expect(text()).toContain('3 jobs left')
    expect(cardTitle()).not.toBe('DevOps Engineer')
  })

  it('saves, skips and undoes, until everything is seen', async () => {
    const { applications, jobs, cvs } = useServices()
    jobs.jobs.find(job => job.id === seedId(2))!.source = 'import'
    await mountPage(JobSearch)
    await showDeck('Show 4 jobs')
    expect(cardTitle()).toBe('DevOps Engineer')

    await click(button('Save this job'))
    await wait(260)
    expect(applications.applications.get(seedId(2))?.status).toBe('saved')
    expect(cvs.analyses).toEqual([seedId(2)])
    expect(text()).toContain('Saved · DevOps Engineer')
    await click(button('Undo'))
    expect(applications.applications.has(seedId(2))).toBe(false)
    expect(cardTitle()).toBe('DevOps Engineer')

    await click(button('Skip this job'))
    await wait(260)
    expect(text()).toContain('Skipped · DevOps Engineer')
    await click(button('Undo'))
    expect(applications.skips.length).toBe(0)

    for (let i = 0; i < 4; i += 1) {
      await click(button('Skip this job'))
      await wait(260)
    }
    expect(text()).toContain('You are all caught up')
    await click(button('Bring back skipped jobs'))
    expect(applications.skips.length).toBe(0)
    expect(cardTitle()).toBeTruthy()
  })

  it('hides the undo after a while', async () => {
    useServices()
    await mountPage(JobSearch)
    await showDeck('Show 4 jobs')
    vi.useFakeTimers()
    button('Save this job').click()
    await vi.advanceTimersByTimeAsync(300)
    expect(text()).toContain('Saved · DevOps Engineer')
    await vi.advanceTimersByTimeAsync(3600)
    vi.useRealTimers()
    await settle()
    expect(text()).not.toContain('Saved · DevOps Engineer')
  })

  it('opens Prepare your CV on Tailor CV, and the job on Details', async () => {
    const { applications } = useServices()
    const { router } = await mountPage(JobSearch)
    await showDeck('Show 4 jobs')
    await click(button('Details'))
    expect(router.currentRoute.value).toMatchObject({ name: 'hoply-job', params: { id: seedId(2) } })
    await router.push('/jobs')
    await settle()
    await click(button('Tailor CV'))
    await wait(260)
    expect(router.currentRoute.value).toMatchObject({ name: 'hoply-prep', params: { id: seedId(2) } })
    expect(applications.applications.get(seedId(2))?.status).toBe('saved')
  })

  it('reacts to swipes and taps on the card', async () => {
    useServices()
    const { router } = await mountPage(JobSearch)
    await showDeck('Show 4 jobs')
    const card = () => document.querySelector<HTMLElement>('[data-test="job-card"]')!
    const pointer = (type: string, x: number, y: number) => {
      const event = new MouseEvent(type, { bubbles: true, clientX: x, clientY: y }) as unknown as PointerEvent
      card().dispatchEvent(event)
    }
    const swipe = async (dx: number, dy: number) => {
      pointer('pointerdown', 200, 300)
      pointer('pointermove', 200 + dx, 300 + dy)
      await settle(1)
      pointer('pointerup', 200 + dx, 300 + dy)
      await wait(260)
    }
    await swipe(30, 0)
    expect(cardTitle()).toBe('DevOps Engineer')
    await swipe(150, 0)
    expect(text()).toContain('Saved · DevOps Engineer')
    await swipe(-150, 0)
    expect(text()).toContain('Skipped · ')
    pointer('pointermove', 0, 0)
    pointer('pointerup', 0, 0)
    pointer('pointerdown', 200, 300)
    pointer('pointerup', 200, 300)
    await settle()
    expect(router.currentRoute.value.name).toBe('hoply-job')
    await router.push('/jobs')
    await settle()
    await swipe(0, -200)
    expect(router.currentRoute.value.name).toBe('hoply-prep')
  })

  it('fills the filters from the person’s preferences, once', async () => {
    useAllServices(sampleProfile({ city: 'Bucharest', work_modes: ['hybrid', 'onsite'], target_seniority: ['mid'] }))
    await mountPage(JobSearch)
    expect(document.querySelector<HTMLInputElement>('input[placeholder="City or country"]')!.value).toBe('Bucharest')
    expect(button('Include remote').getAttribute('aria-pressed')).toBe('false')
    expect(button('Mid').getAttribute('aria-pressed')).toBe('true')
    expect(button('Show 2 jobs')).toBeTruthy()
    await click(button('Reset filters'))
    await click(button('Show 4 jobs'))
    await click(button('Filters'))
    expect(button('Show 4 jobs')).toBeTruthy()
  })

  it('leaves a level the filters do not have, and several seniorities, alone', async () => {
    useAllServices(sampleProfile({ city: '', work_modes: ['remote'], target_seniority: ['lead'] }))
    await mountPage(JobSearch)
    expect(button('Any').getAttribute('aria-pressed')).toBe('true')
    expect([...document.querySelectorAll('button')].some(item => item.textContent?.trim() === 'Include remote')).toBe(false)
    await click(button('Iași'))
    expect(button('Include remote').getAttribute('aria-pressed')).toBe('true')
    await click(button('Include remote'))
    expect(button('Include remote').getAttribute('aria-pressed')).toBe('false')
  })

  it('reminds the person of their market, searches their country, and offers to add a job', async () => {
    useAllServices(sampleProfile({ city: '', country: 'Romania', work_modes: ['hybrid'], target_seniority: [] }))
    await mountPage(JobSearch)
    expect(document.querySelector('[data-test="market-note"]')!.textContent).toContain('From your profile: Romania · hybrid · not relocating. Change it in Profile.')
    expect(button('Anywhere in Romania').getAttribute('aria-pressed')).toBe('true')
    expect(button(/^Add a job from a link or its text/).getAttribute('href')).toBe('/jobs/add')
    await click(button('Show 4 jobs'))
    expect(text()).toContain('All areas · Anywhere in Romania')
    expect(() => button('Add a job')).toThrow()
  })

  it('searches anywhere without a country, and finds nothing in another one', async () => {
    useAllServices(sampleProfile({ city: '', country: 'Moldova', work_modes: [], target_seniority: [] }))
    await mountPage(JobSearch)
    expect(document.querySelector('[data-test="market-note"]')!.textContent).toContain('Moldova')
    expect(button('No jobs match yet').disabled).toBe(true)
  })

  it('says so when jobs cannot be loaded', async () => {
    useServices()
    server.use(http.get('*/api/jobs/v1/jobs', () => HttpResponse.json({ message: 'down' }, { status: 503 })))
    await mountPage(JobSearch)
    useJobSearchStore().stage = 'deck'
    await settle()
    expect(text()).toContain('We could not load jobs')
  })
})

describe('JobSearch in grid view', () => {
  it('shows the jobs as a grid by default, best match first, a page at a time, and remembers the view', async () => {
    const { jobs, applications } = useServices()
    const devops = jobs.jobs.find(job => job.id === seedId(2))!
    for (let i = 0; i < 26; i += 1) jobs.jobs.push({ ...structuredClone(devops), id: `33333333-0000-4000-8000-${String(i).padStart(12, '0')}`, title: `Platform Engineer ${i}` })
    const { router } = await mountPage(JobSearch)
    await settle()
    expect(button('Grid').getAttribute('aria-pressed')).toBe('true')
    expect(document.querySelector('[data-test="job-card"]')).toBeNull()
    expect(rows()).toHaveLength(24)
    expect(rows()[0]).toBe('DevOps Engineer at Bitdefender')
    await click(button('Show 6 more'))
    expect(rows()).toHaveLength(30)

    await click(button('Save'))
    expect(applications.applications.get(seedId(2))?.status).toBe('saved')
    expect(text()).toContain('Saved · DevOps Engineer')
    expect(rows()).toHaveLength(29)
    await click(button('Undo'))
    expect(applications.applications.has(seedId(2))).toBe(false)
    expect(rows()[0]).toBe('DevOps Engineer at Bitdefender')

    await click(button('Skip'))
    expect(text()).toContain('Skipped · DevOps Engineer')
    expect(rows()).not.toContain('DevOps Engineer at Bitdefender')
    await click(document.querySelector<HTMLButtonElement>('[data-test="job-tile"] button')!)
    expect(router.currentRoute.value.name).toBe('hoply-job')
    await router.push('/jobs')
    await settle()
    await click(button('Details'))
    expect(router.currentRoute.value.name).toBe('hoply-job')
    await router.push('/jobs')
    await settle()
    await click(button('Tailor CV'))
    expect(router.currentRoute.value.name).toBe('hoply-prep')

    await router.push('/jobs')
    await settle()
    await click(button('Swipe'))
    expect(localStorage.getItem('hoply.jobs.view')).toBe('swipe')
    expect(cardTitle()).toBeTruthy()
  })

  it('shows a view remembered from before the grid as the grid', async () => {
    useServices()
    localStorage.setItem('hoply.jobs.view', 'list')
    await mountPage(JobSearch)
    await settle()
    expect(button('Grid').getAttribute('aria-pressed')).toBe('true')
    expect(rows()).toHaveLength(4)
  })

  it('says when the grid is all seen, and brings skipped jobs back', async () => {
    const { applications } = useServices()
    for (const n of [1, 2, 3, 4] as const) applications.skips.push(seedId(n))
    await mountPage(JobSearch)
    await settle()
    expect(text()).toContain('You are all caught up')
    const store = useJobSearchStore()
    store.query = 'devops'
    await click(button('Reset filters'))
    expect(store.query).toBe('')
    await click(button('Bring back skipped jobs'))
    expect(rows()).toHaveLength(4)
  })

  it('presses the chip of the profile’s city, whatever its spelling', async () => {
    useAllServices(sampleProfile({ city: 'București', work_modes: [], target_seniority: [] }))
    await mountPage(JobSearch)
    await settle()
    expect(button('Bucharest').getAttribute('aria-pressed')).toBe('true')
    expect(button('Iași').getAttribute('aria-pressed')).toBe('false')
  })

  it('says when the service sent only the newest of the jobs found', async () => {
    const { jobs } = useServices()
    server.use(http.get('*/api/jobs/v1/jobs', ({ request }) => {
      const answer = jobs.list(new URL(request.url).searchParams).body as { items: unknown[] }
      return HttpResponse.json({ ...answer, total: 900 })
    }))
    await mountPage(JobSearch)
    await settle()
    expect(text()).toContain('4 jobs left, from the newest 4 of 900: narrow the filters to see the rest')
  })

  it('filters from the bar above the grid, and offers to add a job', async () => {
    useServices()
    await mountPage(JobSearch)
    await settle()
    expect(document.querySelector('[data-test="filter-bar"]')).toBeTruthy()
    expect(rows()).toHaveLength(4)
    await click(button('Bucharest'))
    expect(rows()).toHaveLength(2)
    expect(button('Include remote').getAttribute('aria-pressed')).toBe('true')
    await click(button('Include remote'))
    await click(button('Backend'))
    expect(rows()).toEqual(['Software Engineer at Amazon'])
    await click(button('Senior'))
    expect(text()).toContain('You are all caught up')
    await click(button('Reset filters'))
    await type(document.querySelector<HTMLInputElement>('input[placeholder="Job title, skill or company"]')!, 'devops')
    expect(rows()).toHaveLength(4)
    await wait(SEARCH_DELAY_MS + 50)
    expect(rows()).toHaveLength(2)
    expect(() => button('Filters')).toThrow()
  })
})
