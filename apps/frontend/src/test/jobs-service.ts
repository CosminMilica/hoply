import { http, HttpResponse } from 'msw'
import { CATALOG_JOBS } from './jobs-seed'

type Job = (typeof CATALOG_JOBS)[number]
interface Answer { status: number, body?: unknown }

const plain = (value: string) => value.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase()
const unreadable = (reason: string): Answer => ({
  status: 422,
  body: { error: { code: 'PAGE_UNREADABLE', message: 'The page could not be read', type: 'ValidationError', context: { reason } } },
})
const invalid = (field: string, message: string): Answer => ({
  status: 422,
  body: { error: { code: 'VALIDATION_ERROR', message, type: 'ValidationError', context: { field } } },
})

/** What a link reads as when the rules find a job: the prototype's Northwind ad. */
export const LINK_DRAFT = {
  title: 'Platform Engineer', company: 'Northwind Cloud', city: 'Cluj-Napoca', country: 'Romania', work_mode: 'hybrid', level: 'mid',
  years_required: 4, must_have: ['Kubernetes', 'AWS', 'CI/CD', 'GitHub Actions', 'Terraform', 'Linux', 'Docker', 'Python', 'Go'],
  nice_to_have: ['Helm', 'Kafka', 'PostgreSQL'],
  responsibilities: ['Run and evolve our Kubernetes clusters on AWS.', 'Build CI/CD pipelines with GitHub Actions and Terraform.',
    'Improve monitoring and on-call with Prometheus and Grafana.', 'Write runbooks and help developers ship safely.'],
  about: 'We run the infrastructure behind Northwind payment products and we want releases to be boring.',
  url: 'https://careers.northwind.example/jobs/platform-engineer', source: 'link', read_from: 'json_ld', host: 'careers.northwind.example',
}

/**
 * The jobs service in memory, by docs/api/jobs.md: the catalog, the jobs the person adds, and reading
 * an ad. A link's host decides how reading it fails: linkedin.com, empty., down., huge., or 10.0.0.1.
 */
export function createJobsBackend() {
  const jobs: Job[] = structuredClone(CATALOG_JOBS)
  let sequence = 0
  const find = (id: string) => jobs.find(job => job.id === id)

  return {
    jobs,
    list(params: URLSearchParams): Answer {
      const q = plain(params.get('q') ?? '')
      const city = plain(params.get('city') ?? '')
      const country = plain(params.get('country') ?? '')
      const remote = params.get('include_remote') === 'true'
      const areas = params.getAll('area')
      const level = params.get('level') ?? ''
      const items = jobs.filter((job) => {
        const hay = plain(`${job.title} ${job.company} ${job.must_have.join(' ')} ${job.nice_to_have.join(' ')}`)
        return (!q || hay.includes(q))
          && (!city || plain(job.city) === city || (remote && job.work_mode === 'remote'))
          && (!country || plain(job.country) === country)
          && (!areas.length || job.areas.some(area => areas.includes(area)))
          && (!level || job.level === level)
      }).sort((a, b) => b.created_at.localeCompare(a.created_at))
      return { status: 200, body: { items, total: items.length } }
    },
    get(id: string): Answer {
      const job = find(id)
      return job ? { status: 200, body: job } : { status: 404, body: { error: { code: 'NOT_FOUND', message: 'Job not found' } } }
    },
    read(body: { url?: string, text?: string, title?: string, company?: string }): Answer {
      if (body.text !== undefined) {
        if (body.text.trim().length < 80) return invalid('text', 'text must be at least 80 characters')
        const lines = body.text.split('\n').map(line => line.trim()).filter(Boolean)
        return {
          status: 200,
          body: {
            ...LINK_DRAFT, title: body.title || lines[0]!, company: body.company || '', city: /bra[sș]ov/i.test(body.text) ? 'Brașov' : '',
            work_mode: /hibrid|hybrid/i.test(body.text) ? 'hybrid' : '', must_have: ['Linux', 'Docker', 'Terraform'], nice_to_have: ['Kubernetes'],
            responsibilities: lines.filter(line => line.startsWith('-')).map(line => line.slice(1).trim()), years_required: null, level: '',
            url: body.url ?? '', source: 'text', read_from: 'text', host: '',
          },
        }
      }
      const url = body.url ?? ''
      if (!/^https?:\/\//i.test(url)) return invalid('url', 'url must be an http(s) link')
      const host = url.replace(/^https?:\/\//i, '').split('/')[0]!
      if (/linkedin\.com$|glassdoor\./i.test(host)) return unreadable('login_required')
      if (host.startsWith('empty.')) return unreadable('no_job_data')
      if (host.startsWith('down.')) return unreadable('fetch_failed')
      if (host.startsWith('huge.')) return unreadable('too_large')
      if (/^(10\.|127\.|localhost)/.test(host)) return unreadable('blocked_address')
      if (host.startsWith('odd.')) return unreadable('something_new')
      return { status: 200, body: { ...LINK_DRAFT, url, host } }
    },
    create(body: Record<string, unknown>): Answer {
      if (!String(body.title ?? '').trim()) return invalid('title', 'title is required')
      if (body.source === 'catalog') return invalid('source', 'source must be link or text')
      sequence += 1
      const job = {
        ...(body as unknown as Job), id: `added-${sequence}`, about_company: '', product: '', perks: [], areas: [],
        sample: false, added_by_me: true, created_at: `2026-12-0${sequence}T08:00:00Z`,
      }
      jobs.unshift(job)
      return { status: 201, body: job }
    },
    remove(id: string): Answer {
      const job = find(id)
      if (!job || !job.added_by_me) return { status: 404, body: { error: { code: 'NOT_FOUND', message: 'Job not found' } } }
      jobs.splice(jobs.indexOf(job), 1)
      return { status: 204 }
    },
  }
}

export type JobsBackend = ReturnType<typeof createJobsBackend>

const reply = ({ status, body }: Answer) => (body === undefined ? new HttpResponse(null, { status }) : HttpResponse.json(body as object, { status }))

/** The jobs backend behind MSW, for unit and integration tests. */
export function jobsServiceHandlers(backend: JobsBackend = createJobsBackend()) {
  const BASE = '*/api/jobs/v1/jobs'
  return {
    backend,
    handlers: [
      http.get(BASE, ({ request }) => reply(backend.list(new URL(request.url).searchParams))),
      http.post(`${BASE}/read`, async ({ request }) => reply(backend.read(await request.json() as never))),
      http.get(`${BASE}/:id`, ({ params }) => reply(backend.get(String(params.id)))),
      http.post(BASE, async ({ request }) => reply(backend.create(await request.json() as never))),
      http.delete(`${BASE}/:id`, ({ params }) => reply(backend.remove(String(params.id)))),
    ],
  }
}
