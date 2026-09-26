import { expect, test } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

/**
 * Academic screens: curriculum, attendance and formative assessment.
 *
 * These replace three placeholder pages whose buttons did nothing. They run only
 * when E2E_STAFF_TOKEN is set, because there is no honest way to assert on a
 * curriculum form without a signed-in staff session behind it.
 */
const TOKEN = process.env.E2E_STAFF_TOKEN;
const HAVE_TOKEN = Boolean(TOKEN);

test.describe('curriculum screen', () => {
  test.beforeEach(async ({ context }) => {
    test.skip(!HAVE_TOKEN, 'Set E2E_STAFF_TOKEN to run the authenticated academic tests');
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

  test('lists the five curriculum levels and opens the add form', async ({ page }) => {
    await page.goto('/academic/curriculum');

    await expect(page.getByRole('heading', { name: /CBC Curriculum/ })).toBeVisible();
    for (const tab of ['learning areas', 'strands', 'sub-strands', 'core competencies', 'values']) {
      await expect(page.getByRole('tab', { name: tab, exact: true })).toBeVisible();
    }

    // The button that used to do nothing.
    await page.getByRole('button', { name: /Add learning area/ }).first().click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await expect(dialog.getByLabel(/^Name/)).toBeVisible();
    await expect(dialog.getByLabel(/KICD code/)).toBeVisible();
    await expect(dialog.getByLabel(/Grade level/)).toBeVisible();
  });

  test('will not save a curriculum item without a name', async ({ page }) => {
    await page.goto('/academic/curriculum');
    await page.getByRole('button', { name: /Add learning area/ }).first().click();

    const dialog = page.getByRole('dialog');
    // The submit button is gated on the name, so the form cannot post a blank.
    await expect(dialog.getByRole('button', { name: /Add learning area/ })).toBeDisabled();

    await dialog.getByLabel(/^Name/).fill('Test Learning Area');
    await expect(dialog.getByRole('button', { name: /Add learning area/ })).toBeEnabled();
  });

  test('switching to a child level asks for its parent', async ({ page }) => {
    await page.goto('/academic/curriculum');
    await page.getByRole('tab', { name: 'strands', exact: true }).click();

    // A strand cannot exist without a learning area, so the level must offer one.
    await expect(page.getByLabel('Learning area', { exact: true })).toBeVisible();
  });

  test('curriculum page has no obvious accessibility violations', async ({ page }) => {
    await page.goto('/academic/curriculum');
    const results = await new AxeBuilder({ page }).analyze();
    expect(results.violations.filter((v) => v.impact === 'critical' || v.impact === 'serious')).toEqual(
      []
    );
  });
});

test.describe('attendance screen', () => {
  test.beforeEach(async ({ context }) => {
    test.skip(!HAVE_TOKEN, 'Set E2E_STAFF_TOKEN to run the authenticated academic tests');
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

  test('marks a register and saves it', async ({ page }) => {
    await page.goto('/academic/attendance');

    await expect(page.getByRole('heading', { name: /Attendance Management/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /Mark all present/ })).toBeVisible();

    // Marking everyone is the common case, so the bulk action must be there.
    await page.getByRole('button', { name: /Mark all present/ }).click();
    await expect(page.getByText(/All learners marked/)).toBeVisible();

    const save = page.getByRole('button', { name: /Save register/ });
    await expect(save).toBeEnabled();
    await save.click();
    await expect(page.getByText(/Register saved/)).toBeVisible();
  });

  test('shows the chronic absence view', async ({ page }) => {
    await page.goto('/academic/attendance');
    await page.getByRole('tab', { name: /Chronic absence/ }).click();
    // Either a table of learners or the honest empty state — never a blank page.
    await expect(page.getByText(/No chronic absentees|Learner/).first()).toBeVisible();
  });
});

test.describe('assessments screen', () => {
  test.beforeEach(async ({ context }) => {
    test.skip(!HAVE_TOKEN, 'Set E2E_STAFF_TOKEN to run the authenticated academic tests');
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

  test('opens the observation form with the full curriculum cascade', async ({ page }) => {
    await page.goto('/academic/assessments');
    await expect(page.getByRole('heading', { name: /Formative Assessment/ })).toBeVisible();

    await page.getByRole('button', { name: /New Observation/ }).first().click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await expect(dialog.getByLabel(/Learner/)).toBeVisible();
    await expect(dialog.getByLabel('Learning area', { exact: true })).toBeVisible();
    await expect(dialog.getByLabel('Strand', { exact: true })).toBeVisible();
    await expect(dialog.getByLabel(/^Sub-strand/)).toBeVisible();

    // The four CBC rubric levels, each a real choice.
    for (const level of ['Below Expectation', 'Approaching', 'Meeting', 'Exceeding']) {
      await expect(dialog.getByRole('button', { name: new RegExp(level) })).toBeVisible();
    }
  });

  test('cannot record an observation without a learner', async ({ page }) => {
    await page.goto('/academic/assessments');
    await page.getByRole('button', { name: /New Observation/ }).first().click();

    const dialog = page.getByRole('dialog');
    await expect(dialog.getByRole('button', { name: /Record observation/ })).toBeDisabled();
  });
});
