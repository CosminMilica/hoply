import { afterEach, describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { setLocale } from '@/features/common/i18n'
import { button, click, mountPage, useServices } from '@/test/crashr'
import ApplicationsPage from './ApplicationsPage.vue'
import { seedId } from '@/test/jobs-seed'

const text = () => document.body.textContent ?? ''
const stage = (id: string) => document.querySelector<HTMLElement>(`[data-test="stage-${id}"]`)!
const titles = (id: string) => [...stage(id).querySelectorAll('h3')].map(title => title.textContent)

afterEach(() => {
  document.body.innerHTML = ''
})

describe('ApplicationsPage', () => {
  it('shows the jobs with a CV by stage, and marks one as applied', async () => {
    const { applications } = useServices()
    applications.track(seedId(1), 'saved')
    applications.track(seedId(2), 'cv_ready')
    applications.track(seedId(3), 'applied')
    const { router } = await mountPage(ApplicationsPage)
    expect(titles('cv_ready')).toHaveLength(1)
    expect(titles('applied')).toHaveLength(1)
    expect(text()).not.toContain('Snowflake Administrator DevOps')
    expect([...document.querySelectorAll('[data-test="since"]')].map(line => line.textContent)).toEqual(['CV ready 25 Nov', 'Applied 25 Nov'])
    expect(button('CV ready 1').getAttribute('aria-pressed')).toBe('true')
    expect(stage('applied').className).toContain('hidden')

    await click(button('Applied 1'))
    expect(stage('applied').className).not.toContain(' hidden')
    expect(stage('cv_ready').className).toContain('hidden')

    await click(button('I applied'))
    expect(applications.applications.get(seedId(2))?.status).toBe('applied')
    await vi.waitFor(() => expect(titles('applied')).toHaveLength(2))
    expect(stage('cv_ready').textContent).toContain('No CV waiting to be sent.')

    await click(button('View CV'))
    expect(router.currentRoute.value.name).toBe('crashr-cv')
  })

  it('points to Saved while nothing has a CV yet', async () => {
    const { applications } = useServices()
    applications.track(seedId(1), 'saved')
    await mountPage(ApplicationsPage)
    expect(text()).toContain('No applications yet. Prepare a CV for a saved job and it shows up here.')
    expect(button('Go to saved jobs').getAttribute('href')).toBe('/saved')
    expect(text()).toContain('Applications')
  })

  it('speaks Romanian, and switches language live', async () => {
    const { applications } = useServices()
    applications.track(seedId(2), 'cv_ready')
    applications.track(seedId(3), 'applied')
    await mountPage(ApplicationsPage)
    setLocale('ro')
    await vi.waitFor(() => expect(text()).toContain('Aplicări'))
    expect(text()).toContain('Joburile cu CV adaptat, de la pregătit la trimis.')
    expect([...document.querySelectorAll('[data-test="since"]')].map(line => line.textContent)).toEqual(['CV gata pe 25 nov.', 'Aplicat pe 25 nov.'])
    expect(button('CV gata 1').getAttribute('aria-pressed')).toBe('true')
    expect(button('Am aplicat')).toBeTruthy()
    expect(button('Vezi CV-ul')).toBeTruthy()
    expect(stage('applied').getAttribute('aria-label')).toBe('Aplicat')
    setLocale('en')
    await vi.waitFor(() => expect(button('I applied')).toBeTruthy())
  })

  it('points to Saved in Romanian while nothing has a CV yet', async () => {
    setLocale('ro')
    useServices()
    await mountPage(ApplicationsPage)
    expect(text()).toContain('Încă nu ai aplicări. Pregătește un CV pentru un job salvat și apare aici.')
    expect(button('Mergi la joburile salvate').getAttribute('href')).toBe('/saved')
  })

  it('says when the applications cannot be loaded', async () => {
    const { applications } = useServices()
    applications.track(seedId(2), 'cv_ready')
    server.use(http.get('*/api/jobs/v1/jobs', () => HttpResponse.json({ message: 'down' }, { status: 503 })))
    await mountPage(ApplicationsPage)
    await vi.waitFor(() => expect(text()).toContain('We could not load your applications'))
  })
})
