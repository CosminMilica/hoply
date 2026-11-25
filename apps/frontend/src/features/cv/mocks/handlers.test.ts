import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { cvApi } from '../api/cvApi'
import type { CvRequest } from '../api/cv.schema'
import { sampleProfile } from '@/test/profile-service'
import { engineJob } from '@/test/jobs'
import { CATALOG_JOBS } from '@/test/jobs-seed'
import { createCvHandlers, loadJobFromService, loadProfileFromService } from './handlers'
import { seedId } from '@/test/jobs-seed'

const loadJob = async (_: Request, id: string) => {
  const index = CATALOG_JOBS.findIndex(job => job.id === id)
  return index >= 0 ? engineJob(index) : null
}

const request = (overrides: Partial<CvRequest> = {}): CvRequest => ({ job_id: seedId(1), skip_roles: [], focus: [], confirmed: [], ...overrides })

describe('the mocked cv service', () => {
  it('asks the questions for a job from the profile it reads', async () => {
    server.use(...createCvHandlers({ loadJob, loadProfile: async () => sampleProfile() }))
    const questions = await cvApi.questions(seedId(2))
    expect(questions.job_id).toBe(seedId(2))
    expect(questions.gaps.map(gap => gap.skill)).toEqual(['Kubernetes', 'Terraform', 'AWS'])
  })

  it('makes a CV, keeps it, and lists the CVs of a job newest first', async () => {
    let headline = 'First'
    server.use(...createCvHandlers({ loadJob, loadProfile: async () => sampleProfile({ headline }), now: () => new Date('2026-11-25T10:00:00Z') }))
    const first = await cvApi.generate(request())
    headline = 'Second'
    const second = await cvApi.generate(request())
    await cvApi.generate(request({ job_id: seedId(3) }))
    expect(first).toMatchObject({ id: 'cv-1', job_id: seedId(1), created_at: '2026-11-25T10:00:00.000Z' })
    expect(first.profile.headline).toBe('First')
    expect(first.snapshot.headline).toBe('First')
    const items = await cvApi.forJob(seedId(1))
    expect(items.map(cv => cv.id)).toEqual([second.id, first.id])
    const one = await fetch(`${window.location.origin}/api/cv/v1/cvs/${first.id}`).then(response => response.json())
    expect(one.profile.headline).toBe('First')
    expect((await fetch(`${window.location.origin}/api/cv/v1/cvs`).then(r => r.json())).items).toHaveLength(3)
  })

  it('answers 404 for an unknown job or CV, 422 for a bad request and 502 without a profile', async () => {
    server.use(...createCvHandlers({ loadJob, loadProfile: async () => { throw new Error('down') } }))
    const post = (body: unknown) => fetch(`${window.location.origin}/api/cv/v1/cvs`, { method: 'POST', body: JSON.stringify(body) })
    expect((await post({ job_id: 1 })).status).toBe(422)
    expect((await post(request({ job_id: 'nope' }))).status).toBe(404)
    expect((await post(request())).status).toBe(502)
    expect((await fetch(`${window.location.origin}/api/cv/v1/jobs/nope/questions`)).status).toBe(404)
    expect((await fetch(`${window.location.origin}/api/cv/v1/jobs/${seedId(1)}/questions`)).status).toBe(502)
    expect((await fetch(`${window.location.origin}/api/cv/v1/cvs/none`)).status).toBe(404)
  })

  it('answers 502 when the jobs service cannot be read', async () => {
    server.use(...createCvHandlers({ loadJob: async () => { throw new Error('down') }, loadProfile: async () => sampleProfile() }))
    expect((await fetch(`${window.location.origin}/api/cv/v1/jobs/${seedId(1)}/questions`)).status).toBe(502)
    const post = await fetch(`${window.location.origin}/api/cv/v1/cvs`, { method: 'POST', body: JSON.stringify(request()) })
    expect(post.status).toBe(502)
  })

  it('reads a job from the jobs service, past the mocks', async () => {
    const original = globalThis.fetch
    const answers: Record<string, Response> = {
      [`/api/jobs/v1/jobs/${seedId(1)}`]: HttpResponse.json(CATALOG_JOBS[0]),
      '/api/jobs/v1/jobs/9': new Response(null, { status: 404 }),
      '/api/jobs/v1/jobs/x': new Response(null, { status: 500 }),
    }
    globalThis.fetch = (async (input: RequestInfo | URL) => answers[new URL(input instanceof Request ? input.url : String(input)).pathname]!) as typeof fetch
    try {
      const request = new Request('http://localhost/api/cv/v1/cvs')
      const job = await loadJobFromService(request, seedId(1))
      expect(job).toMatchObject({ title: 'Snowflake Administrator DevOps', keywords: CATALOG_JOBS[0]!.must_have, key_responsibilities: [0, 1, 2, 3] })
      expect(await loadJobFromService(request, '9')).toBeNull()
      await expect(loadJobFromService(request, 'x')).rejects.toThrow('500')
    }
    finally {
      globalThis.fetch = original
    }
  })

  it('reads the profile from the profile service, past the mocks', async () => {
    server.use(http.get('*/api/profile/v1/profiles/me', () => HttpResponse.json(sampleProfile())))
    // bypass() sends the request to the network: here the test's handler stands in for it.
    const original = globalThis.fetch
    globalThis.fetch = (async (input: RequestInfo | URL) => {
      const url = input instanceof Request ? input.url : String(input)
      if (url.endsWith('/api/profile/v1/profiles/me')) return HttpResponse.json(sampleProfile({ headline: 'From service' }))
      return original(input)
    }) as typeof fetch
    try {
      const profile = await loadProfileFromService(new Request('http://localhost/api/cv/v1/cvs'))
      expect(profile.headline).toBe('From service')
      globalThis.fetch = (async () => new Response(null, { status: 503 })) as unknown as typeof fetch
      await expect(loadProfileFromService(new Request('http://localhost/api/cv/v1/cvs'))).rejects.toThrow('503')
    }
    finally {
      globalThis.fetch = original
    }
  })
})
