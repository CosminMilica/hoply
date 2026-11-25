import type { Page } from '@playwright/test';
import { createApplicationsBackend } from '../../src/test/applications-service';

/**
 * Answers `/api/applications/v1/...` in the browser context with the in-memory applications backend, which
 * outlives page reloads as the real service does. On the context: the app's mock service worker fetches
 * the requests it passes through.
 */
export async function mockApplicationsService(page: Page) {
  const backend = createApplicationsBackend();
  await page.context().route(url => url.pathname.startsWith('/api/applications/v1/'), async (route) => {
    const request = route.request();
    const [kind, jobId] = new URL(request.url()).pathname.replace(/^\/api\/applications\/v1\//, '').split('/') as [string, string | undefined];
    const method = request.method();
    const answer = kind === 'applications'
      ? method === 'GET' ? backend.list()
        : method === 'PUT' ? backend.put(jobId ?? '', (request.postDataJSON() as { status?: string } | null)?.status ?? '')
          : backend.remove(jobId ?? '')
      : method === 'GET' ? backend.listSkips()
        : method === 'PUT' ? backend.skip(jobId ?? '')
          : jobId ? backend.unskip(jobId) : backend.clearSkips();
    await route.fulfill(answer.body === undefined ? { status: answer.status, body: '' } : { status: answer.status, json: answer.body });
  });
  return backend;
}
