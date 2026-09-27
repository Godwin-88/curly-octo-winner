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

  /**
   * Regression: the KICD code is offered as optional, so the form sends an
   * empty string. The column carried a UNIQUE constraint, which meant the first
   * code-less row occupied "" and every later one was rejected as a duplicate --
   * so only ever one sub-strand per strand could be added without knowing the
   * ministry codes. This adds two in a row, which is exactly what used to fail.
   */
  test('adds several sub-strands without KICD codes', async ({ page, request }) => {
    const area = `E2E codeless area ${Date.now()}`;
    const strand = `E2E codeless strand ${Date.now()}`;

    // Build a parent area + strand through the API so the form has somewhere
    // to put the sub-strands, then remove the strand (cascades) afterwards.
    const auth = { Authorization: `Bearer ${TOKEN}` };
    const mk = async (path: string, body: unknown) => {
      const res = await request.post(`/api/v1${path}`, { headers: auth, data: body });
      expect(res.ok(), `${path} -> ${res.status()} ${await res.text()}`).toBeTruthy();
      return (await res.json()) as { id: string };
    };
    const areaId = (
      await mk('/curriculum/learning-areas', {
        name: area,
        kicd_code: '',
        description: '',
        grade_level: 'Lower Primary',
      })
    ).id;
    const strandId = (
      await mk('/curriculum/strands', {
        name: strand,
        kicd_code: '',
        description: '',
        learning_area_id: areaId,
      })
    ).id;

    try {
      await page.goto('/academic/curriculum');
      await page.getByRole('tab', { name: 'sub-strands', exact: true }).click();
      await page.getByLabel('Learning area', { exact: true }).selectOption(areaId);
      await page.getByLabel('Strand', { exact: true }).selectOption(strandId);

      for (const name of ['Counting and place value', 'Addition and subtraction']) {
        // The dialog renders a button with the same accessible name, so scope
        // the opener to the page (it comes first in the DOM) and the submit to
        // the dialog. Otherwise a failure reads as a locator ambiguity instead
        // of the error the user actually sees.
        await page.getByRole('button', { name: 'Add sub-strand' }).first().click();
        const dialog = page.getByRole('dialog');
        await dialog.locator('#curriculum-name').fill(name);
        await dialog.getByRole('button', { name: 'Add sub-strand' }).click();

        // The dialog must close on success: staying open with an error is how
        // the "already exists" collision surfaced.
        await expect(dialog).toHaveCount(0, { timeout: 5000 });
        await expect(page.getByRole('listitem').filter({ hasText: name })).toBeVisible();
      }

      // Both rows present: the second create is the one that used to 409.
      const rows = await page.getByRole('listitem').filter({ hasText: /place value|subtraction/ }).count();
      expect(rows).toBe(2);
    } finally {
      await request.delete(`/api/v1/curriculum/strands/${strandId}`, { headers: auth });
      await request.delete(`/api/v1/curriculum/learning-areas/${areaId}`, { headers: auth });
    }
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
