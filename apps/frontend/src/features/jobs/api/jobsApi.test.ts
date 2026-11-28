import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { jobsServiceHandlers } from '@/test/jobs-service'
import { setLocale } from '@/features/common/i18n'
import { jobDraftSchema, jobInputSchema } from './jobs.schema'
import { jobSearchParams, jobsApi, unreadableReason } from './jobsApi'
import { seedId } from '@/test/jobs-seed'

describe('jobsApi', () => {
  it('lists, filters, reads, adds and removes jobs by the contract', async () => {
    server.use(...jobsServiceHandlers().handlers)
    expect((await jobsApi.list()).total).toBe(4)
    expect((await jobsApi.list({ city: 'Iasi', country: 'Romania' })).items.map(job => job.id)).toEqual([seedId(1)])
    expect((await jobsApi.list({ level: 'Senior' })).items.map(job => job.id)).toEqual([seedId(1)])
    expect((await jobsApi.get(seedId(2))).company).toBe('Bitdefender')
    await expect(jobsApi.get('nope')).rejects.toThrow()
    const draft = await jobsApi.read({ url: 'https://careers.northwind.example/jobs/1' })
    expect(draft).toMatchObject({ title: 'Platform Engineer', host: 'careers.northwind.example', read_from: 'json_ld' })
    const { host: _host, ...input } = draft
    const job = await jobsApi.create({ ...input, work_mode: 'hybrid', level: 'mid', source: 'link' })
    expect(job).toMatchObject({ added_by_me: true, source: 'link' })
    expect((await jobsApi.list()).items[0]!.id).toBe(job.id)
    await jobsApi.remove(job.id)
    await expect(jobsApi.remove(seedId(1))).rejects.toThrow()
  })

  it('reads every documented read_from, and a new one as plain text', () => {
    for (const value of ['', 'json_ld', 'page_text', 'ats_api', 'text', 'something_new']) {
      expect(jobDraftSchema.parse({ title: 'x', read_from: value }).read_from).toBe(value)
    }
    expect(jobDraftSchema.parse({ title: 'x', read_from: null }).read_from).toBe('')
    expect(jobInputSchema.safeParse({
      title: 'x', company: '', city: '', country: '', work_mode: '', level: '', years_required: null, must_have: [], nice_to_have: [],
      responsibilities: [], about: '', url: '', source: 'link', read_from: 'ats_api',
    }).success).toBe(true)
  })

  it('sends only the filters that are set', () => {
    expect(jobSearchParams({}).toString()).toBe('')
    expect(jobSearchParams({ q: ' sql ', city: 'Iași', country: 'Romania', includeRemote: false, areas: ['Data', 'DevOps'], level: 'Mid' }).toString())
      .toBe('q=sql&city=Ia%C8%99i&country=Romania&include_remote=false&area=Data&area=DevOps&level=mid')
    expect(jobSearchParams({ level: 'Any', city: ' ' }).toString()).toBe('')
  })

  it('tells why a page was unreadable, and nothing for other errors', async () => {
    server.use(...jobsServiceHandlers().handlers)
    const failure = (body: object) => jobsApi.read(body as never).catch((error: unknown) => error)
    expect(unreadableReason(await failure({ url: 'https://www.linkedin.com/jobs/view/1' }))).toBe('login_required')
    expect(unreadableReason(await failure({ url: 'https://odd.example.org/x' }))).toBe('unknown')
    expect(unreadableReason(await failure({ text: 'too short' }))).toBeNull()
    expect(unreadableReason(new Error('offline'))).toBeNull()
    server.use(http.post('*/api/jobs/v1/jobs/read', () => HttpResponse.json({ error: { code: 'X' } }, { status: 500 })))
    expect(unreadableReason(await failure({ url: 'https://a.example/b' }))).toBeNull()
  })
})

describe('a job read from an ad, checked before it is saved', () => {
  it('says what is wrong in the app language', () => {
    const bad = { title: ' ', company: '', city: '', country: '', work_mode: '', level: '', years_required: null, must_have: Array(41).fill('Go'),
      nice_to_have: [], responsibilities: [], about: '', url: 'ftp://x', source: 'link', read_from: '' }
    const messages = () => jobInputSchema.safeParse(bad).error!.issues.map(issue => issue.message)
    expect(messages()).toEqual(['Add the job title.', 'At most 40 requirements', 'Use a link starting with http:// or https://'])
    setLocale('ro')
    expect(messages()).toEqual(['Adaugă titlul jobului.', 'Cel mult 40 de cerințe', 'Folosește un link care începe cu http:// sau https://'])
    expect(jobInputSchema.safeParse({ ...bad, title: 'x'.repeat(161), must_have: [], url: '' }).error!.issues[0]!.message).toBe('Cel mult 160 de caractere')
  })
})
