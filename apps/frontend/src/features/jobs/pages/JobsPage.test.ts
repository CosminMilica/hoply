import { afterEach, describe, expect, it } from 'vitest'
import { button, mountPage, useServices } from '@/test/hoply'
import JobsPage from './JobsPage.vue'

afterEach(() => {
  document.body.innerHTML = ''
})

describe('JobsPage', () => {
  it('finds jobs with the filters above them, and still offers to add one', async () => {
    useServices()
    await mountPage(JobsPage)
    expect(document.querySelector('h1')!.textContent).toBe('Find jobs')
    expect(document.querySelector('[data-test="filter-bar"]')).toBeTruthy()
    expect(button('Add a job').getAttribute('href')).toBe('/jobs/add')
  })
})
