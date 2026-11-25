import type { Page } from '@playwright/test';
import type { Profile } from '../../src/features/profile/api/profile.schema';
import { createCvBackend } from '../../src/test/cv-service';

/**
 * Answers `/api/cv/v1/...` in the browser context with the in-memory cv service, which reads the profile and
 * the jobs from the other in-memory services of the test. On the context: the app's mock service worker fetches
 * the requests it passes through.
 */
export async function mockCvService(page: Page, profile: object, jobs: { jobs: { id: string }[] }) {
  const backend = createCvBackend({
    profile: () => profile as Profile,
    job: id => jobs.jobs.find(job => job.id === id),
  }, { now: () => new Date() });
  await page.context().route(url => url.pathname.startsWith('/api/cv/v1/'), async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const [kind, id, rest] = url.pathname.replace(/^\/api\/cv\/v1\//, '').split('/') as [string, string | undefined, string | undefined];
    const method = request.method();
    const answer = kind === 'jobs' && rest === 'questions' ? backend.questions(id ?? '')
      : kind === 'jobs' && rest === 'probe' ? backend.probe(id ?? '')
      : method === 'POST' ? backend.create(request.postDataJSON())
        : method === 'DELETE' ? backend.remove(id ?? '')
          : id ? backend.get(id) : backend.list(url.searchParams.get('job_id'));
    await route.fulfill(answer.body === undefined ? { status: answer.status, body: '' } : { status: answer.status, json: answer.body });
  });
  return backend;
}
