import { afterEach, describe, expect, it } from 'vitest'
import { button, click, mountPage, useServices } from '@/test/hoply'
import SavedPage from './SavedPage.vue'
import { seedId } from '@/test/jobs-seed'

const text = () => document.body.textContent ?? ''

afterEach(() => {
  document.body.innerHTML = ''
})

describe('SavedPage', () => {
  it('lists tracked jobs by status, with what each needs next', async () => {
    const { applications } = useServices()
    applications.track(seedId(1), 'saved')
    applications.track(seedId(2), 'cv_ready')
    applications.track(seedId(3), 'applied')
    const { router } = await mountPage(SavedPage)
    expect(button('All 3').getAttribute('aria-pressed')).toBe('true')
    expect([...document.querySelectorAll('[data-test="since"]')].map(line => line.textContent)).toEqual(['Applied 25 Nov', 'CV ready 25 Nov', 'Saved 25 Nov'])
    expect(text()).toContain('Missing: Snowflake, Terraform, Infrastructure as Code, RBAC')
    expect([...document.querySelectorAll('a')].find(link => link.textContent?.trim() === 'Skill gaps')?.getAttribute('href')).toBe('/saved/skills')

    await click(button('Saved 1'))
    expect(text()).toContain('Snowflake Administrator DevOps')
    expect(text()).not.toContain('Software Engineer')

    await click(button('CV ready 1'))
    await click(button('I applied with this CV'))
    expect(applications.applications.get(seedId(2))?.status).toBe('applied')
    expect(text()).toContain('No jobs with this status yet.')

    await click(button('Applied 2'))
    await click(button('View CV'))
    expect(router.currentRoute.value.name).toBe('hoply-cv')
  })

  it('prepares a CV for a saved job, or removes it', async () => {
    const { applications } = useServices()
    applications.track(seedId(4), 'saved')
    const { router } = await mountPage(SavedPage)
    expect(text()).toContain('Missing: Windows Server')
    await click(button('Prepare CV'))
    expect(router.currentRoute.value).toMatchObject({ name: 'hoply-prep', params: { id: seedId(4) } })
    await click(button('Remove from saved'))
    expect(applications.applications.size).toBe(0)
    expect(text()).toContain('Nothing saved yet. Add a job you found, from its link or its text, and it lands here.')
    const adds = [...document.querySelectorAll('a')].filter(link => link.textContent?.trim() === 'Add a job')
    expect(adds.map(link => link.getAttribute('href'))).toEqual(['/jobs/add?from=saved', '/jobs/add?from=saved'])
    expect(text()).not.toContain('Find jobs')
  })
})
