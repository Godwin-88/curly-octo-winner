import { expect, test } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

// --- Public surface + redirect guards (no API required) ---

test('landing page renders the product', async ({ page }) => {
  await page.goto('/');
  await expect(page).toHaveTitle(/Shule360/);
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

test('authenticated staff reaches the dashboard', async ({ page }) => {
  test.skip(!STAFF_EMAIL || !STAFF_PASSWORD, 'Set E2E_STAFF_EMAIL/E2E_STAFF_PASSWORD against a staging API to run');
  await page.goto('/auth/login');
  await page.getByLabel(/email/i).fill(STAFF_EMAIL!);
  await page.getByLabel(/password/i).fill(STAFF_PASSWORD!);
  await page.getByRole('button', { name: /sign in/i }).click();
  await expect(page).toHaveURL(/\/dashboard/);
  await expect(page.getByText(/dashboard/i).first()).toBeVisible();
});
