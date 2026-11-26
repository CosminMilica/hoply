import type { Page } from '@playwright/test';
import { createJobsBackend } from '../../src/test/jobs-service';

/**
 * Answers `/api/jobs/v1/jobs...` and `/api/jobs/v1/boards...` in the browser context with the in-memory jobs backend.
 * On the context, not the page: the app's mock service worker fetches the requests it passes through.
 */
export async function mockJobsService(page: Page) {
  const backend = createJobsBackend();
  await page.context().route(url => url.pathname.startsWith('/api/jobs/v1/jobs'), async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const rest = url.pathname.replace(/^\/api\/jobs\/v1\/jobs\/?/, '');
    const method = request.method();
    const answer = method === 'GET' && !rest ? backend.list(url.searchParams)
      : method === 'GET' ? backend.get(rest)
        : method === 'POST' && rest === 'read' ? backend.read(request.postDataJSON())
          : method === 'POST' ? backend.create(request.postDataJSON())
            : backend.remove(rest);
    await route.fulfill(answer.body === undefined ? { status: answer.status, body: '' } : { status: answer.status, json: answer.body });
  });
  await page.context().route(url => url.pathname.startsWith('/api/jobs/v1/boards'), async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const answer = request.method() === 'GET' ? backend.boardOf(url.searchParams.get('url') ?? '') : backend.follow(request.postDataJSON());
    await route.fulfill({ status: answer.status, json: answer.body });
  });
  return backend;
}
