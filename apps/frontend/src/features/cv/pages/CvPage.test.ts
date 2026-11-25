import { afterEach, describe, expect, it, vi } from 'vitest'
import { cvApi } from '../api/cvApi'
import { button, click, mountPage, useServices } from '@/test/hoply'
import { sampleProfile } from '@/test/profile-service'
import CvPage from './CvPage.vue'
import { seedId } from '@/test/jobs-seed'

const text = () => document.body.textContent ?? ''
const request = { job_id: seedId(1), skip_roles: [], focus: [], confirmed: [] }

afterEach(() => {
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

describe('CvPage', () => {
  it('shows the tailored CV with its ATS check, and the profile version', async () => {
    const { applications } = useServices()
    applications.track(seedId(1), 'cv_ready')
    await cvApi.generate(request)
    const { router } = await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
    expect(text()).toContain('For Snowflake Administrator DevOps at Capgemini Romania')
    expect(button('Tailored').getAttribute('aria-pressed')).toBe('true')
    expect(text()).toContain('ATS check')
    expect(text()).toContain('DevOps Engineer | SQL, Git, CI/CD')
    expect(document.querySelectorAll('[data-print-root] mark').length).toBeGreaterThan(0)
    expect(text()).toContain('Andrei_Popescu_CV_Capgemini_Romania.pdf')
    const lines = [...document.querySelectorAll('[data-print-root] header span')].map(element => element.textContent)
    expect(lines.slice(0, 2)).toEqual(['Andrei Popescu', 'DevOps Engineer | SQL, Git, CI/CD'])

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

  it('prints the CV under its file name', async () => {
    useServices()
    await cvApi.generate(request)
    await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
    const titles: string[] = []
    vi.spyOn(window, 'print').mockImplementation(() => titles.push(document.title))
    document.title = 'hoply'
    await click(button('Download PDF'))
    expect(titles).toEqual(['Andrei_Popescu_CV_Capgemini_Romania'])
    expect(document.title).toBe('hoply')
  })

  it('starts with the headline and keeps the plain file name without a name', async () => {
    useServices(sampleProfile({ name: '' }))
    await cvApi.generate(request)
    await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
    expect(document.querySelector('[data-test="cv-name"]')).toBeNull()
    expect(document.querySelector('[data-print-root] header span')!.textContent).toBe('DevOps Engineer | SQL, Git, CI/CD')
    expect(text()).toContain('CV_Capgemini_Romania.pdf')
    expect(text()).not.toContain('_CV_Capgemini')
  })

  it('has nothing to show before a CV is made', async () => {
    useServices()
    await mountPage(CvPage, { at: `/jobs/${seedId(2)}/cv`, props: { id: seedId(2) } })
    expect(text()).toContain('No CV yet')
    expect(text()).toContain('Add a job, prepare your CV, and the tailored version shows up here.')
    expect(button('Add a job').getAttribute('href')).toBe('/jobs/add')
  })
})
