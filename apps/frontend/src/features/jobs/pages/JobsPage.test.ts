import { afterEach, describe, expect, it } from 'vitest'
import { button, mountPage } from '@/test/hoply'
import JobsPage from './JobsPage.vue'

afterEach(() => {
  document.body.innerHTML = ''
})

describe('JobsPage', () => {
  it('says finding jobs is coming, and offers to add a job instead', async () => {
    await mountPage(JobsPage)
    expect(document.querySelector('h1')!.textContent).toBe('Finding jobs is coming soon')
    expect(document.body.textContent).toContain('We will search job boards for you, based on your profile')
    expect(button(/^Add a job from a link or its text/).getAttribute('href')).toBe('/jobs/add')
    expect(document.body.textContent).not.toContain('Position area')
    expect(document.querySelector('[data-test="job-card"]')).toBeNull()
  })
})
