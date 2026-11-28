import { afterEach, describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { setLocale } from '@/features/common/i18n'
import { button, click, input, mountPage, settle, type, useServices } from '@/test/crashr'
import { emptyProfile, sampleProfile } from '@/test/profile-service'
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
    expect(profile.experience[1]!.notes).toEqual([{ text: 'Wrote modules for 3 environments', skills: ['Terraform'] }])
    expect(terraform().textContent).toContain('Saved in your profile under Junior system administrator at Bonbon Labs')
    expect(document.querySelector('[data-test="match"]')!.textContent).toBe('56% → 78%')
    await click(within(terraform(), 'Also add Terraform to my skills'))
    expect(profile.skills).toContain('Terraform')
    expect(terraform().textContent).toContain('In your profile skills')
    await click(within(terraform(), 'Change'))
    expect(profile.experience[1]!.notes).toEqual([])
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
    expect(profile.notes).toEqual([{
      text: 'Automated provisioning, configuration, and deployment processes through Infrastructure as Code and CI/CD pipelines',
      skills: ['Infrastructure as Code', 'CI/CD'],
    }])
    expect(claim(1).textContent).toContain('Saved in your profile under Your notes')
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

  it('writes the CV in the ad’s language unless the person picks the other one', async () => {
    const { applications } = useServices()
    applications.track(seedId(1), 'saved')
    const sent: unknown[] = []
    const record = ({ request }: { request: Request }) => {
      if (request.method === 'POST' && new URL(request.url).pathname.endsWith('/api/cv/v1/cvs')) void request.clone().json().then(body => sent.push(body))
    }
    server.events.on('request:start', record)
    await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    expect(button('English').getAttribute('aria-pressed')).toBe('true')
    await click(button('Română'))
    expect(button('Română').getAttribute('aria-pressed')).toBe('true')
    await click(button(/^Generate CV/))
    await vi.waitFor(() => expect(sent).toHaveLength(1))
    server.events.removeListener('request:start', record)
    expect(sent[0]).toMatchObject({ job_id: seedId(1), language: 'ro' })
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
    expect(router.currentRoute.value).toMatchObject({ name: 'crashr-cv', params: { id: seedId(1) } })
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
    expect(router.currentRoute.value.name).not.toBe('crashr-cv')
    await vi.advanceTimersByTimeAsync(3000)
    vi.useRealTimers()
    await settle()
    expect(router.currentRoute.value).toMatchObject({ name: 'crashr-cv', params: { id: seedId(1) } })
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

  it('shows the close equivalents apart from the gaps, and takes an answer for them', async () => {
    const person = sampleProfile()
    person.experience[0]!.tools.push('FluxCD')
    const { jobs, profile } = useServices(person)
    jobs.jobs[0]!.must_have.push('Argo CD')
    await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    const close = () => card('[data-test="close"]', 'Argo CD')
    expect(close().textContent).toContain('You use FluxCD, a GitOps tool like Argo CD. Your CV shows your GitOps experience, without naming Argo CD.')
    expect(close().textContent).toContain('Close')
    expect([...document.querySelectorAll('[data-test="gap"]')].some(gap => gap.textContent?.includes('Argo CD'))).toBe(false)
    await type(input('Your experience with Argo CD'), 'Ran Argo CD for a side project')
    await click(within(close(), 'Add to my CV'))
    expect(profile.experience[0]!.notes).toEqual([{ text: 'Ran Argo CD for a side project', skills: ['Argo CD'] }])
    expect(close().textContent).toContain('Added')
  })

  it('asks Claude’s questions first, saves an answer as a note of its role, and drops the rules’ duties', async () => {
    const { profile } = useServices(undefined, { probe: { status: 'ready', questions: [{
      theme: 'CI/CD', question: 'How many services use your pipelines?', why: 'Scale makes the CI/CD work concrete.',
      example: 'Used by <number> services', about: 'Built and maintained deployment pipelines for internal services.', place: 'experience:r1',
    }] } })
    await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    const asked = () => document.querySelector<HTMLElement>('[data-test="probe"]')!
    expect(asked().textContent).toContain('How many services use your pipelines?')
    expect(asked().textContent).toContain('Built and maintained deployment pipelines for internal services.')
    expect(text()).not.toContain('What this job will ask you to do')
    expect(text()).toContain('0 of 5 questions answered')
    await type(input('How many services use your pipelines?'), 'Used by 40 services across 3 teams')
    await click(within(asked(), 'Add to my CV'))
    expect(profile.experience[0]!.notes).toEqual([{ text: 'Used by 40 services across 3 teams', skills: [] }])
    expect(asked().textContent).toContain('Saved in your profile under DevOps Engineer at Verde Cloud')
    expect(text()).toContain('1 of 5 questions answered')
    await click(within(asked(), 'Change'))
    expect(profile.experience[0]!.notes).toEqual([])
    await click(within(asked(), 'Skip'))
    expect(asked().textContent).toContain('Skipped')
    await click(within(asked(), 'Change my mind'))
    expect(asked().textContent).toContain('Add to my CV')
  })

  it('says it is still reading while Claude’s questions are asked for', async () => {
    useServices(undefined, { probe: { status: 'pending', questions: [] } })
    await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    expect(document.querySelector('[data-test="probe-pending"]')!.textContent).toContain('Reading the job and your profile')
    expect(text()).toContain('What this job will ask you to do')
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

  it('speaks Romanian when the app does, and switches live', async () => {
    const person = sampleProfile()
    person.experience[0]!.tools.push('FluxCD')
    const { jobs } = useServices(person)
    jobs.jobs[0]!.must_have.push('Argo CD')
    await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    setLocale('ro')
    await settle()
    expect(text()).toContain('Pregătește CV-ul')
    expect(text()).toContain('0 din 8 întrebări cu răspuns')
    expect(text()).toContain('Completează competențele lipsă')
    expect(button(/^Generează CV-ul · potrivire \d+%$/)).toBeTruthy()
    const terraform = () => card('[data-test="gap"]', 'Terraform')
    expect(terraform().textContent).toContain('Lipsește')
    expect(terraform().textContent).toContain('Ai folosit Terraform? Spune-ne unde și ce ai făcut.')
    expect(input('Experiența ta cu Terraform').getAttribute('placeholder')).toBe('ex. Am folosit Terraform pentru ..., ceea ce a îmbunătățit ... cu ...')
    expect(document.querySelector('[data-test="claim"] h3')!.textContent).toMatch(/^Ai făcut așa ceva\? „.+”$/)
    expect(document.querySelector('[data-test="close"]')!.textContent).toMatch(/Folosești .+, un instrument de .+, ca .+\. CV-ul îți arată experiența de /)
    await type(input('Experiența ta cu Terraform'), 'Am scris module pentru 3 medii')
    await click(within(terraform(), 'Bonbon Labs'))
    await click(within(terraform(), 'Adaugă în CV'))
    expect(terraform().textContent).toContain('Salvat în profilul tău, la Junior system administrator la Bonbon Labs')
    expect(input('Ce altceva te recomandă pentru acest job?')).toBeTruthy()
    expect(text()).toContain('Ce îți va cere acest job')
    const claim = document.querySelector<HTMLElement>('[data-test="claim"]')!
    expect(input('Ce ai făcut mai exact?').getAttribute('placeholder')).toBe('Cu cuvintele tale: ce ai făcut, unde și ce s-a schimbat? Cifrele ajută.')
    await click(within(claim, 'Nu chiar'))
    expect(claim.textContent).toContain('Lăsat deoparte')
    const close = card('[data-test="close"]', 'Argo CD')
    expect(close.textContent).toContain('Am folosit și eu Argo CD')
    expect(close.textContent).toContain('Apropiat')
    setLocale('en')
    await settle()
    expect(text()).toContain('Prepare your CV')
  })

  it('writes a new CV in the app’s language when the ad does not tell', async () => {
    const { jobs } = useServices()
    Object.assign(jobs.jobs[0]!, { title: 'DevOps Engineer', about: '', responsibilities: [], must_have: ['Kubernetes'] })
    setLocale('ro')
    await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    expect(button('Română').getAttribute('aria-pressed')).toBe('true')
    await click(button('English'))
    expect(button('English').getAttribute('aria-pressed')).toBe('true')
  })

  it('says so when the profile cannot be read', async () => {
    useServices()
    server.use(http.get('*/api/profile/v1/profiles/me', () => HttpResponse.json({ message: 'down' }, { status: 503 })))
    await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    expect(text()).toContain('We could not prepare this CV')
  })

  it('sends a person whose profile says too little to complete it, and asks Claude nothing', async () => {
    useServices(emptyProfile())
    const asked: string[] = []
    const listen = ({ request }: { request: Request }) => asked.push(new URL(request.url).pathname)
    server.events.on('request:start', listen)
    const { router } = await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    server.events.removeListener('request:start', listen)
    const gate = document.querySelector('[data-test="profile-gate"]')!
    expect(gate.textContent).toContain('Your profile is 0% complete')
    expect(gate.textContent).toContain('at least 60% complete')
    expect(document.querySelector('[data-test="profile-gate-next"]')!.textContent).toBe('Add at least one role or project, with what you did there.')
    expect(document.querySelector('[data-test="gap"]')).toBeNull()
    expect(asked.some(path => path.endsWith('/probe'))).toBe(false)
    expect(button('Complete your profile').getAttribute('href')).toBe('/profile')
    await click(button('Complete your profile'))
    expect(router.currentRoute.value.name).toBe('crashr-profile')
  })

  it('says so in Romanian, with the next step when there is some work', async () => {
    const profile = sampleProfile({ headline: '', about: '', email: '', phone: '', github: '', linkedin: '', city: '' })
    useServices(profile)
    setLocale('ro')
    await mountPage(PrepPage, { at: `/jobs/${seedId(1)}/prepare`, props: { id: seedId(1) } })
    const gate = document.querySelector('[data-test="profile-gate"]')!
    expect(gate.textContent).toContain('Profilul tău e completat 54%')
    expect(gate.textContent).toContain('Completează-ți profilul')
    expect(document.querySelector('[data-test="profile-gate-next"]')!.textContent).toBe('Următorul pas: adaugă un titlu')
  })
})
