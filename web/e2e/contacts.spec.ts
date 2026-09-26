import { expect, test } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

/**
 * Contact book UI checks against a locally running stack.
 *
 * These run only when E2E_STAFF_TOKEN is set: it is a real session token, so
 * the suite stays in the "authenticated" bucket like the rest of the e2e file
 * instead of pretending a login works in CI.
 */
const TOKEN = process.env.E2E_STAFF_TOKEN;
const HAVE_TOKEN = Boolean(TOKEN);

test.describe('contacts screen', () => {
  test.beforeEach(async ({ context }) => {
    test.skip(!HAVE_TOKEN, 'Set E2E_STAFF_TOKEN to run the authenticated contacts tests');
    await context.addCookies([
      {
        name: 'shule360_session',
        value: TOKEN as string,
        domain: 'localhost',
        path: '/',
        httpOnly: true,
        sameSite: 'Lax',
      },
    ]);
  });

  test('lists saved contacts with phone, tags and status', async ({ page }) => {
    await page.goto('/communications/contacts');

    await expect(page.getByRole('heading', { name: 'Contacts' })).toBeVisible();
    await expect(page.getByRole('button', { name: /Add contact/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /Import in bulk/ })).toBeVisible();

    // The table header is the contract for a contact row.
    await expect(page.getByRole('columnheader', { name: 'Name' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'Phone' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'Tags' })).toBeVisible();
  });

  test('rejects an invalid phone number in the form before saving', async ({ page }) => {
    await page.goto('/communications/contacts');
    await page.getByRole('button', { name: /Add contact/ }).first().click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    await dialog.getByLabel(/Full name/).fill('Test Person');
    await dialog.getByLabel(/^Phone/).fill('12345');
    await dialog.getByLabel(/^Phone/).blur();

    // Inline, specific feedback instead of a server round trip.
    await expect(dialog.getByText(/Enter a Kenyan mobile number/)).toBeVisible();
    await expect(dialog.getByRole('button', { name: /Save contact/ })).toBeDisabled();
  });

  test('previews a pasted list before importing it', async ({ page }) => {
    await page.goto('/communications/contacts');
    await page.getByRole('button', { name: /Import in bulk/ }).first().click();

    const dialog = page.getByRole('dialog');
    await expect(dialog.getByRole('heading', { name: /Import contacts in bulk/ })).toBeVisible();

    // A number that cannot already exist in the book: the preview reports
    // numbers it finds as duplicates, so a fixed number would make this test
    // depend on whatever the database happens to hold.
    const freshNumber = `07${Math.floor(10000000 + Math.random() * 89999999)}`;

    await dialog.getByLabel(/Rows to import/).fill(
      `Name,Phone\nE2E Probe,${freshNumber}\nE2E Broken,12345`
    );
    await dialog.getByRole('button', { name: /Check rows/ }).click();

    // The preview must separate the good row from the broken one, and the
    // import button must count only the importable rows.
    await expect(dialog.getByRole('button', { name: /Import 1 contact$/ })).toBeVisible();
    // exact: the tiles also contain the words "ready to save" and
    // "already saved", so a substring match would be ambiguous.
    await expect(dialog.getByText('Ready', { exact: true })).toBeVisible();
    await expect(dialog.getByText(/not a valid Kenyan phone number/)).toBeVisible();

    // Line numbers must point at the pasted file, not the result table.
    const badRow = dialog.getByRole('row').filter({ hasText: 'not a valid' });
    await expect(badRow.getByRole('cell').first()).toHaveText('3');
  });

  test('contacts screen passes axe WCAG A/AA', async ({ page }) => {
    await page.goto('/communications/contacts');
    await expect(page.getByRole('heading', { name: 'Contacts' })).toBeVisible();

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();
    expect(results.violations).toEqual([]);
  });
});