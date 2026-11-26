import { afterEach, describe, expect, it } from 'vitest'
import { button, mountPage, useServices } from '@/test/hoply'
import JobsPage from './JobsPage.vue'

afterEach(() => {
  document.body.innerHTML = ''
})

describe('JobsPage', () => {
  it('finds jobs with filters, and still offers to add one', async () => {
    useServices()
    await mountPage(JobsPage)
    expect(document.querySelector('h1')!.textContent).toBe('Find jobs')
    expect(document.body.textContent).toContain('Position area')
    expect(button(/^Add a job from a link or its text/).getAttribute('href')).toBe('/jobs/add')
  })
})
