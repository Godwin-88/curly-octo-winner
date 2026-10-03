import { expect, request, test, type Cookie, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

/**
 * Finance, against the local stack:
 *
 *   docker compose up --build        (repo root)
 *   E2E_STACK=1 npx playwright test e2e/finance.spec.ts
 *
 * No money moves: the stack's stand-in for Safaricom "pays" an M-Pesa request
 * three seconds after it is sent and posts the result back, so "received"
 * here is the real callback path. Skipped unless E2E_STACK is set.
 *
 * Sign-in is limited to 5 attempts per address per 15 minutes; before a second
 * run within that window:  docker compose exec redis redis-cli FLUSHALL
 */
const STACK = Boolean(process.env.E2E_STACK);
const PASSWORD = 'password123';
const BASE_URL = `http://localhost:${process.env.E2E_PORT || 3100}`;
// The stand-in Safaricom, published by docker-compose for exactly this.
const SAFARICOM = process.env.E2E_DEVSTUB || 'http://localhost:9191';
const PAYBILL = '174379';

test.describe.configure({ mode: 'serial' });

const sessions = new Map<string, Cookie[]>();

async function signIn(page: Page, email: string) {
  let cookies = sessions.get(email);
  if (!cookies) {
    const api = await request.newContext({ baseURL: BASE_URL });
    const response = await api.post('/api/v1/login', { data: { email, password: PASSWORD } });
    expect(response.status(), `sign-in as ${email}`).toBe(200);
    cookies = (await api.storageState()).cookies;
    await api.dispose();
    sessions.set(email, cookies);
  }
  await page.context().addCookies(cookies);
}

/** A parent pays the paybill from their own phone. */
async function payPaybill(amount: string, account: string): Promise<string> {
  const api = await request.newContext();
  const response = await api.post(`${SAFARICOM}/dev/c2b`, { form: { amount, account, shortcode: PAYBILL, name: 'Test Parent' } });
  expect(response.status()).toBe(200);
  const body = await response.json();
  await api.dispose();
  expect(body.answer, 'the API accepted the paybill confirmation').toBe(200);
  return body.trans_id as string;
}

test.beforeEach(async ({ page }) => {
  test.skip(!STACK, 'Set E2E_STACK=1 with the docker compose stack running');
  await signIn(page, 'bursar@juakali.sch.ke');
});

// A year nobody else has used, so the run can be repeated on the same database.
const YEAR = String(2030 + Math.floor(Math.random() * 60));
const detail = (page: Page, noun: string) => page.getByRole('region', { name: `${noun} detail` });

/** Narrows a list with its filter form: label → option value. */
async function narrow(page: Page, list: string, choices: Record<string, string>) {
  const region = page.getByRole('region', { name: `${list} list` });
  await region.getByText(/^Filters/).click();
  for (const [label, value] of Object.entries(choices)) await region.getByRole('combobox', { name: label, exact: true }).selectOption(value);
  await region.getByRole('button', { name: 'Apply filters' }).click();
  return region;
}

test('the old Finance address opens the overview in the workspace', async ({ page }) => {
  await page.goto('/finance');
  await expect(page).toHaveURL(/\/w\/all\/[0-9a-f-]{36}\/finance\/overview$/);
  await expect(page.getByText('Billed', { exact: true })).toBeVisible();
  await expect(page.getByText('Still owed', { exact: true })).toBeVisible();
});

test('a fee structure is created, given an item, and bills a grade once', async ({ page }) => {
  await page.goto('/finance/fees');
  await page.getByRole('link', { name: 'New fee structure' }).click();
  const panel = detail(page, 'fee structure');
  await panel.getByLabel('Grade').selectOption('Grade 6');
  await panel.getByLabel('Term').selectOption('2');
  await panel.getByLabel('Year').fill(YEAR);
  await panel.getByLabel('Tuition (KES)').fill('12,000');
  await panel.getByRole('button', { name: 'New fee structure' }).click();
  await expect(page).toHaveURL(/\/finance\/fees\/[0-9a-f-]{36}$/);

  await expect(panel.getByText('KES 12,000.00').first()).toBeVisible();

  await panel.getByRole('button', { name: 'Add an item' }).click();
  await panel.getByRole('textbox', { name: 'Item', exact: true }).fill('Activity');
  await panel.getByLabel('Amount (KES)').fill('2500');
  await panel.getByRole('button', { name: 'Add an item' }).click();
  await expect(panel.getByText('KES 14,500.00').first()).toBeVisible();

  // The first click writes nothing: it says what would be created.
  await panel.getByRole('button', { name: 'Bill this grade' }).click();
  await panel.getByRole('button', { name: 'Check and continue' }).click();
  await expect(panel.getByText(/This creates 18 invoices of KES 14,500\.00 each: KES 261,000\.00 in all\./)).toBeVisible();
  await panel.getByRole('button', { name: 'Confirm: Create the invoices' }).click();
  await expect(panel.getByRole('button', { name: 'Bill this grade' })).toBeVisible();

  // A second run has nobody left to bill, and says so instead of billing twice.
  await panel.getByRole('button', { name: 'Bill this grade' }).click();
  await panel.getByRole('button', { name: 'Check and continue' }).click();
  await expect(panel.getByText(/All 18 learners already have an invoice/)).toBeVisible();
});

test('a cash payment gets a receipt, and more than is owed is refused', async ({ page }) => {
  await page.goto('/finance/invoices');
  const list = await narrow(page, 'Invoices', { Show: 'open', Grade: 'Grade 6' });
  await list.getByRole('link', { name: /Learner/ }).first().click();
  const panel = detail(page, 'invoice');
  await expect(panel.getByText('Still owed', { exact: true })).toBeVisible();

  await panel.getByRole('button', { name: 'Record a payment' }).click();
  await panel.getByLabel('Amount received (KES)').fill('999999');
  await panel.getByRole('button', { name: 'Check and continue' }).click();
  await panel.getByRole('button', { name: 'Confirm: Record the payment' }).click();
  await expect(panel.getByRole('alert')).toContainText(/Only KES [\d,]+\.00 is still owed on this invoice/);

  await panel.getByLabel('Amount received (KES)').fill('4500');
  await panel.getByRole('button', { name: 'Check and continue' }).click();
  await expect(panel.getByText(/KES 4,500\.00 is recorded against .* KES 10,000\.00 will still be owed\./)).toBeVisible();
  await panel.getByRole('button', { name: 'Confirm: Record the payment' }).click();

  await expect(panel.getByRole('link', { name: /^RCT-\d{4}-\d{5}$/ })).toBeVisible();
  await expect(panel.locator('div').filter({ hasText: /^Still owedKES 10,000\.00$/ }).first()).toBeVisible();
});

test('an M-Pesa request counts only once Safaricom confirms it', async ({ page }) => {
  await page.goto('/finance/invoices');
  const list = await narrow(page, 'Invoices', { Show: 'open', Grade: 'Grade 6' });
  await list.getByRole('link', { name: /Learner/ }).nth(1).click();
  const panel = detail(page, 'invoice');

  await panel.getByRole('button', { name: 'Request M-Pesa payment' }).click();
  await panel.getByLabel('Parent’s Safaricom number').fill('0712 345 678');
  await panel.getByLabel('Amount (KES)').fill('2000');
  await panel.getByRole('button', { name: 'Send the request' }).click();

  // Paid shows nothing until the confirmation arrives, then the amount.
  await expect(panel.locator('div').filter({ hasText: /^PaidKES 2,000\.00$/ }).first()).toBeVisible({ timeout: 20_000 });
  await expect(panel.getByRole('link', { name: /^RCT-\d{4}-\d{5}$/ })).toBeVisible();
});

test('a cancelled M-Pesa request pays nothing and says why', async ({ page }) => {
  await page.goto('/finance/invoices');
  const list = await narrow(page, 'Invoices', { Show: 'open', Grade: 'Grade 6' });
  await list.getByRole('link', { name: /Learner/ }).nth(2).click();
  const panel = detail(page, 'invoice');

  await panel.getByRole('button', { name: 'Request M-Pesa payment' }).click();
  await panel.getByLabel('Parent’s Safaricom number').fill('0712345699');
  await panel.getByLabel('Amount (KES)').fill('1000');
  await panel.getByRole('button', { name: 'Send the request' }).click();

  await expect(panel.getByText('Failed').first()).toBeVisible({ timeout: 20_000 });
  await expect(panel.locator('div').filter({ hasText: /^PaidKES 0\.00$/ }).first()).toBeVisible();

  await panel.getByRole('link', { name: 'Open' }).first().click();
  await expect(detail(page, 'payment').getByRole('alert')).toContainText('The parent cancelled the request on their phone.');
});

test('money paid to the paybill with an unknown account waits to be allocated', async ({ page }) => {
  const transId = await payPaybill('1500', 'school fees');

  await page.goto('/finance/paybill');
  await page.getByRole('link', { name: /school fees/ }).first().click();
  const panel = detail(page, 'paybill payment');
  await expect(panel.getByText(transId).first()).toBeVisible();
  await expect(panel.getByText(/KES 1,500\.00 of this payment is not against any invoice yet/)).toBeVisible();

  await panel.getByRole('button', { name: 'Put against an invoice' }).click();
  await panel.getByPlaceholder('Search by learner or invoice number').fill(`INV-${YEAR}`);
  await panel.getByRole('list', { name: 'Matches for Invoice' }).getByRole('button').first().click();
  await panel.getByRole('button', { name: 'Allocate' }).click();

  await expect(panel.getByRole('link', { name: /^RCT-\d{4}-\d{5}$/ })).toBeVisible();
  await expect(panel.getByText('Allocated', { exact: true }).first()).toBeVisible();
});

test('a payment is reversed only with a reason, and stays on record', async ({ page }) => {
  await page.goto('/finance/payments');
  const list = await narrow(page, 'Payments', { Status: 'completed', 'Paid by': 'cash' });
  await list.getByRole('link', { name: /Learner/ }).first().click();
  const panel = detail(page, 'payment');

  await panel.getByRole('button', { name: 'Reverse this payment' }).click();
  await panel.getByRole('button', { name: 'Reverse the payment' }).click();
  await expect(panel.getByText('This is needed.')).toBeVisible();

  await panel.getByLabel('Why is it being reversed?').fill('Entered against the wrong learner');
  await panel.getByRole('button', { name: 'Reverse the payment' }).click();
  await expect(panel.getByText(/Reversed .*: Entered against the wrong learner/)).toBeVisible();
  await expect(panel.getByRole('button', { name: 'Reverse this payment' })).toHaveCount(0);
});

test('balances list who owes and open a full statement', async ({ page }) => {
  await page.goto('/finance/balances');
  await page.getByRole('link', { name: /Learner/ }).first().click();
  const panel = detail(page, 'learner');
  await expect(panel.getByText('Balance', { exact: true })).toBeVisible();
  await expect(panel.getByRole('heading', { name: 'Invoices' })).toBeVisible();
  await expect(panel.getByRole('button', { name: 'Print this statement' })).toBeVisible();
});

test('a teacher is not offered Finance and the API refuses them', async ({ page, context }) => {
  await context.clearCookies();
  await signIn(page, 'teacher1@juakali.sch.ke');
  await page.goto('/communications/messages');
  await expect(page.getByRole('navigation', { name: 'Context' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Finance', exact: true })).toHaveCount(0);

  const response = await page.request.get('/api/v1/invoices');
  expect(response.status()).toBe(403);
});

test('the invoice screen has no accessibility violations', async ({ page }) => {
  await page.goto('/finance/invoices');
  await page.getByRole('link', { name: /Learner/ }).first().click();
  await expect(detail(page, 'invoice').getByText('Charged')).toBeVisible();
  const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa']).analyze();
  expect(results.violations.map((violation) => `${violation.id}: ${violation.help}`)).toEqual([]);
});

test('a platform administrator switches Finance off for a school, and back on', async ({ page, context }) => {
  await context.clearCookies();
  await signIn(page, 'ops@shule360.test');
  await page.goto('/w/all/all/platform/schools');
  await page.getByRole('link', { name: 'Baraka Academy' }).click();
  await expect(page).toHaveURL(/\/platform\/schools\/[0-9a-f-]{36}$/);
  const schoolId = page.url().split('/').pop()!;
  const panel = detail(page, 'school');
  const invoices = () => page.request.get('/api/v1/invoices', { headers: { 'X-School-ID': schoolId } });

  await panel.getByRole('button', { name: 'Choose modules' }).click();
  await panel.getByLabel('Finance').uncheck();
  await panel.getByRole('button', { name: 'Save modules' }).click();
  await expect(panel.getByRole('button', { name: 'Choose modules' })).toBeVisible();

  // The API refuses at once, whatever the menus show.
  const refused = await invoices();
  expect(refused.status()).toBe(403);
  expect((await refused.json()).code).toBe('MODULE_NOT_ENABLED');
  // Communications was kept.
  expect((await page.request.get('/api/v1/messages', { headers: { 'X-School-ID': schoolId } })).status()).toBe(200);

  // Inside that school, Finance is no longer offered.
  await page.goto(`/w/all/${schoolId}/communications/messages`);
  await expect(page.getByRole('heading', { name: 'Messages', level: 1 })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Finance', exact: true })).toHaveCount(0);

  await page.goto(`/w/all/all/platform/schools/${schoolId}`);
  await panel.getByRole('button', { name: 'Choose modules' }).click();
  await panel.getByLabel('Finance').check();
  await panel.getByRole('button', { name: 'Save modules' }).click();
  await expect(panel.getByRole('button', { name: 'Choose modules' })).toBeVisible();
  expect((await invoices()).status()).toBe(200);
});
