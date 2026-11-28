import { afterEach, describe, expect, it } from 'vitest'
import { mountPage, useServices } from '@/test/crashr'
import JobsPage from './JobsPage.vue'

afterEach(() => {
  document.body.innerHTML = ''
})

describe('JobsPage', () => {
  it('finds jobs with the filters above them', async () => {
    useServices()
    await mountPage(JobsPage)
    expect(document.querySelector('h1')!.textContent).toBe('Find jobs')
    expect(document.querySelector('[data-test="filter-bar"]')).toBeTruthy()
  })
})
