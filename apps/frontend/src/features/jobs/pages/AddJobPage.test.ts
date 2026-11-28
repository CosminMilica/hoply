import { afterEach, describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { setLocale } from '@/features/common/i18n'
import { button, click, input, mountPage, settle, type, useServices } from '@/test/hoply'
import { sampleProfile } from '@/test/profile-service'
import AddJobPage from './AddJobPage.vue'

const text = () => document.body.textContent ?? ''
const failed = () => document.querySelector('[data-test="read-failed"]')?.textContent ?? ''
const submit = async (label: string) => {
  input(label).closest('form')!.dispatchEvent(new Event('submit', { cancelable: true }))
  await settle()
}
const AD = ['DevOps Engineer', 'Firma: Carpathia Systems, Brașov (hibrid)', 'Responsabilități', '- Administrezi servere Linux.',
  '- Automatizezi infrastructura cu Terraform.', 'Cerințe', '- Minimum 3 ani experiență în DevOps.'].join('\n')

async function readLink(url: string) {
  await type(input('Link to the job ad'), url)
  await submit('Link to the job ad')
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('AddJobPage', () => {
  it('reads a link, lets the person fix it, and saves it to their jobs', async () => {
    const { jobs, applications, cvs } = useServices()
    const { router } = await mountPage(AddJobPage, { at: '/jobs/add?from=saved' })
    expect(button('Link').getAttribute('aria-selected')).toBe('true')
    await readLink('https://careers.northwind.example/jobs/platform-engineer')
    expect(text()).toContain('Read from the job data on careers.northwind.example. Fix anything we got wrong.')
    expect(document.querySelector('[data-test="draft-match"]')!.textContent).toContain('38%')
    expect(document.querySelector('[data-test="knockout"]')!.textContent).toBe('Based in Cluj-Napoca, hybrid. You live in Iași and are not open to relocation.')
    expect(document.querySelector('[data-test="years"]')!.textContent).toBe('Asks for 4+ years of experience. You have about 6.')
    expect(button('Remove Linux').className).toContain('bg-hoply-900')
    expect(button('Remove Kubernetes').className).not.toContain('bg-hoply-900')
    expect(text()).toContain('Responsibilities we found · 4')

    await type(input('Job title'), 'Senior Platform Engineer')
    await type(input('City'), 'Iași')
    await click(button('Hybrid'))
    expect(button('Hybrid').getAttribute('aria-pressed')).toBe('false')
    await click(button('Senior'))
    await click(button('Remove Kafka'))
    await type(input('Add a requirement'), 'Ansible')
    await click(button('+ Nice'))
    await type(input('Add a requirement'), 'kubernetes')
    await click(button('+ Must'))
    await type(input('Add a requirement'), '  ')
    await click(button('+ Must'))
    expect(document.querySelector('[data-test="knockout"]')).toBeNull()

    await click(button('Save to my jobs'))
    await settle()
    const saved = jobs.jobs[0]!
    expect(saved).toMatchObject({
      title: 'Senior Platform Engineer', company: 'Northwind Cloud', city: 'Iași', work_mode: '', level: 'senior', years_required: 4,
      nice_to_have: ['Helm', 'PostgreSQL', 'Ansible'], source: 'link', read_from: 'json_ld', added_by_me: true,
      url: 'https://careers.northwind.example/jobs/platform-engineer',
      qualifications: ['4+ years running Kubernetes on AWS.', 'You write Go and PostgreSQL every week.'], bonuses: ['Kafka in production.'],
      about_company: 'Northwind Cloud runs payments for small shops across Europe.', perks: ['Learning budget'],
    })
    expect(saved.must_have).not.toContain('kubernetes')
    await vi.waitFor(() => expect(cvs.analyses).toEqual([saved.id]))
    expect(applications.applications.get(saved.id)?.status).toBe('saved')
    expect(router.currentRoute.value.name).toBe('hoply-saved')
  })

  it('saves and opens Prepare your CV', async () => {
    const { jobs } = useServices()
    const { router } = await mountPage(AddJobPage, { at: '/jobs/add' })
    await readLink('https://careers.northwind.example/jobs/platform-engineer')
    await click(button('Save and prepare my CV'))
    await settle()
    expect(router.currentRoute.value).toMatchObject({ name: 'hoply-prep', params: { id: jobs.jobs[0]!.id } })
  })

  it('saves the job once when only saving its status failed, then retries the status', async () => {
    const { jobs, applications } = useServices()
    let failures = 1
    server.use(http.put('*/api/applications/v1/applications/:jobId', () => (failures-- > 0 ? HttpResponse.json({ error: { code: 'DEPENDENCY_UNAVAILABLE', message: 'down' } }, { status: 503 }) : undefined)))
    await mountPage(AddJobPage, { at: '/jobs/add' })
    await readLink('https://careers.northwind.example/jobs/platform-engineer')
    const before = jobs.jobs.length
    await click(button('Save to my jobs'))
    await settle()
    expect(document.querySelector('[role="alert"]')).toBeTruthy()
    expect(jobs.jobs.length).toBe(before + 1)
    await click(button('Save to my jobs'))
    await settle()
    expect(jobs.jobs.length).toBe(before + 1)
    expect(applications.applications.get(jobs.jobs[0]!.id)?.status).toBe('saved')
  })

  it('reads pasted text with its hints and keeps the link', async () => {
    const { jobs } = useServices(sampleProfile({ work_modes: ['remote'] }))
    await mountPage(AddJobPage, { at: '/jobs/add' })
    await click(button('Text'))
    await type(input('The job ad'), 'Too short')
    await submit('The job ad')
    expect(text()).toContain('Paste the whole ad: a few lines with what they ask for and what you would do.')
    await type(input('The job ad'), AD)
    await type(input('Company (optional)'), 'Carpathia Systems')
    await type(input('Link to the ad (optional, to find it again)'), 'https://ejobs.ro/123')
    await submit('The job ad')
    expect(text()).toContain('Read from the text you pasted')
    expect((input('Job title') as HTMLInputElement).value).toBe('DevOps Engineer')
    expect((input('Company') as HTMLInputElement).value).toBe('Carpathia Systems')
    expect(document.querySelector('[data-test="knockout"]')!.textContent).toBe('Hybrid in Brașov. Your profile asks for remote work, and you are not open to relocation.')
    expect(document.querySelector('[data-test="years"]')).toBeNull()
    await click(button('Save to my jobs'))
    await settle()
    expect(jobs.jobs[0]).toMatchObject({ source: 'text', read_from: 'text', url: 'https://ejobs.ro/123', responsibilities: ['Administrezi servere Linux.', 'Automatizezi infrastructura cu Terraform.', 'Minimum 3 ani experiență în DevOps.'] })
  })

  it('reads a job the Send to Hoply bookmark sent, and offers the bookmark', async () => {
    useServices()
    const sent = new URLSearchParams({ text: AD, url: 'https://www.linkedin.com/jobs/view/42', title: 'Platform Engineer', company: 'spektr' })
    const { router } = await mountPage(AddJobPage, { at: `/jobs/add#${sent}` })
    await vi.waitFor(() => expect(text()).toContain('Read from the text you pasted'))
    expect((input('Job title') as HTMLInputElement).value).toBe('Platform Engineer')
    expect((input('Company') as HTMLInputElement).value).toBe('spektr')
    expect(router.currentRoute.value.hash).toBe('')
    await click(button('Back'))
    expect((input('Link to the ad (optional, to find it again)') as HTMLInputElement).value).toBe('https://www.linkedin.com/jobs/view/42')
    const bookmark = document.querySelector<HTMLAnchorElement>('[data-test="send-to-hoply"] a')!
    expect(bookmark.getAttribute('href')).toMatch(/^javascript:/)
    bookmark.click()
    expect(router.currentRoute.value.name).toBe('hoply-job-add')
  })

  it.each([
    ['https://www.linkedin.com/jobs/view/1', 'LinkedIn shows job pages only to signed-in members'],
    ['https://empty.example.org/jobs', 'We found no job ad on this page'],
    ['https://down.example.org/jobs', 'The page did not answer'],
    ['https://huge.example.org/jobs', 'The page is too big for us to read'],
    ['https://10.0.0.1/jobs', 'private or local address'],
    ['https://odd.example.org/jobs', 'Something on this page stopped us'],
  ])('explains why %s could not be read and offers the text instead', async (url, message) => {
    useServices()
    await mountPage(AddJobPage, { at: '/jobs/add' })
    await readLink(url)
    expect(failed()).toContain('We could not read this page')
    expect(failed()).toContain(message)
    await click(button('Paste the text instead'))
    expect(button('Text').getAttribute('aria-selected')).toBe('true')
    expect((input('Link to the ad (optional, to find it again)') as HTMLInputElement).value).toBe(url)
    await click(button('Link'))
    await readLink(url)
    await click(button('Try another link'))
    expect(button('Link').getAttribute('aria-selected')).toBe('true')
  })

  it('checks the link and the title, and shows what the service refuses', async () => {
    useServices()
    await mountPage(AddJobPage, { at: '/jobs/add' })
    await readLink('linkedin.com/jobs/1')
    expect(text()).toContain('Enter the full link, starting with https://')
    server.use(http.post('*/api/jobs/v1/jobs/read', () => HttpResponse.json({ error: { code: 'VALIDATION_ERROR', message: 'url is too long' } }, { status: 422 })))
    await readLink('https://a.example.org/x')
    expect(document.querySelector('[role="alert"]')!.textContent).toBe('url is too long')
    server.resetHandlers()
    useServices()
    await readLink('https://careers.northwind.example/jobs/1')
    await type(input('Job title'), ' ')
    await click(button('Save to my jobs'))
    expect(text()).toContain('Add the job title.')
    await type(input('Job title'), 'Platform Engineer')
    server.use(http.post('*/api/jobs/v1/jobs', () => HttpResponse.json({ error: { code: 'VALIDATION_ERROR', message: 'title is too long' } }, { status: 422 })))
    await click(button('Save to my jobs'))
    await settle()
    expect(document.querySelector('[role="alert"]')!.textContent).toBe('title is too long')
  })

  it('asks for requirements when none were found', async () => {
    useServices()
    server.use(http.post('*/api/jobs/v1/jobs/read', () => HttpResponse.json({
      title: 'Office manager', company: '', city: '', country: '', work_mode: '', level: '', years_required: null, must_have: [], nice_to_have: [],
      responsibilities: [], about: '', url: 'https://a.example.org/x', source: 'link', read_from: 'page_text', host: 'a.example.org',
    })))
    await mountPage(AddJobPage, { at: '/jobs/add' })
    await readLink('https://a.example.org/x')
    expect(text()).toContain('Read from the page on a.example.org')
    expect(text()).toContain('We did not recognise any skills in this ad.')
    expect(text()).not.toContain('Responsibilities we found')
  })

  it('goes back to the form from the preview, then where the person came from', async () => {
    useServices()
    const { router } = await mountPage(AddJobPage, { at: '/jobs/add?from=saved' })
    await readLink('https://careers.northwind.example/jobs/1')
    await click(button('Back'))
    expect(text()).toContain('Link to the job ad')
    await click(button('Back'))
    expect(router.currentRoute.value.name).toBe('hoply-saved')
    await router.push('/jobs/add')
    await settle()
    await click(button('Back'))
    expect(router.currentRoute.value.name).toBe('hoply-jobs')
  })

  it('speaks Romanian: the form, its checks, the bookmark, a page it could not read and the preview', async () => {
    setLocale('ro')
    useServices()
    await mountPage(AddJobPage, { at: '/jobs/add' })
    expect(text()).toContain('Adaugă un job')
    expect(button('Link').getAttribute('aria-selected')).toBe('true')
    expect(text()).toContain('Trimite joburi aici de pe LinkedIn, eJobs sau orice site')
    expect(document.querySelector('[data-test="send-to-hoply"] a')!.getAttribute('title')).toBe('Trage-mă în bara de marcaje')
    await type(input('Linkul anunțului'), 'linkedin.com/jobs/1')
    await submit('Linkul anunțului')
    expect(text()).toContain('Scrie linkul complet, care începe cu https://')
    await type(input('Linkul anunțului'), 'https://www.linkedin.com/jobs/view/1')
    await submit('Linkul anunțului')
    expect(failed()).toContain('Nu am putut citi pagina')
    expect(failed()).toContain('LinkedIn arată anunțurile doar membrilor autentificați')
    await click(button('Lipește textul'))
    expect(input('Linkul anunțului (opțional, ca să-l regăsești)')).toBeTruthy()
    await type(input('Anunțul'), 'Prea scurt')
    await submit('Anunțul')
    expect(text()).toContain('Lipește tot anunțul: câteva rânduri cu ce cer și ce vei face.')
    await click(button('Link'))

    await type(input('Linkul anunțului'), 'https://careers.northwind.example/jobs/platform-engineer')
    await submit('Linkul anunțului')
    expect(text()).toContain('Verifică ce am găsit')
    expect(text()).toContain('Citit din datele jobului de pe careers.northwind.example. Corectează ce am greșit.')
    expect(document.querySelector('[data-test="knockout"]')!.textContent).toBe('Jobul e în Cluj-Napoca, hibrid. Tu locuiești în Iași și nu vrei să te muți.')
    expect(document.querySelector('[data-test="years"]')!.textContent).toBe('Cere cel puțin 4 ani de experiență. Tu ai cam 6 ani.')
    expect(text()).toContain('Responsabilități găsite · 4')
    expect(button('Hibrid').getAttribute('aria-pressed')).toBe('true')
    expect(button('La birou')).toBeTruthy()
    expect(button('Șterge Linux')).toBeTruthy()
    await type(input('Adaugă o cerință'), 'Ansible')
    await click(button('+ De preferat'))
    expect(button('Șterge Ansible').className).toContain('border-dashed')
    expect(button('Salvează și pregătește CV-ul')).toBeTruthy()
    expect(button('Salvează în joburile mele')).toBeTruthy()
  })
})
