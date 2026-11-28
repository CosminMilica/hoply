import { expect, type Page, type TestInfo } from '@playwright/test';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';

/** Saves the page's Istanbul counters for the instrumented run (FORGE_BROWSER_COVERAGE). */
export async function saveCoverage(page: Page, testInfo: TestInfo) {
  const directory = process.env.FORGE_BROWSER_COVERAGE;
  if (!directory) return;
  const coverage = await page.evaluate(() => (window as any).__coverage__);
  expect(coverage, 'Browser instrumentation must produce real coverage').toBeTruthy();
  await mkdir(directory, { recursive: true });
  await writeFile(path.join(directory, `crashr-${testInfo.parallelIndex}-${Date.now()}.json`), JSON.stringify(coverage));
}
