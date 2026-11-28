import { afterEach, describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { setLocale } from '@/features/common/i18n'
import { button, click, mountPage, useServices } from '@/test/crashr'
import SavedPage from './SavedPage.vue'
import { seedId } from '@/test/jobs-seed'

const text = () => document.body.textContent ?? ''
const titles = () => [...document.querySelectorAll('[data-test="saved-job"] h2')].map(title => title.textContent)

afterEach(() => {
  document.body.innerHTML = ''
})

describe('SavedPage', () => {
  it('lists only the saved jobs, best match first or newest first, with what each needs', async () => {
    const { applications } = useServices()
    applications.track(seedId(1), 'saved')
    applications.track(seedId(4), 'saved')
    applications.track(seedId(2), 'cv_ready')
    applications.track(seedId(3), 'applied')
    await mountPage(SavedPage)
    expect(button('Best match').getAttribute('aria-pressed')).toBe('true')
    const byMatch = titles()
    expect(byMatch).toHaveLength(2)
    expect([...document.querySelectorAll('[data-test="since"]')].map(line => line.textContent)).toEqual(['Saved 25 Nov', 'Saved 25 Nov'])
    expect(text()).toContain('Missing: Snowflake, Terraform, Infrastructure as Code, RBAC')
    expect(text()).not.toContain('Skill gaps')

    await click(button('Newest'))
    expect(button('Newest').getAttribute('aria-pressed')).toBe('true')
    expect(titles()).toEqual([...byMatch].reverse())
  })

  it('moves a saved job whose CV got ready in the background on to Applications', async () => {
    const { applications, cvs } = useServices()
    applications.track(seedId(1), 'saved')
    applications.track(seedId(4), 'saved')
    cvs.create({ job_id: seedId(1), skip_roles: [], focus: [], confirmed: [] })
    await mountPage(SavedPage)
    await vi.waitFor(() => expect(applications.applications.get(seedId(1))?.status).toBe('cv_ready'))
    expect(applications.applications.get(seedId(4))?.status).toBe('saved')
    await vi.waitFor(() => expect(titles()).toHaveLength(1))
  })

  it('says when the saved jobs cannot be loaded, and asks for them 200 at a time', async () => {
    const { applications } = useServices()
    applications.track(seedId(1), 'saved')
    for (let i = 0; i < 204; i += 1) applications.track(`44444444-0000-4000-8000-${String(i).padStart(12, '0')}`, 'saved')
    const asked: number[] = []
    server.use(http.get('*/api/jobs/v1/jobs', ({ request }) => {
      asked.push((new URL(request.url).searchParams.get('ids') ?? '').split(',').length)
      return HttpResponse.json({ message: 'down' }, { status: 503 })
    }))
    await mountPage(SavedPage)
    await vi.waitFor(() => expect(text()).toContain('We could not load your saved jobs'))
    expect(asked.slice(0, 2).sort((a, b) => b - a)).toEqual([200, 5])
    expect(text()).not.toContain('Nothing saved yet')
  })

  it('prepares a CV for a saved job, or removes it', async () => {
    const { applications } = useServices()
    applications.track(seedId(4), 'saved')
    const { router } = await mountPage(SavedPage)
    expect(text()).toContain('Missing: Windows Server')
    expect(() => button('Best match')).toThrow()
    await click(button('Prepare CV'))
    expect(router.currentRoute.value).toMatchObject({ name: 'crashr-prep', params: { id: seedId(4) } })
    await click(button('Remove from saved'))
    expect(applications.applications.size).toBe(0)
    expect(text()).toContain('Nothing saved yet. Add a job you found, from its link or its text, and it lands here.')
    expect(button('Add a job').getAttribute('href')).toBe('/jobs/add?from=saved')
  })

  it('speaks Romanian', async () => {
    setLocale('ro')
    const { applications } = useServices()
    applications.track(seedId(1), 'saved')
    applications.track(seedId(4), 'saved')
    await mountPage(SavedPage)
    expect(text()).toContain('Joburi salvate')
    expect(button('Cele mai potrivite').getAttribute('aria-pressed')).toBe('true')
    expect(button('Cele mai noi')).toBeTruthy()
    expect([...document.querySelectorAll('[data-test="since"]')].map(line => line.textContent)).toEqual(['Salvat pe 25 nov.', 'Salvat pe 25 nov.'])
    expect(text()).toContain('Lipsă: Snowflake, Terraform, Infrastructure as Code, RBAC')
    expect(button('Pregătește CV-ul')).toBeTruthy()
    expect(button('Scoate din salvate')).toBeTruthy()
  })

  it('says in Romanian when nothing is saved, or every saved job has a CV', async () => {
    setLocale('ro')
    useServices()
    await mountPage(SavedPage)
    expect(text()).toContain('Încă n-ai salvat nimic.')
    expect(button('Adaugă un job').getAttribute('href')).toBe('/jobs/add?from=saved')
    document.body.innerHTML = ''

    const { applications } = useServices()
    applications.track(seedId(2), 'applied')
    await mountPage(SavedPage)
    expect(text()).toContain('Toate joburile salvate au acum CV.')
  })

  it('says every saved job has a CV once they all moved on', async () => {
    const { applications } = useServices()
    applications.track(seedId(2), 'applied')
    await mountPage(SavedPage)
    expect(text()).toContain('Every saved job has a CV now.')
  })
})
