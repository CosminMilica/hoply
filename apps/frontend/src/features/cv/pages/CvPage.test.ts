import { afterEach, describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { setLocale } from '@/features/common/i18n'
import { cvApi } from '../api/cvApi'
import { button, click, mountPage, useServices } from '@/test/hoply'
import { sampleProfile } from '@/test/profile-service'
import CvPage from './CvPage.vue'
import { seedId } from '@/test/jobs-seed'

const text = () => document.body.textContent ?? ''
const request = { job_id: seedId(1), skip_roles: [], focus: [], confirmed: [], language: 'en' as const }

afterEach(() => {
  document.body.innerHTML = ''
  localStorage.clear()
  vi.restoreAllMocks()
})

describe('CvPage', () => {
  it('shows the tailored CV with its ATS check, and the profile version', async () => {
    const { applications } = useServices()
    applications.track(seedId(1), 'cv_ready')
    await cvApi.generate(request)
    const { router } = await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
    expect(text()).toContain('For Snowflake Administrator DevOps at Capgemini Romania')
    expect(document.querySelector('[data-test="written-by"]')!.textContent).toBe('Written without AI. The service is set to write with rules.')
    expect(button('Tailored').getAttribute('aria-pressed')).toBe('true')
    expect(text()).toContain('ATS check')
    expect(text()).toContain('strongest in SQL, Git and CI/CD')
    expect(document.querySelectorAll('[data-print-root] mark').length).toBeGreaterThan(0)
    expect(text()).toContain('Andrei_Popescu_CV_Capgemini_Romania.pdf')
    const lines = [...document.querySelectorAll('[data-print-root] header span')].map(element => element.textContent)
    expect(lines.slice(0, 2)).toEqual(['Andrei Popescu', 'DevOps Engineer'])

    await click(button('Your profile'))
    expect(text()).not.toContain('ATS check')
    expect(document.querySelectorAll('[data-print-root] mark')).toHaveLength(0)

    await click(button('I applied to this job'))
    expect(applications.applications.get(seedId(1))?.status).toBe('applied')
    expect(button('Marked as applied ✓').disabled).toBe(true)

    await click(button('Tailored'))
    await click(button('Improve this CV'))
    expect(router.currentRoute.value.name).toBe('hoply-prep')
  })

  it('downloads the version on screen as a PDF or a DOCX from the service', async () => {
    const { cvs } = useServices()
    URL.createObjectURL = vi.fn(() => 'blob:cv')
    URL.revokeObjectURL = vi.fn()
    const saved: string[] = []
    const clicked = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) { saved.push(this.download) })
    await cvApi.generate(request)
    await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
    const id = cvs.cvs[0]!.cv.id
    await click(button('Download PDF'))
    await click(button('DOCX'))
    await click(button('Your profile'))
    await click(button('Download PDF'))
    await vi.waitFor(() => expect(cvs.files).toHaveLength(3))
    expect(cvs.files).toEqual([
      { id, format: 'pdf', version: 'tailored', template: 'classic' },
      { id, format: 'docx', version: 'tailored', template: 'classic' },
      { id, format: 'pdf', version: 'profile', template: 'classic' },
    ])
    expect(saved).toEqual([cvs.cvs[0]!.cv.file_name, cvs.cvs[0]!.cv.file_name, cvs.cvs[0]!.cv.file_name])

    server.use(http.get('*/api/cv/v1/cvs/:id/file', () => HttpResponse.json({ error: { code: 'DEPENDENCY_UNAVAILABLE' } }, { status: 503 })))
    await click(button('Download PDF'))
    await vi.waitFor(() => expect(document.querySelector('[role="alert"]')?.textContent).toContain('We could not download the CV'))
    clicked.mockRestore()
  })

  it('shows and downloads the CV in the template the person picks, and remembers it', async () => {
    const { cvs } = useServices()
    await cvApi.generate(request)
    await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
    const id = cvs.cvs[0]!.cv.id
    const paper = () => document.querySelector('[data-print-root]')!
    URL.createObjectURL = vi.fn(() => 'blob:cv')
    URL.revokeObjectURL = vi.fn()
    const clicked = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    expect(paper().getAttribute('data-template')).toBe('classic')
    expect(button('Classic').getAttribute('aria-pressed')).toBe('true')
    await click(button('Elegant'))
    expect(paper().getAttribute('data-template')).toBe('elegant')
    await click(button('DOCX'))
    await vi.waitFor(() => expect(cvs.files.at(-1)).toEqual({ id, format: 'docx', version: 'tailored', template: 'elegant' }))
    clicked.mockRestore()
    await click(button('Modern'))
    document.body.innerHTML = ''
    await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
    expect(paper().getAttribute('data-template')).toBe('modern')
    localStorage.setItem('hoply.cv.template', 'toString')
    document.body.innerHTML = ''
    await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
    expect(paper().getAttribute('data-template')).toBe('classic')
  })

  it('draws a Romanian CV with Romanian headings', async () => {
    const { applications } = useServices()
    applications.track(seedId(1), 'cv_ready')
    await cvApi.generate({ ...request, language: 'ro' })
    await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
    const paper = document.querySelector('[data-print-root]')!
    expect(paper.getAttribute('lang')).toBe('ro')
    const headings = [...paper.querySelectorAll('h2')].map(heading => heading.textContent)
    expect(headings).toContain('Experiență profesională')
    expect(headings).not.toContain('Experience')
  })

  it('leaves out the sections the CV has nothing for', async () => {
    useServices(sampleProfile({ languages: [], skills: [], certifications: [], education: [] }))
    await cvApi.generate(request)
    await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
    const headings = [...document.querySelectorAll('[data-print-root] h2')].map(element => element.textContent)
    expect(headings).not.toContain('Languages')
    expect(headings).not.toContain('Education')
    expect(headings).not.toContain('Certifications')
    expect(headings).toContain('Experience')
  })

  it('starts with the headline and keeps the plain file name without a name', async () => {
    useServices(sampleProfile({ name: '' }))
    await cvApi.generate(request)
    await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
    expect(document.querySelector('[data-test="cv-name"]')).toBeNull()
    expect(document.querySelector('[data-print-root] header span')!.textContent).toBe('DevOps Engineer')
    expect(text()).toContain('CV_Capgemini_Romania.pdf')
    expect(text()).not.toContain('_CV_Capgemini')
  })

  it('shows only a CV that is ready', async () => {
    useServices(undefined, { writing: 5 })
    await cvApi.generate(request)
    await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
    expect(text()).toContain('No CV yet')
  })

  it('says how Claude tailored the CV', async () => {
    const { cvs } = useServices()
    await cvApi.generate(request)
    cvs.cvs[0]!.cv.tailoring = {
      positioning: 'The engineer who automates delivery',
      priorities: [{ theme: 'CI/CD', strength: 'direct' }, { theme: 'GitOps', strength: 'transferable' }],
      close: [{ skill: 'Argo CD', via: ['FluxCD'], category: 'GitOps' }],
      changes: ['Led with the pipelines'],
    }
    await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
    const card = document.querySelector('[data-test="tailoring"]')!
    expect(card.textContent).toContain('The engineer who automates delivery')
    expect(card.textContent).toContain('CI/CDProven')
    expect(card.textContent).toContain('GitOpsTransferable')
    expect(card.textContent).toContain('Led with the pipelines')
    expect(card.textContent).toContain('Argo CD as GitOps experience with FluxCD')
    await click(button('Your profile'))
    expect(document.querySelector('[data-test="tailoring"]')).toBeNull()
  })

  it('speaks Romanian around an English CV when the app is in Romanian', async () => {
    const { cvs } = useServices()
    await cvApi.generate(request)
    cvs.cvs[0]!.cv.tailoring = {
      positioning: 'The engineer who automates delivery',
      priorities: [{ theme: 'CI/CD', strength: 'direct' }],
      close: [{ skill: 'Argo CD', via: ['FluxCD'], category: '' }],
      changes: [],
    }
    setLocale('ro')
    await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
    expect(text()).toContain('CV-ul tău e gata')
    expect(text()).toContain('Pentru Snowflake Administrator DevOps la Capgemini Romania')
    expect(document.querySelector('[data-test="written-by"]')!.textContent).toBe('Scris fără AI. Serviciul e setat să scrie cu reguli.')
    expect(text()).toContain('Verificare ATS')
    expect(text()).toContain('O singură coloană, text simplu, fără tabele sau imagini')
    expect(text()).toMatch(/Cuvinte-cheie din anunț: \d+ din \d+/)
    expect(text()).toContain('Titlul jobului „Snowflake Administrator DevOps” în rezumat')
    expect(text()).not.toContain('Single column, plain text')
    expect(button('Clasic').getAttribute('aria-pressed')).toBe('true')
    expect(button('Descarcă PDF')).toBeTruthy()
    expect(button('Am aplicat la acest job')).toBeTruthy()
    const tailoring = document.querySelector('[data-test="tailoring"]')!
    expect(tailoring.textContent).toContain('CI/CDDovedit')
    expect(tailoring.textContent).toContain('Argo CD ca experiență înrudită cu FluxCD')
    const paper = document.querySelector('[data-print-root]')!
    expect(paper.getAttribute('aria-label')).toBe('Previzualizare CV')
    expect([...paper.querySelectorAll('h2')].map(heading => heading.textContent)).toContain('Experience')
  })

  it('has nothing to show before a CV is made', async () => {
    useServices()
    await mountPage(CvPage, { at: `/jobs/${seedId(2)}/cv`, props: { id: seedId(2) } })
    expect(text()).toContain('No CV yet')
    expect(text()).toContain('Add a job, prepare your CV, and the tailored version shows up here.')
    expect(button('Add a job').getAttribute('href')).toBe('/jobs/add')
  })
})
