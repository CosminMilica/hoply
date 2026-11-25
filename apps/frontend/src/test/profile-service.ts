import { http, HttpResponse } from 'msw'
import type { Bullet, Profile } from '@/features/profile/api/profile.schema'

const bullet = (text: string, result = '', skills: string[] = []): Bullet => ({ text, result, skills })

/** The prototype's person, as the profile service returns it. */
export function sampleProfile(overrides: Partial<Profile> = {}): Profile {
  return {
    id: 'p-1',
    name: 'Andrei Popescu',
    headline: 'DevOps Engineer',
    about: 'DevOps engineer who automates infrastructure and keeps services running. I like reliable systems, clear documentation and calm on-call weeks.',
    email: 'andrei@example.com',
    phone: '+40 700 000 000',
    github: 'github.com/andrei-popescu',
    linkedin: 'linkedin.com/in/andrei-popescu',
    city: 'Iași',
    country: 'Romania',
    work_modes: ['remote', 'hybrid'],
    open_to_relocation: false,
    work_authorization: 'citizen',
    work_authorization_region: 'EU',
    notice_period_weeks: 4,
    available_from: '',
    target_roles: ['DevOps Engineer'],
    target_seniority: ['mid'],
    employment_types: ['full_time'],
    salary_min: null,
    salary_currency: '',
    salary_period: '',
    additional_info: [],
    notes: [],
    skills: ['Python', 'SQL', 'Git', 'Linux', 'Docker', 'CI/CD', 'Bash', 'Monitoring'],
    experience: [
      {
        id: 'r1', role: 'DevOps Engineer', company: 'Verde Cloud', location: 'Iași', employment_type: 'full_time', industry: 'Cloud',
        team_size: 6, people_managed: null, start: '2022-03', end: '',
        tools: ['Linux', 'Docker', 'CI/CD', 'Python', 'Git', 'Monitoring', 'SQL'],
        bullets: [
          bullet('Built and maintained deployment pipelines for internal services.'),
          bullet('Automated server setup and configuration with scripts.'),
          bullet('Monitored services and responded to incidents on call.'),
          bullet('Wrote SQL queries and reports for the data team.'),
        ],
        notes: [],
      },
      {
        id: 'r2', role: 'Junior system administrator', company: 'Bonbon Labs', location: 'Iași', employment_type: '', industry: '',
        team_size: null, people_managed: null, start: '2020-09', end: '2022-02',
        tools: ['Linux', 'Bash', 'Monitoring'],
        bullets: [
          bullet('Maintained servers and user accounts for a small office.'),
          bullet('Set up backups and checked that they ran every night.', 'restored every test backup within an hour'),
          bullet('Fixed issues reported by users and kept a log of recurring problems.'),
        ],
        notes: [],
      },
    ],
    projects: [],
    education: [{ id: 'e1', level: 'bachelor', field: 'Computer Science', degree: 'BSc Computer Science', school: 'University of Iași', start_year: '2016', end_year: '2020' }],
    certifications: [],
    languages: [{ id: 'l1', name: 'Romanian', level: 'Native' }, { id: 'l2', name: 'English', level: 'C1' }],
    created_at: '2026-11-25T00:00:00Z',
    updated_at: '2026-11-25T00:00:00Z',
    ...overrides,
  }
}

export function emptyProfile(): Profile {
  return sampleProfile({
    name: '', headline: '', about: '', email: '', phone: '', github: '', linkedin: '', city: '', country: '',
    work_modes: [], work_authorization: '', work_authorization_region: '', notice_period_weeks: null,
    target_roles: [], target_seniority: [], employment_types: [],
    skills: [], experience: [], projects: [], education: [], certifications: [], languages: [],
  })
}

type ListKey = 'experience' | 'projects' | 'education' | 'certifications' | 'languages'
const NEWEST_FIRST: ListKey[] = ['experience', 'projects']

/** An in-memory profile service for tests, by docs/api/profile.md. */
export function profileServiceHandlers(profile: Profile = sampleProfile()) {
  let sequence = 0
  const ME = '*/api/profile/v1/profiles/me'
  const items = (key: string) => profile[key as ListKey] as { id: string }[]
  const notFound = () => HttpResponse.json({ message: 'Not found' }, { status: 404 })
  return {
    profile,
    handlers: [
      http.get(ME, () => HttpResponse.json(profile)),
      http.patch(ME, async ({ request }) => {
        Object.assign(profile, await request.json())
        return HttpResponse.json(profile)
      }),
      http.put(`${ME}/skills`, async ({ request }) => {
        profile.skills = ((await request.json()) as { skills: string[] }).skills
        return HttpResponse.json(profile)
      }),
      http.post(`${ME}/:collection`, async ({ params, request }) => {
        const key = params.collection as ListKey
        const item = { id: `n${(sequence += 1)}`, ...(await request.json() as object) }
        if (NEWEST_FIRST.includes(key)) items(key).unshift(item)
        else items(key).push(item)
        return HttpResponse.json(item, { status: 201 })
      }),
      http.put(`${ME}/:collection/:id`, async ({ params, request }) => {
        const list = items(String(params.collection))
        const at = list.findIndex(item => item.id === params.id)
        if (at < 0) return notFound()
        const item = { ...(await request.json() as object), id: String(params.id) }
        list.splice(at, 1, item)
        return HttpResponse.json(item)
      }),
      http.delete(`${ME}/:collection/:id`, ({ params }) => {
        const list = items(String(params.collection))
        const at = list.findIndex(item => item.id === params.id)
        if (at < 0) return notFound()
        list.splice(at, 1)
        return new HttpResponse(null, { status: 204 })
      }),
    ],
  }
}
