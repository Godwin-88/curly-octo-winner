import { expect, test } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

/**
 * Learner roster import (staging) UI checks against a locally running stack.
 *
 * These run only when E2E_STAFF_TOKEN is set: it is a real session token, so
 * the suite stays in the "authenticated" bucket like the rest of the e2e file
 * instead of pretending a login works in CI.
 */
const TOKEN = process.env.E2E_STAFF_TOKEN;
const HAVE_TOKEN = Boolean(TOKEN);

/** Numbers that cannot already exist, so the test does not depend on the DB. */
const unique = () => `E2E${Math.floor(100000 + Math.random() * 899999)}`;

test.describe('learner roster import', () => {
  test.beforeEach(async ({ context }) => {
    test.skip(!HAVE_TOKEN, 'Set E2E_STAFF_TOKEN to run the authenticated import tests');
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

  test('stages a messy roster without rejecting the incomplete rows', async ({ page }) => {
    await page.goto('/learners/import');

    await expect(page.getByRole('heading', { name: /Import learners from a spreadsheet/ })).toBeVisible();

    const number = unique();
    // One complete row, one missing a grade, one missing a number. The point of
    // staging is that all three are kept: the file is not rejected over a blank.
    await page
      .getByLabel(/paste the rows/i)
      .fill(
        [
          'Parent Name,Parent Phone,Student Name,Student Number,Grade',
          `Mary Achieng,0712345001,Brian Achieng,${number},4`,
          `Grace Wanjiru,,Faith Wanjiru,${number}B,`,
          `Peter Otieno,0712345003,Daniel Otieno,,4`,
        ].join('\n')
      );
    await page.getByRole('button', { name: /Stage these rows/ }).click();

    // All three rows are on screen, and the table names the file's own columns.
    const table = page.getByRole('table');
    await expect(table).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'Student number' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'Grade' })).toBeVisible();
    await expect(table.locator('input[value="Brian Achieng"]')).toBeVisible();
    await expect(table.locator('input[value="Faith Wanjiru"]')).toBeVisible();

    // The two incomplete rows say what is missing, in the user's words.
    await expect(page.getByText('Needs Grade')).toBeVisible();
    await expect(page.getByText('Needs Student number')).toBeVisible();
  });

  test('editing a row later makes it importable', async ({ page }) => {
    await page.goto('/learners/import');

    const number = unique();
    await page
      .getByLabel(/paste the rows/i)
      .fill(
        [
          'Student Name,Student Number,Grade',
          `Late Learner,${number},`,
        ].join('\n')
      );
    await page.getByRole('button', { name: /Stage these rows/ }).click();

    // The row is staged but not importable yet.
    await expect(page.getByText('Needs Grade')).toBeVisible();

    // Fill the gap. The cell saves on blur, and the server — not this page —
    // decides the row is now sufficient.
    const gradeCell = page.getByLabel(/Grade for line 2/);
    await gradeCell.fill('5');
    await gradeCell.blur();

    // The row flips to ready and offers the import action.
    await expect(page.getByRole('button', { name: /^Import$/ }).first()).toBeVisible();
    await expect(page.getByText('Needs Grade')).toBeHidden();
  });

  test('has no obvious accessibility violations', async ({ page }) => {
    await page.goto('/learners/import');
    // Every editable cell carries a label naming its row, so a screen reader
    // user knows which line they are editing.
    const results = await new AxeBuilder({ page }).analyze();
    expect(results.violations).toEqual([]);
  });
});
