import { afterEach, describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { button, click, mountPage, settle, useServices } from '@/test/hoply'
import JobPage from './JobPage.vue'
import { seedId } from '@/test/jobs-seed'

const text = () => document.body.textContent ?? ''

afterEach(() => {
  document.body.innerHTML = ''
})

describe('JobPage', () => {
  it('shows the job, the match and lets the person save it or not', async () => {
    const { applications, cvs } = useServices()
    await mountPage(JobPage, { at: `/jobs/${seedId(2)}`, props: { id: seedId(2) } })
    expect(text()).toContain('Bitdefender · Bucharest, hybrid')
    expect(text()).toContain('Sample listing, not a real posting · DevOps · Mid')
    expect(text()).toContain('Good match, with a couple of things to pick up.')
    expect(text()).toContain('Why people like it here')
    expect(text()).toContain('About Bitdefender')
    expect(text()).toContain('What we look for·Experience with Kubernetes.')
    expect(text()).not.toContain('View the original ad')
    expect(button('Prepare my CV').getAttribute('href')).toBe(`/jobs/${seedId(2)}/prepare`)
    await click(button('Save for later'))
    expect(applications.applications.get(seedId(2))?.status).toBe('saved')
    await click(button('Saved · tap to remove'))
    expect(applications.applications.has(seedId(2))).toBe(false)
    expect(cvs.analyses).toEqual([])
  })

  it('has Claude read an imported job once it is saved, and says when its board no longer lists it', async () => {
    const { applications, jobs, cvs } = useServices()
    Object.assign(jobs.jobs.find(job => job.id === seedId(2))!, {
      source: 'import', origin: 'greenhouse', sample: false, posted_at: null, closed_at: '2026-11-25T10:00:00Z',
    })
    await mountPage(JobPage, { at: `/jobs/${seedId(2)}`, props: { id: seedId(2) } })
    expect(text()).toContain('via Greenhouse · DevOps · Mid')
    expect(document.querySelector('[data-test="closed"]')?.textContent).toContain('The company no longer lists this job')
    await click(button('Save for later'))
    expect(applications.applications.get(seedId(2))?.status).toBe('saved')
    expect(cvs.analyses).toEqual([seedId(2)])
  })

  it('follows the company of a job added from its board, so its jobs come to Find jobs', async () => {
    const { jobs } = useServices()
    const added = (id: string, url: string) => jobs.jobs.push({ ...structuredClone(jobs.jobs[1]!), id, url, source: 'link', added_by_me: true, sample: false, company: 'Northwind' })
    added('11111111-0000-4000-8000-000000000001', 'https://jobs.lever.co/northwind/aa11')
    added('11111111-0000-4000-8000-000000000002', 'https://www.linkedin.com/jobs/view/42')
    added('11111111-0000-4000-8000-000000000003', 'https://job-boards.greenhouse.io/awin/jobs/1')
    jobs.boards.push({ platform: 'greenhouse', board: 'awin', company: 'Awin', followed_at: '' })

    await mountPage(JobPage, { at: '/jobs/11111111-0000-4000-8000-000000000001', props: { id: '11111111-0000-4000-8000-000000000001' } })
    await vi.waitFor(() => expect(text()).toContain('Northwind posts its jobs on Lever.'))
    await click(button('Follow Northwind'))
    expect(jobs.boards.map(board => `${board.platform}/${board.board}/${board.company}`)).toEqual(['greenhouse/awin/Awin', 'lever/northwind/Northwind'])
    expect(text()).toContain('Following Northwind. Its jobs in Romania show up in Find jobs in a few minutes.')
    document.body.innerHTML = ''

    await mountPage(JobPage, { at: '/jobs/11111111-0000-4000-8000-000000000003', props: { id: '11111111-0000-4000-8000-000000000003' } })
    await vi.waitFor(() => expect(text()).toContain('Hoply brings Awin’s jobs in Romania to Find jobs.'))
    document.body.innerHTML = ''

    await mountPage(JobPage, { at: '/jobs/11111111-0000-4000-8000-000000000002', props: { id: '11111111-0000-4000-8000-000000000002' } })
    await settle()
    expect(document.querySelector('[data-test="follow-company"]')).toBeNull()
  })

  it('says when following a company fails', async () => {
    const { jobs } = useServices()
    jobs.jobs.push({ ...structuredClone(jobs.jobs[1]!), id: '11111111-0000-4000-8000-000000000001', url: 'https://jobs.ashbyhq.com/linear/1', source: 'link', added_by_me: true, company: '' })
    server.use(http.post('*/api/jobs/v1/boards', () => HttpResponse.json({ error: { code: 'INTERNAL' } }, { status: 500 })))
    await mountPage(JobPage, { at: '/jobs/11111111-0000-4000-8000-000000000001', props: { id: '11111111-0000-4000-8000-000000000001' } })
    await vi.waitFor(() => expect(text()).toContain('This company posts its jobs on Ashby.'))
    await click(button('Follow This company'))
    await vi.waitFor(() => expect(document.querySelector('[role="alert"]')?.textContent).toContain('We could not follow This company. Try again.'))
  })

  it('shows what the ad asks in its own words, and what is a plus', async () => {
    const { jobs } = useServices()
    Object.assign(jobs.jobs.find(job => job.id === seedId(2))!, {
      qualifications: ['6+ years of experience working in Python;', 'Good unit test and debugging skills;'], bonuses: ['Knowledge of MongoDB;'],
    })
    await mountPage(JobPage, { at: `/jobs/${seedId(2)}`, props: { id: seedId(2) } })
    expect(text()).toContain('What we look for·6+ years of experience working in Python;·Good unit test and debugging skills;')
    expect(text()).toContain('Nice to have·Knowledge of MongoDB;')
    expect(text()).not.toContain('Experience with Kubernetes.')
  })

  it('offers a new CV once one is made, and no save button', async () => {
    const { applications } = useServices()
    applications.track(seedId(1), 'applied')
    await mountPage(JobPage, { at: `/jobs/${seedId(1)}`, props: { id: seedId(1) } })
    expect(button('Prepare a new CV')).toBeTruthy()
    expect(text()).not.toContain('Save for later')
  })

  it('goes back where the person came from, or to Saved', async () => {
    useServices()
    const { router } = await mountPage(JobPage, { at: `/jobs/${seedId(1)}`, props: { id: seedId(1) } })
    await click(button('Back'))
    expect(router.currentRoute.value.name).toBe('hoply-saved')
  })

  it('shows a job the person added: its source, its original ad, and only the sections it has', async () => {
    const { jobs } = useServices()
    jobs.jobs.unshift({
      ...jobs.jobs[1]!, id: 'mine', source: 'text', read_from: 'text', added_by_me: true, sample: false, url: 'https://ejobs.ro/123',
      about: '', about_company: '', perks: [], responsibilities: [], areas: [], must_have: [], nice_to_have: [], work_mode: '', city: '',
    })
    await mountPage(JobPage, { at: '/jobs/mine', props: { id: 'mine' } })
    expect(text()).toContain('Added by you · from pasted text · Mid')
    expect(text()).toContain('Bitdefender · Location not given')
    expect(button('View the original ad ↗').getAttribute('href')).toBe('https://ejobs.ro/123')
    for (const section of ['About the role', 'What you will do', 'What we look for', 'Why people like it here', 'About Bitdefender']) {
      expect(text()).not.toContain(section)
    }
  })

  it('says when Claude read the requirements', async () => {
    const { jobs } = useServices()
    await mountPage(JobPage, { at: `/jobs/${jobs.jobs[0]!.id}`, props: { id: jobs.jobs[0]!.id } })
    expect(document.querySelector('[data-test="requirements-from"]')).toBeNull()
    document.body.innerHTML = ''
    jobs.jobs.unshift({ ...jobs.jobs[0]!, id: 'read', requirements_from: 'analysis' })
    await mountPage(JobPage, { at: '/jobs/read', props: { id: 'read' } })
    expect(document.querySelector('[data-test="requirements-from"]')!.textContent).toBe('Requirements as Claude read them from the ad.')
  })

  it('says so for an unknown job', async () => {
    useServices()
    await mountPage(JobPage, { at: '/jobs/9', props: { id: '9' } })
    expect(text()).toContain('We could not find this job')
  })
})
