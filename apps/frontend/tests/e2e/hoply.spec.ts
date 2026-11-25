import { expect, test } from '@playwright/test';
import { saveCoverage } from './coverage';
import { mockApplicationsService } from './applications';
import { mockCvService } from './cv';
import { mockJobsService } from './jobs';
import { mockProfileService } from './profile';

test.afterEach(async ({ page }, testInfo) => saveCoverage(page, testInfo));

test('a phone edits the profile with the tab bar at hand', async ({ page }) => {
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.setViewportSize({ width: 360, height: 780 });
  const profile = await mockProfileService(page);
  await mockJobsService(page);
  await mockApplicationsService(page);
  await page.goto('/');
  await expect(page.getByText('Ready to tailor')).toBeVisible();
  await expect(page.getByRole('heading', { level: 1, name: 'Andrei Popescu' })).toBeVisible();
  await expect(page.locator('[data-test="avatar"]')).toHaveText('AP');
  await expect(page.getByText('Next up: add a GitHub or LinkedIn link')).toHaveCount(0);
  await page.getByLabel('Add a skill').fill('Terraform');
  await page.getByRole('button', { name: 'Add', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Remove Terraform' })).toBeVisible();
  expect(profile.skills).toContain('Terraform');
  await page.getByRole('button', { name: /Contact data/ }).click();
  await page.getByLabel('GitHub').fill('');
  await page.getByLabel('GitHub').blur();
  await expect(page.getByText('Next up: add a GitHub or LinkedIn link')).toBeVisible();
  const scrollWidth = await page.evaluate(() => document.documentElement.scrollWidth);
  expect(scrollWidth).toBeLessThanOrEqual(360);
  await expect(page.locator('[data-shell="tabs"]')).toBeVisible();
  expect(errors).toEqual([]);
});

test('from Find jobs to a tailored CV for an added job, marked as applied', async ({ page }) => {
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  const profile = await mockProfileService(page);
  const jobs = await mockJobsService(page);
  await mockCvService(page, profile, jobs);
  await mockApplicationsService(page);
  await page.goto('/jobs');
  await expect(page.getByRole('heading', { name: 'Finding jobs is coming soon' })).toBeVisible();
  await page.getByRole('link', { name: /Add a job from a link or its text/ }).click();
  await page.getByLabel('Link to the job ad').fill('https://careers.northwind.example/jobs/platform-engineer');
  await page.getByRole('button', { name: 'Read the job' }).click();
  await expect(page.getByText('Check what we found')).toBeVisible();
  await page.getByRole('button', { name: 'Save and prepare my CV' }).click();
  await expect(page.getByRole('heading', { name: 'Prepare your CV' })).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Main' }).getByText('1', { exact: true })).toBeVisible();

  await page.getByLabel('Your experience with Kubernetes').fill('Ran a Kubernetes cluster with 30 services');
  await page.locator('[data-test="gap"]').filter({ hasText: 'Kubernetes' }).getByRole('button', { name: 'Add to my CV' }).click();
  await expect(page.locator('[data-test="match"]')).toHaveText('38% → 48%');
  expect(profile.experience[0]!.notes.at(-1)).toEqual({ text: 'Ran a Kubernetes cluster with 30 services', skills: ['Kubernetes'] });
  await page.getByRole('button', { name: 'Generate CV · 48% match' }).click();
  await expect(page.getByRole('heading', { name: 'Tailoring your CV' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Your CV is ready' })).toBeVisible({ timeout: 10_000 });
  await expect(page.getByRole('button', { name: 'Tailored' })).toHaveAttribute('aria-pressed', 'true');
  await expect(page.getByText('ATS check')).toBeVisible();
  await expect(page.locator('[data-test="cv-name"]')).toHaveText('Andrei Popescu');
  await expect(page.getByText('Andrei_Popescu_CV_Northwind_Cloud.pdf')).toBeVisible();
  await expect(page.getByRole('article', { name: 'CV preview' })).toContainText('Looking to join Northwind Cloud as Platform Engineer');
  await page.getByLabel("Recruiter's name (optional)").fill('Ana Pop');
  await page.getByRole('button', { name: 'Write them' }).click();
  await expect(page.getByRole('textbox', { name: 'Cover letter' })).toHaveValue(/^Dear Ana Pop,/);
  await page.getByRole('button', { name: 'LinkedIn note' }).click();
  await expect(page.getByRole('textbox', { name: 'LinkedIn note' })).toHaveValue(/^Hi Ana, I applied for the Platform Engineer role/);
  await page.getByRole('button', { name: 'I applied to this job' }).click();
  await expect(page.getByRole('button', { name: 'Marked as applied ✓' })).toBeVisible();

  await page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: /Saved/ }).click();
  await page.getByRole('button', { name: 'Applied 1' }).click();
  await expect(page.getByRole('heading', { name: 'Platform Engineer' })).toBeVisible();
  expect(errors).toEqual([]);
});

test('a role and an education entry are edited in place', async ({ page }) => {
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  const profile = await mockProfileService(page);
  await mockJobsService(page);
  await mockApplicationsService(page);
  await page.goto('/profile');
  await page.getByRole('button', { name: 'Edit DevOps Engineer' }).click();
  const roleForm = page.getByRole('form', { name: 'Edit DevOps Engineer' });
  await expect(roleForm.getByLabel('Company', { exact: true })).toHaveValue('Verde Cloud');
  await roleForm.getByLabel('Role', { exact: true }).fill('Platform Engineer');
  await roleForm.getByRole('button', { name: '+ Add a highlight' }).click();
  await roleForm.getByLabel('Highlight 4', { exact: true }).fill('Cut cloud costs by 30%');
  await roleForm.getByRole('button', { name: 'Save' }).click();
  await expect(roleForm).toHaveCount(0);
  await expect(page.getByText('Cut cloud costs by 30%')).toBeVisible();
  expect(profile.experience[0]).toMatchObject({ id: 'r1', role: 'Platform Engineer' });

  await page.getByRole('button', { name: /Education/ }).click();
  await page.getByRole('button', { name: 'Edit BSc Computer Science' }).click();
  const educationForm = page.getByRole('form', { name: 'Edit BSc Computer Science' });
  await expect(educationForm.getByLabel('School', { exact: true })).toHaveValue('University of Iași');
  await educationForm.getByLabel('Degree title', { exact: true }).fill('BSc Informatics');
  await educationForm.getByRole('button', { name: 'Save' }).click();
  await expect(page.getByText('BSc Informatics')).toBeVisible();
  expect(profile.education[0]).toMatchObject({ id: 'e1', degree: 'BSc Informatics' });
  expect(errors).toEqual([]);
});

test('work preferences, a project and a certification are saved from the profile', async ({ page }) => {
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  const profile = await mockProfileService(page);
  await mockJobsService(page);
  await mockApplicationsService(page);
  await page.goto('/profile');
  await page.getByRole('button', { name: /Work preferences/ }).click();
  await page.getByRole('button', { name: 'Remote', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Remote', exact: true })).toHaveAttribute('aria-pressed', 'true');
  await page.getByLabel('Available from', { exact: true }).fill('2027-02');
  await page.getByLabel('Available from', { exact: true }).blur();
  await expect.poll(() => profile.available_from).toBe('2027-02');
  expect(profile.work_modes).toEqual(['hybrid', 'remote']);

  await page.getByRole('button', { name: /Projects/ }).click();
  await page.getByRole('button', { name: '+ Add a project' }).click();
  const projectForm = page.getByRole('form', { name: 'Add a project' });
  await projectForm.getByLabel('Project name', { exact: true }).fill('Home lab');
  await projectForm.getByLabel('Highlight 1', { exact: true }).fill('Ran 12 containers on one host');
  await projectForm.getByLabel('Result of highlight 1', { exact: true }).fill('kept uptime above 99.9%');
  await projectForm.getByRole('button', { name: 'Save' }).click();
  await expect(page.getByText('kept uptime above 99.9%')).toBeVisible();

  await page.getByRole('button', { name: /Certifications/ }).click();
  await page.getByRole('button', { name: '+ Add a certification' }).click();
  const certificationForm = page.getByRole('form', { name: 'Add a certification' });
  await certificationForm.getByLabel('Certification', { exact: true }).fill('Terraform Associate');
  await certificationForm.getByLabel('Issued', { exact: true }).fill('2025-04');
  await certificationForm.getByRole('button', { name: 'Save' }).click();
  await expect(page.getByText('issued 04/2025')).toBeVisible();
  expect(profile.certifications[0]).toMatchObject({ name: 'Terraform Associate', issued: '2025-04' });
  expect(errors).toEqual([]);
});

test('a job read from pasted text after its link needed a login, then prepared', async ({ page }) => {
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.setViewportSize({ width: 390, height: 860 });
  const profile = await mockProfileService(page);
  const jobs = await mockJobsService(page);
  await mockCvService(page, profile, jobs);
  await mockApplicationsService(page);
  await page.goto('/saved');
  await page.getByRole('link', { name: 'Add a job', exact: true }).first().click();
  await expect(page.locator('[data-shell="tabs"]').getByRole('link', { name: /Saved/ })).toHaveAttribute('aria-current', 'page');
  await page.getByLabel('Link to the job ad').fill('https://www.linkedin.com/jobs/view/42');
  await page.getByRole('button', { name: 'Read the job' }).click();
  await expect(page.getByText('LinkedIn shows job pages only to signed-in members')).toBeVisible();
  await page.getByRole('button', { name: 'Paste the text instead' }).click();
  await expect(page.getByLabel('Link to the ad (optional, to find it again)')).toHaveValue('https://www.linkedin.com/jobs/view/42');
  await page.getByLabel('The job ad').fill([
    'DevOps Engineer', 'Firma: Carpathia Systems, Brașov (hibrid)', 'Responsabilități', '- Administrezi servere Linux și containere Docker.',
    '- Automatizezi infrastructura cu Terraform.', 'Cerințe', '- Minimum 3 ani experiență în DevOps.',
  ].join('\n'));
  await page.getByRole('button', { name: 'Read the job' }).click();
  await expect(page.getByText('Read from the text you pasted')).toBeVisible();
  await expect(page.getByRole('note')).toContainText('Based in Brașov, hybrid. You live in Iași');
  await page.getByRole('button', { name: 'Save and prepare my CV' }).click();
  await expect(page.getByRole('heading', { name: 'Prepare your CV' })).toBeVisible();
  await expect(page.getByText(/^DevOps Engineer ·/)).toBeVisible();
  expect(jobs.jobs[0]).toMatchObject({ source: 'text', url: 'https://www.linkedin.com/jobs/view/42', added_by_me: true });
  expect(errors).toEqual([]);
});

test('a saved job is still in Saved after a reload', async ({ page }) => {
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  await mockProfileService(page);
  await mockJobsService(page);
  const applications = await mockApplicationsService(page);
  await page.goto('/jobs/add?from=saved');
  await page.getByLabel('Link to the job ad').fill('https://careers.northwind.example/jobs/platform-engineer');
  await page.getByRole('button', { name: 'Read the job' }).click();
  await page.getByRole('button', { name: 'Save to my jobs' }).click();
  await expect(page.getByRole('heading', { name: 'Platform Engineer' })).toBeVisible();
  await page.reload();
  await expect(page.getByRole('heading', { name: 'Platform Engineer' })).toBeVisible();
  await expect(page.locator('[data-test="since"]')).toHaveText(/^Saved \d{1,2} [A-Z][a-z]{2}$/);
  await expect(page.getByRole('button', { name: 'Saved 1' })).toBeVisible();
  expect([...applications.applications.values()].map(item => item.status)).toEqual(['saved']);
  expect(errors).toEqual([]);
});
