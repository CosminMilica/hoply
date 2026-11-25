import type { Page, Route } from '@playwright/test';

const bullet = (text: string, result = '', skills: string[] = []) => ({ text, result, skills });

/** A profile as docs/api/profile.md describes it: the person of the approved prototype. */
export function sampleProfile() {
  return {
    id: 'p-1',
    name: 'Andrei Popescu',
    headline: 'DevOps Engineer',
    about: 'DevOps engineer who automates infrastructure and keeps services running. I like reliable systems, clear documentation and calm on-call weeks.',
    email: 'andrei@example.com',
    phone: '+40 700 000 000',
    github: 'github.com/andrei-popescu',
    linkedin: '',
    city: 'Iași',
    country: 'Romania',
    work_modes: ['hybrid'] as string[],
    open_to_relocation: false,
    work_authorization: 'citizen',
    work_authorization_region: 'EU',
    notice_period_weeks: 4 as number | null,
    available_from: '',
    target_roles: [] as string[],
    target_seniority: [] as string[],
    employment_types: ['full_time'] as string[],
    salary_min: null as number | null,
    salary_currency: '',
    salary_period: '',
    additional_info: [] as string[],
    skills: ['Python', 'SQL', 'Git', 'Linux', 'Docker', 'CI/CD', 'Bash', 'Monitoring'],
    experience: [
      {
        id: 'r1', role: 'DevOps Engineer', company: 'Verde Cloud', location: 'Iași', employment_type: 'full_time', industry: '',
        team_size: null, people_managed: null, start: '2022-03', end: '',
        tools: ['Linux', 'Docker', 'CI/CD', 'Python', 'Git', 'Monitoring', 'SQL'],
        bullets: [
          bullet('Built and maintained deployment pipelines for internal services.', 'cut release time from 2 hours to 15 minutes'),
          bullet('Automated server setup and configuration with scripts.'),
          bullet('Monitored services and responded to incidents on call.'),
        ],
      },
    ],
    projects: [] as Record<string, unknown>[],
    education: [{ id: 'e1', level: 'bachelor', field: 'Computer Science', degree: 'BSc Computer Science', school: 'University of Iași', start_year: '2016', end_year: '2020' }],
    certifications: [] as Record<string, unknown>[],
    languages: [{ id: 'l1', name: 'Romanian', level: 'Native' }],
    created_at: '2026-11-25T00:00:00Z',
    updated_at: '2026-11-25T00:00:00Z',
  };
}

type Profile = ReturnType<typeof sampleProfile>;
const LISTS = ['experience', 'projects', 'education', 'certifications', 'languages'] as const;
const NEWEST_FIRST = ['experience', 'projects'];

/**
 * Answers `/api/profile/v1/profiles/me...` in the browser context, keeping the profile in memory.
 * On the context, not the page: the app's mock service worker fetches the requests it passes through.
 */
export async function mockProfileService(page: Page, profile: Profile = sampleProfile()) {
  let sequence = 0;
  const handle = async (route: Route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname.replace(/^\/api\/profile\/v1\/profiles\/me/, '');
    const method = request.method();
    const body = request.postDataJSON?.() ?? null;
    if (method === 'GET' && path === '') return route.fulfill({ json: profile });
    if (method === 'PATCH' && path === '') {
      Object.assign(profile, body);
      return route.fulfill({ json: profile });
    }
    if (method === 'PUT' && path === '/skills') {
      profile.skills = body.skills;
      return route.fulfill({ json: profile });
    }
    const [, collection, itemId] = path.split('/') as [string, (typeof LISTS)[number], string | undefined];
    if (!LISTS.includes(collection)) return route.fulfill({ status: 404, json: { message: 'Not found' } });
    const list = profile[collection] as { id: string }[];
    if (method === 'POST' && !itemId) {
      const item = { id: `n${(sequence += 1)}`, ...body };
      if (NEWEST_FIRST.includes(collection)) list.unshift(item);
      else list.push(item);
      return route.fulfill({ status: 201, json: item });
    }
    const at = list.findIndex(item => item.id === itemId);
    if (at < 0) return route.fulfill({ status: 404, json: { message: 'Not found' } });
    if (method === 'PUT') {
      const item = { ...body, id: itemId };
      list.splice(at, 1, item);
      return route.fulfill({ json: item });
    }
    if (method === 'DELETE') {
      list.splice(at, 1);
      return route.fulfill({ status: 204, body: '' });
    }
    return route.fulfill({ status: 405, json: { message: 'Method not allowed' } });
  };
  await page.context().route(url => url.pathname.startsWith('/api/profile/v1/profiles/me'), handle);
  return profile;
}
