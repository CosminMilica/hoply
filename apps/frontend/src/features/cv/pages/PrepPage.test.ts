import { afterEach, describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { button, click, input, mountPage, settle, type, useServices } from '@/test/hoply'
import { sampleProfile } from '@/test/profile-service'
import PrepPage from './PrepPage.vue'
import { seedId } from '@/test/jobs-seed'

const text = () => document.body.textContent ?? ''
const card = (selector: string, title: string) =>
  [...document.querySelectorAll<HTMLElement>(selector)].find(element => element.querySelector('h3')?.textContent === title)!
const within = (root: HTMLElement, name: string) =>
  [...root.querySelectorAll<HTMLButtonElement>('button')].find(element => (element.getAttribute('aria-label') ?? element.textContent?.trim()) === name)!
const gaps = () => [...document.querySelectorAll('[data-test="gap"] h3')].map(element => element.textContent)

afterEach(() => {
  document.body.innerHTML = ''
  vi.useRealTimers()
})

describe('PrepPage', () => {
  it('saves gap answers in the profile, and the match follows them', async () => {
    const { profile } = useServices()
    await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    expect(text()).toContain('Snowflake Administrator DevOps · Capgemini Romania')
    expect(text()).toContain('0 of 8 questions answered')
    expect(document.querySelector('[data-test="match"]')!.textContent).toBe('56%')

    const terraform = () => card('[data-test="gap"]', 'Terraform')
    expect(within(terraform(), 'Add to my CV').disabled).toBe(true)
    await type(input('Your experience with Terraform'), 'Wrote modules for 3 environments')
    await click(within(terraform(), 'Bonbon Labs'))
    expect(within(terraform(), 'Bonbon Labs').getAttribute('aria-pressed')).toBe('true')
    await click(within(terraform(), 'Add to my CV'))
    expect(profile.experience[1]!.bullets.at(-1)).toEqual({ text: 'Wrote modules for 3 environments', result: '', skills: ['Terraform'] })
    expect(terraform().textContent).toContain('Saved in your profile under Junior system administrator at Bonbon Labs')
    expect(document.querySelector('[data-test="match"]')!.textContent).toBe('56% → 67%')
    await click(within(terraform(), 'Also add Terraform to my skills'))
    expect(profile.skills).toContain('Terraform')
    expect(terraform().textContent).toContain('In your profile skills')
    await click(within(terraform(), 'Change'))
    expect(profile.experience[1]!.bullets).toHaveLength(3)
    expect(terraform().textContent).toContain('Missing')

    const rbac = () => card('[data-test="gap"]', 'RBAC')
    await click(within(rbac(), 'Skip'))
    expect(rbac().textContent).toContain('Skipped')
    await click(within(rbac(), 'Change my mind'))
    expect(rbac().textContent).toContain('Missing')
  })

  it('confirms responsibilities in the person’s words, from their profile, or not at all', async () => {
    const person = sampleProfile()
    person.experience[0]!.bullets.push({ text: 'Kept Terraform state tidy', result: '', skills: [] })
    const { profile } = useServices(person)
    await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    const claim = (index: number) => [...document.querySelectorAll<HTMLElement>('[data-test="claim"]')][index]!

    expect(claim(3).textContent).toContain('From your profile')
    await click(within(claim(3), 'Verde Cloud: Kept Terraform state tidy'))
    expect(claim(3).textContent).toContain('From your profile')
    expect(claim(3).textContent).toContain('Kept Terraform state tidy')

    await click(within(claim(1), 'Start from the job’s wording, then make it yours'))
    expect((claim(1).querySelector('textarea') as HTMLTextAreaElement).value).toBe('Automated provisioning, configuration, and deployment processes through Infrastructure as Code and CI/CD pipelines')
    await click(within(claim(1), 'Elsewhere'))
    await click(within(claim(1), 'Add to my CV'))
    expect(profile.additional_info).toEqual(['Automated provisioning, configuration, and deployment processes through Infrastructure as Code and CI/CD pipelines'])
    expect(claim(1).textContent).toContain('Saved in your profile under Additional information')
    await click(within(claim(2), 'Not really'))
    expect(claim(2).textContent).toContain('Left out')
    await click(within(claim(2), 'Change'))
    expect(text()).toContain('2 of 8 questions answered')
  })

  it('remembers a fact for the next job, which does not ask about it again', async () => {
    useServices()
    const first = await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    expect(gaps()).toContain('Terraform')
    await type(input('Your experience with Terraform'), 'Wrote Terraform modules for 3 environments')
    await click(within(card('[data-test="gap"]', 'Terraform'), 'Add to my CV'))
    first.wrapper.unmount()
    document.body.innerHTML = ''

    await mountPage(PrepPage, { at: `/jobs/${seedId(2)}/prepare`, props: { id: seedId(2) } })
    expect(gaps()).toEqual(['Kubernetes', 'AWS'])
    expect(text()).toContain('Terraform')
    expect(document.querySelector('[data-test="match"]')!.textContent).toBe('75%')
  })

  it('saves results and extra lines in the profile, and keeps the CV’s choices', async () => {
    const { profile } = useServices()
    await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    const impacts = document.querySelectorAll<HTMLInputElement>('[aria-label="What changed because of it"]')
    expect(impacts).toHaveLength(3)
    await type(impacts[0]!, 'cut release time to 15 minutes')
    expect(profile.experience[0]!.bullets[0]!.result).toBe('cut release time to 15 minutes')
    expect(document.querySelectorAll('[aria-label="What changed because of it"]')).toHaveLength(3)
    await type(input('Anything else that makes you a better fit?'), 'Certified Kubernetes Administrator')
    expect(profile.additional_info).toEqual(['Certified Kubernetes Administrator'])

    const role = button(/^Junior system administrator/)
    await click(role)
    expect(role.getAttribute('aria-pressed')).toBe('false')
    await click(button('SQL'))
    await click(button('Python'))
    expect(button('Python').getAttribute('aria-pressed')).toBe('true')
  })

  it('generates the CV, steps through the progress and opens it', async () => {
    const { applications } = useServices()
    applications.track(seedId(1), 'saved')
    const { router } = await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    vi.useFakeTimers()
    button('Generate CV · 56% match').click()
    await vi.advanceTimersByTimeAsync(50)
    expect(text()).toContain('Tailoring your CV')
    await vi.advanceTimersByTimeAsync(1000)
    expect(text()).toContain('Keywords in the job description')
    await vi.advanceTimersByTimeAsync(2000)
    expect(text()).toContain('Rewritten for this job')
    await vi.advanceTimersByTimeAsync(1600)
    vi.useRealTimers()
    await settle()
    expect(router.currentRoute.value).toMatchObject({ name: 'hoply-cv', params: { id: seedId(1) } })
    expect(applications.applications.get(seedId(1))?.status).toBe('cv_ready')
  })

  it('waits on the service while the CV is still being written', async () => {
    const { applications } = useServices(undefined, { writing: 6 })
    applications.track(seedId(1), 'saved')
    const { router } = await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    vi.useFakeTimers()
    button('Generate CV · 56% match').click()
    await vi.advanceTimersByTimeAsync(4600)
    expect(text()).toContain('Tailoring your CV')
    expect(text()).not.toContain('Rewritten for this job')
    expect(router.currentRoute.value.name).not.toBe('hoply-cv')
    await vi.advanceTimersByTimeAsync(3000)
    vi.useRealTimers()
    await settle()
    expect(router.currentRoute.value).toMatchObject({ name: 'hoply-cv', params: { id: seedId(1) } })
    expect(applications.applications.get(seedId(1))?.status).toBe('cv_ready')
  })

  it('goes back to the form and says so when the CV cannot be written', async () => {
    const { applications } = useServices(undefined, { writing: 1, failing: true })
    applications.track(seedId(1), 'saved')
    await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    vi.useFakeTimers()
    button('Generate CV · 56% match').click()
    await vi.advanceTimersByTimeAsync(2500)
    vi.useRealTimers()
    await settle()
    expect(text()).toContain('Prepare your CV')
    expect(text()).toContain('We could not write this CV. Try again.')
    expect(applications.applications.get(seedId(1))?.status).toBe('saved')
  })

  it('keeps an applied job applied and goes back to the form when generation fails', async () => {
    const { applications } = useServices()
    applications.track(seedId(2), 'applied')
    await mountPage(PrepPage, { at: `/jobs/${seedId(2)}/prepare`, props: { id: seedId(2) } })
    server.use(http.post('*/api/cv/v1/cvs', () => HttpResponse.json({ message: 'busy' }, { status: 503 })))
    await click(button(/^Generate CV/))
    await settle()
    expect(text()).toContain('Prepare your CV')
    expect(applications.applications.get(seedId(2))?.status).toBe('applied')
  })

  it('says so when the profile cannot be read', async () => {
    useServices()
    server.use(http.get('*/api/profile/v1/profiles/me', () => HttpResponse.json({ message: 'down' }, { status: 503 })))
    await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    expect(text()).toContain('We could not prepare this CV')
  })
})
