import { expect, test } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

// --- Public surface + redirect guards (no API required) ---

test('landing page renders the product', async ({ page }) => {
  await page.goto('/');
  await expect(page).toHaveTitle(/Shule360/);
});

test('unauthenticated workspace address is bounced to staff sign-in (edge guard)', async ({ page }) => {
  await page.goto('/w/all/a0000000-0000-0000-0000-000000000001/communications/messages');
  await expect(page).toHaveURL(/\/auth\/login/);
  await expect(page.getByRole('button', { name: /sign in/i })).toBeVisible();
});

test('unauthenticated /dashboard is bounced to staff sign-in (edge guard)', async ({ page }) => {
  await page.goto('/dashboard');
  await expect(page).toHaveURL(/\/auth\/login/);
  await expect(page.getByRole('button', { name: /sign in/i })).toBeVisible();
});

test('unauthenticated /parent is bounced to parent sign-in (edge guard)', async ({ page }) => {
  await page.goto('/parent');
  await expect(page).toHaveURL(/\/parent\/login/);
  await expect(page.getByRole('heading', { name: 'Parent Portal', exact: true })).toBeVisible();
});

// Regression guard: the sidebar has linked /settings since Phase 1 while the
// route did not exist, and it was also missing from the edge guard list — so
// the page 404'd for signed-in staff and rendered for anonymous visitors.
test('/settings exists and is behind the staff edge guard', async ({ page }) => {
  const response = await page.goto('/settings');
  expect(response?.status(), 'the route must exist (not 404)').not.toBe(404);
  await expect(page).toHaveURL(/\/auth\/login/);
  await expect(page.getByRole('button', { name: /sign in/i })).toBeVisible();
});

test('staff sign-in form renders with labelled controls', async ({ page }) => {
  await page.goto('/auth/login');
  await expect(page.getByLabel(/email/i)).toBeVisible();
  await expect(page.getByLabel(/password/i)).toBeVisible();
  await expect(page.getByRole('button', { name: /sign in/i })).toBeVisible();
});

test('parent sign-in form renders school picker, phone and PIN', async ({ page }) => {
  await page.goto('/parent/login');
  // School list fetch fails without the API; the form must still render.
  await expect(page.getByRole('heading', { name: 'Parent Portal', exact: true })).toBeVisible();
  await expect(page.getByLabel(/phone number/i)).toBeVisible();
  await expect(page.getByLabel(/pin/i)).toBeVisible();
});

// --- WCAG 2.1 A/AA automated checks on the public surface ---

test('staff login passes axe WCAG A/AA', async ({ page }) => {
  await page.goto('/auth/login');
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
    .analyze();
  expect(results.violations).toEqual([]);
});

test('parent login passes axe WCAG A/AA', async ({ page }) => {
  await page.goto('/parent/login');
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
    .analyze();
  expect(results.violations).toEqual([]);
});

// --- Authenticated smoke (skipped without staging credentials) ---

const STAFF_EMAIL = process.env.E2E_STAFF_EMAIL;
const STAFF_PASSWORD = process.env.E2E_STAFF_PASSWORD;
const HAVE_STAFF = Boolean(STAFF_EMAIL && STAFF_PASSWORD);

async function signInAsStaff(page: import('@playwright/test').Page) {
  await page.goto('/auth/login');
  await page.getByLabel(/email/i).fill(STAFF_EMAIL!);
  await page.getByLabel(/password/i).fill(STAFF_PASSWORD!);
  await page.getByRole('button', { name: /sign in/i }).click();
  await expect(page).toHaveURL(/\/dashboard/);
}

// The authenticated tests share one sign-in per test. They run serially
// because the API's login rate limiter (5 attempts per IP per 15 minutes, by
// design) would otherwise trip when four workers sign in at once.
test.describe('authenticated staff', () => {
  test.describe.configure({ mode: 'serial' });

  test('reaches the dashboard', async ({ page }) => {
    test.skip(!HAVE_STAFF, 'Set E2E_STAFF_EMAIL/E2E_STAFF_PASSWORD against a staging API to run');
    await signInAsStaff(page);
    await expect(page.getByText(/dashboard/i).first()).toBeVisible();
  });

  test('reaches the settings screen', async ({ page }) => {
    test.skip(!HAVE_STAFF, 'Set E2E_STAFF_EMAIL/E2E_STAFF_PASSWORD against a staging API to run');
    await signInAsStaff(page);

    await page.goto('/settings');
    await expect(page.getByRole('heading', { name: 'Settings' })).toBeVisible();
    for (const tab of ['School', 'Operations', 'Integrations', 'Access']) {
      await expect(page.getByRole('tab', { name: tab })).toBeVisible();
    }

    // School tab: the principal's own school profile is loaded from the API.
    await expect(page.getByLabel('School name')).toBeVisible();

    // Integrations tab: every provider is offered, with a live test action.
    await page.getByRole('tab', { name: 'Integrations' }).click();
    await expect(page.getByRole('heading', { name: 'M-Pesa (Safaricom Daraja)' })).toBeVisible();
    await expect(page.getByRole('button', { name: /test connection/i }).first()).toBeVisible();
    // Credentials are password inputs and never pre-filled with a stored value.
    await expect(page.getByLabel('Consumer secret')).toHaveAttribute('type', 'password');
    await expect(page.getByLabel('Consumer secret')).toHaveValue('');

    // Access tab: the signed-in user's own account.
    await page.getByRole('tab', { name: 'Access' }).click();
    await expect(page.getByRole('heading', { name: 'Session & security' })).toBeVisible();
  });

  // Regression guard: since the session moved into an HttpOnly cookie, the
  // in-memory token only exists right after signing in. Guarding pages and
  // queries on that token made every admin page bounce to the sign-in form on a
  // hard reload, so this asserts the cookie session alone keeps a page working.
  test('keeps working across a hard page reload (cookie, not in-memory token)', async ({ page }) => {
    test.skip(!HAVE_STAFF, 'Set E2E_STAFF_EMAIL/E2E_STAFF_PASSWORD against a staging API to run');
    await signInAsStaff(page);

    await page.goto('/learners');
    await expect(page).toHaveURL(/\/learners/);
    await expect(page.getByRole('heading', { name: /learners/i }).first()).toBeVisible();

    await page.goto('/finance');
    await expect(page).toHaveURL(/\/finance/);
    await expect(page.getByRole('heading', { name: /finance/i }).first()).toBeVisible();
  });

  test('settings screen passes axe WCAG A/AA', async ({ page }) => {
    test.skip(!HAVE_STAFF, 'Set E2E_STAFF_EMAIL/E2E_STAFF_PASSWORD against a staging API to run');
    await signInAsStaff(page);
    await page.goto('/settings');
    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();
    expect(results.violations).toEqual([]);
  });
});
