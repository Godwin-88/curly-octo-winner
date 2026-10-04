import { expect, request, test } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

/**
 * Registering a school from nothing, against the local stack:
 *
 *   docker compose up --build        (repo root)
 *   E2E_STACK=1 npx playwright test e2e/onboarding.spec.ts
 *
 * The stack's stand-in for Supabase Auth keeps the accounts created here in
 * memory, so a new user really signs in with the password they were shown.
 * Before a second run within 15 minutes:
 *   docker compose exec redis redis-cli FLUSHALL
 */
const STACK = Boolean(process.env.E2E_STACK);
const BASE_URL = `http://localhost:${process.env.E2E_PORT || 3100}`;

test.describe.configure({ mode: 'serial' });
test.beforeEach(() => {
  test.skip(!STACK, 'Set E2E_STACK=1 with the docker compose stack running');
});

const tag = Date.now().toString(36);
const SCHOOL = `Riverside Academy ${tag}`;
const PRINCIPAL = `principal-${tag}@example.test`;
const TEACHER = `teacher-${tag}@example.test`;
const PASSWORD = 'a-long-password-1';

async function signInStatus(email: string, password: string) {
  const api = await request.newContext({ baseURL: BASE_URL });
  const response = await api.post('/api/v1/login', { data: { email, password } });
  const state = await api.storageState();
  await api.dispose();
  return { status: response.status(), cookies: state.cookies };
}

test('a school is registered in three steps and lands on what to set up', async ({ page }) => {
  await page.goto('/auth/login');
  await page.getByRole('link', { name: 'Register your school' }).click();
  await expect(page.getByRole('heading', { name: 'Register your school' })).toBeVisible();

  // Nothing moves on until the step in view is complete.
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(page.getByText('Enter the school’s full name.')).toBeVisible();
  await expect(page.getByText('Choose the county the school is in.')).toBeVisible();
  await expect(page.getByText('Say whether the school is public or private.')).toBeVisible();

  await page.getByLabel('School name').fill(SCHOOL);
  await page.getByRole('radio', { name: /Public/ }).check();
  await page.getByLabel('County').selectOption('Nakuru');
  await page.getByLabel('School phone (optional)').fill('0712 345 678');
  const accessibility = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa']).analyze();
  expect(accessibility.violations.map((violation) => `${violation.id}: ${violation.help}`)).toEqual([]);
  await page.getByRole('button', { name: 'Continue' }).click();

  await page.getByLabel('Your full name').fill('Mary Wanjiku');
  await page.getByLabel('Your email').fill(PRINCIPAL);
  await page.getByLabel('Choose a password').fill(PASSWORD);
  await page.getByLabel('Type the password again').fill('something-else-1');
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(page.getByText('The two passwords are not the same.')).toBeVisible();
  await page.getByLabel('Type the password again').fill(PASSWORD);
  await page.getByRole('button', { name: 'Continue' }).click();

  await page.getByLabel('Current term').selectOption('3');
  await expect(page.getByText(`${SCHOOL}, a public school in Nakuru`)).toBeVisible();
  await page.getByRole('button', { name: 'Create the school' }).click();

  // Signed in to the new school, on its checklist.
  await expect(page).toHaveURL(/\/w\/all\/[0-9a-f-]{36}\/school\/setup$/, { timeout: 20_000 });
  await expect(page.getByRole('navigation', { name: 'Context' })).toContainText(SCHOOL);
  await expect(page.getByRole('heading', { name: 'Getting started' })).toBeVisible();
  await expect(page.getByText(/0 learners and 0 parents recorded/)).toBeVisible();
  await expect(page.getByText(/1 person can sign in/)).toBeVisible();

  // A public school starts with levies, not tuition; the list is its own to change.
  await page.goto('/finance/fee-items');
  await expect(page.getByRole('link', { name: 'Lunch programme' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Tuition', exact: true })).toHaveCount(0);

  // The dashboard is this school's: its own name and county, not the pilot's.
  await page.goto('/dashboard');
  const dashboard = page.getByRole('main');
  await expect(dashboard.getByText(`School overview · ${SCHOOL}`)).toBeVisible();
  await expect(dashboard.getByText('Nakuru County')).toBeVisible();
  await expect(dashboard).not.toContainText('Jua Kali');
  await expect(dashboard).not.toContainText('Kasarani');
});

test('the principal adds a teacher, who signs in; deactivating them signs them out at once', async ({ page }) => {
  const principal = await signInStatus(PRINCIPAL, PASSWORD);
  expect(principal.status).toBe(200);
  await page.context().addCookies(principal.cookies);

  await page.goto('/school/users');
  await page.getByRole('link', { name: 'Add user' }).click();
  const panel = page.getByRole('region', { name: 'user detail' });
  await panel.getByRole('textbox', { name: 'Full name' }).fill('Tom Teacher');
  await panel.getByRole('textbox', { name: 'Email' }).fill(TEACHER);
  await panel.getByRole('button', { name: 'Add user' }).click();

  // The password is shown once, here.
  const shown = panel.getByRole('status').filter({ hasText: 'Sign-in details for Tom Teacher' });
  await expect(shown).toBeVisible();
  const password = (await shown.locator('code').nth(1).innerText()).trim();
  expect(password).toMatch(/^[A-Za-z0-9]{4}(-[A-Za-z0-9]{4}){3}$/);
  await shown.getByRole('button', { name: 'I have saved it' }).click();
  await expect(panel.locator('code')).toHaveCount(0);

  // The same email cannot be added twice.
  await page.getByRole('link', { name: 'Add user' }).click();
  await panel.getByRole('textbox', { name: 'Full name' }).fill('Tom Again');
  await panel.getByRole('textbox', { name: 'Email' }).fill(TEACHER);
  await panel.getByRole('button', { name: 'Add user' }).click();
  await expect(panel.getByRole('alert')).toContainText('already signs in to');

  // The teacher signs in with what was shown, as a teacher of this school.
  const teacher = await signInStatus(TEACHER, password);
  expect(teacher.status).toBe(200);
  const asTeacher = await request.newContext({ baseURL: BASE_URL, storageState: { cookies: teacher.cookies, origins: [] } });
  expect((await asTeacher.get('/api/v1/learners')).status()).toBe(200);
  // A teacher does not manage users.
  expect((await asTeacher.get('/api/v1/school/users')).status()).toBe(403);

  await page.goto('/school/users');
  await page.getByRole('link', { name: 'Tom Teacher' }).click();
  await panel.getByRole('button', { name: 'Deactivate', exact: true }).click();
  await panel.getByRole('button', { name: 'Deactivate this user' }).click();
  await expect(panel.getByText('Deactivated').first()).toBeVisible();

  // Their existing session stops working at the next request.
  const after = await asTeacher.get('/api/v1/learners');
  expect(after.status()).toBe(401);
  expect((await after.json()).code).toBe('ACCOUNT_DEACTIVATED');
  await asTeacher.dispose();

  // The principal cannot remove themselves.
  await page.goto('/school/users');
  await page.getByRole('link', { name: 'Mary Wanjiku' }).click();
  await expect(panel.getByRole('button', { name: 'Deactivate', exact: true })).toHaveCount(0);
});

test('an email that already signs in cannot register a second school', async ({ page }) => {
  await page.goto('/auth/register');
  await page.getByLabel('School name').fill(`Second School ${tag}`);
  await page.getByRole('radio', { name: /Private/ }).check();
  await page.getByLabel('County').selectOption('Kiambu');
  await page.getByRole('button', { name: 'Continue' }).click();
  await page.getByLabel('Your full name').fill('Mary Wanjiku');
  await page.getByLabel('Your email').fill(PRINCIPAL);
  await page.getByLabel('Choose a password').fill(PASSWORD);
  await page.getByLabel('Type the password again').fill(PASSWORD);
  await page.getByRole('button', { name: 'Continue' }).click();
  await page.getByRole('button', { name: 'Create the school' }).click();

  // The form goes back to the step the problem is on and says what it is.
  await expect(page.getByText(new RegExp(`already signs in to ${SCHOOL}`))).toBeVisible();
  await expect(page.getByLabel('Your email')).toBeVisible();
});
