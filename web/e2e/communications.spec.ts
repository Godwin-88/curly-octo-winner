import { expect, request, test, type Cookie, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

/**
 * Communications (SMS) and the school context, against the local stack:
 *
 *   docker compose up --build        (repo root)
 *   E2E_STACK=1 npx playwright test e2e/communications.spec.ts
 *
 * The stack's stand-in for Africa's Talking sends nothing and posts a delivery
 * report back a few seconds later, so "delivered" here is the real callback
 * path. Skipped unless E2E_STACK is set: the CI smoke suite runs without an API.
 *
 * Sign-in is limited to 5 attempts per address per 15 minutes and a run uses
 * several, so before a second run within that window:
 *   docker compose exec redis redis-cli FLUSHALL
 */
const STACK = Boolean(process.env.E2E_STACK);
const PASSWORD = 'password123';
const SCHOOL = 'Jua Kali Primary School';
const BASE_URL = `http://localhost:${process.env.E2E_PORT || 3100}`;

test.describe.configure({ mode: 'serial' });

// Sign-in is rate limited (5 attempts per address per 15 minutes), so each
// account signs in once and its session cookie is reused by every test.
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
  await page.goto('/dashboard');
}

test.beforeEach(() => {
  test.skip(!STACK, 'Set E2E_STACK=1 with the docker compose stack running');
});

test.describe('school staff', () => {
  test.beforeEach(async ({ page }) => {
    await signIn(page, 'principal@juakali.sch.ke');
  });

  test('signs in through the form', async ({ browser }) => {
    const context = await browser.newContext();
    const page = await context.newPage();
    await page.goto('/auth/login');
    await page.getByLabel('Email').fill('principal@juakali.sch.ke');
    await page.getByLabel('Password').fill(PASSWORD);
    await page.getByRole('button', { name: /sign in/i }).click();
    await expect(page).toHaveURL(/\/dashboard$/);
    await expect(page.getByRole('navigation', { name: 'Context' })).toContainText(SCHOOL);
    await context.close();
  });

  test('the old address opens the message list inside the school, with the school fixed', async ({ page }) => {
    await page.goto('/communications');
    await expect(page).toHaveURL(/\/w\/all\/[0-9a-f-]{36}\/communications\/messages$/);
    await expect(page.getByRole('heading', { name: 'Messages', level: 1 })).toBeVisible();

    const context = page.getByRole('navigation', { name: 'Context' });
    await expect(context).toContainText(SCHOOL);
    // One school: there is nothing to switch.
    await expect(context.getByRole('button')).toHaveCount(0);
  });

  test('sends an SMS: preview, confirm, then delivery reports arrive', async ({ page }) => {
    await page.goto('/communications/messages');
    await page.getByRole('link', { name: 'New SMS' }).click();
    await expect(page).toHaveURL(/do=create/);

    const text = `Dear {{parent_name}}, school closes at noon on Friday. Ref ${Date.now()}`;
    await page.getByRole('textbox', { name: 'Message', exact: true }).fill(text);
    await expect(page.getByText(/1 SMS unit before names are filled in/)).toBeVisible();

    // Nothing is sent by the first click: it shows who and how much.
    await page.getByRole('button', { name: 'Check and continue' }).click();
    await expect(page.getByText(/This goes to 20 people, 1 SMS unit each: about KES 16\.00\./)).toBeVisible();

    await page.getByRole('button', { name: 'Confirm: Send' }).click();
    await expect(page).toHaveURL(/\/communications\/messages\/[0-9a-f-]{36}$/);

    // Accepted by the provider first, delivered only once reports come back.
    const detail = page.getByRole('region', { name: 'message detail' });
    await expect(detail.getByText('Recipients', { exact: true }).first()).toBeVisible();
    await expect(detail.locator('div').filter({ hasText: /^Delivered20/ }).first()).toBeVisible({ timeout: 25_000 });
    await expect(detail.getByText('Sent', { exact: true }).first()).toBeVisible();

    // Each recipient got their own name.
    await expect(detail.getByRole('table', { name: 'Recipients of this message' })).toContainText('+254 712 345 101');
  });

  test('changing the message after the preview asks for the preview again', async ({ page }) => {
    await page.goto('/communications/messages');
    await page.getByRole('link', { name: 'New SMS' }).click();
    await page.getByRole('textbox', { name: 'Message', exact: true }).fill('First wording');
    await page.getByRole('button', { name: 'Check and continue' }).click();
    await expect(page.getByRole('button', { name: 'Confirm: Send' })).toBeVisible();

    await page.getByRole('textbox', { name: 'Message', exact: true }).fill('Second wording');
    await expect(page.getByRole('button', { name: 'Confirm: Send' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Check and continue' })).toBeVisible();
  });

  test('refuses a field that cannot be filled in, and a message that is too long', async ({ page }) => {
    await page.goto('/communications/messages');
    await page.getByRole('link', { name: 'New SMS' }).click();

    await page.getByRole('textbox', { name: 'Message', exact: true }).fill('You owe {{fee_balance}}');
    await page.getByRole('button', { name: 'Check and continue' }).click();
    await expect(page.getByRole('main').getByRole('alert')).toContainText('{{fee_balance}} is not something that can be filled in');

    await page.getByRole('textbox', { name: 'Message', exact: true }).fill('a'.repeat(460));
    await expect(page.getByText(/over the limit of 3 units/)).toBeVisible();
    await page.getByRole('button', { name: 'Check and continue' }).click();
    await expect(page.getByRole('main').getByRole('alert')).toContainText('4 SMS units long');
    await expect(page).toHaveURL(/do=create/);
  });

  test('sends to parents chosen by name', async ({ page }) => {
    await page.goto('/communications/messages');
    await page.getByRole('link', { name: 'New SMS' }).click();
    await page.getByLabel('Send to').selectOption('custom');

    // Nobody chosen yet: the form says so instead of sending to no one.
    await page.getByRole('textbox', { name: 'Message', exact: true }).fill(`A word about your child. Ref ${Date.now()}`);
    await page.getByRole('button', { name: 'Check and continue' }).click();
    await expect(page.getByText('This is needed.')).toBeVisible();

    const search = page.getByRole('searchbox', { name: 'Parents' });
    await search.fill('Catherine');
    await page.getByRole('list', { name: 'Matches for Parents' }).getByRole('button', { name: /Catherine Mwikali/ }).click();
    await search.fill('0712345119');
    await page.getByRole('list', { name: 'Matches for Parents' }).getByRole('button', { name: /Daniel Kiprono/ }).click();

    const chosen = page.getByRole('list', { name: 'Chosen: Parents' });
    await expect(chosen.getByRole('listitem')).toHaveCount(2);
    // Someone already chosen is not offered again.
    await search.fill('Catherine');
    await expect(page.getByRole('list', { name: 'Matches for Parents' })).toContainText('Nobody matches.');

    await chosen.getByRole('button', { name: 'Remove Daniel Kiprono' }).click();
    await expect(chosen.getByRole('listitem')).toHaveCount(1);
    await search.fill('Daniel');
    await page.getByRole('list', { name: 'Matches for Parents' }).getByRole('button', { name: /Daniel Kiprono/ }).click();

    await page.getByRole('button', { name: 'Check and continue' }).click();
    await expect(page.getByText(/This goes to 2 people, 1 SMS unit each/)).toBeVisible();
    await page.getByRole('button', { name: 'Confirm: Send' }).click();

    const detail = page.getByRole('region', { name: 'message detail' });
    await expect(detail.getByRole('heading', { name: 'Selected parents' })).toBeVisible();
    const recipients = detail.getByRole('table', { name: 'Recipients of this message' });
    await expect(recipients.getByRole('row')).toHaveCount(3); // header + 2
    await expect(recipients).toContainText('Catherine Mwikali');
    await expect(recipients).toContainText('Daniel Kiprono');
  });

  test('a scheduled message waits, and can be cancelled', async ({ page }) => {
    await page.goto('/communications/messages');
    await page.getByRole('link', { name: 'New SMS' }).click();
    await page.getByRole('textbox', { name: 'Message', exact: true }).fill(`Scheduled notice ${Date.now()}`);
    await page.getByRole('textbox', { name: 'Send later (optional)', exact: true }).fill('2030-01-15T08:00');
    await page.getByRole('button', { name: 'Check and continue' }).click();
    await expect(page.getByText(/It will be sent on 15 Jan 2030/)).toBeVisible();
    await page.getByRole('button', { name: 'Confirm: Send' }).click();

    const detail = page.getByRole('region', { name: 'message detail' });
    await expect(detail.getByText('Scheduled', { exact: true }).first()).toBeVisible();

    await detail.getByRole('button', { name: 'Cancel this message' }).click();
    await detail.getByRole('button', { name: 'Cancel the message' }).click();
    await expect(detail.getByText('Cancelled', { exact: true }).first()).toBeVisible();
    await expect(detail.getByRole('button', { name: 'Cancel this message' })).toHaveCount(0);
  });

  test('contacts: add, edit, archive and restore', async ({ page }) => {
    const suffix = String(Date.now()).slice(-6);
    const name = `Sponsor ${suffix}`;
    await page.goto('/communications/contacts');
    await expect(page.getByRole('heading', { name: 'Contacts', level: 1 })).toBeVisible();

    await page.getByRole('link', { name: 'Add contact' }).click();
    // A number that is not a Kenyan mobile is refused, in the API's words.
    await page.getByRole('textbox', { name: 'Full name', exact: true }).fill(name);
    await page.getByRole('textbox', { name: 'Phone', exact: true }).fill('020 123 4567');
    await page.getByRole('button', { name: 'Add contact' }).click();
    await expect(page.getByRole('main').getByRole('alert')).toContainText(/Kenyan mobile number/);

    await page.getByRole('textbox', { name: 'Phone', exact: true }).fill(`0711${suffix}`);
    await page.getByRole('textbox', { name: 'Tags', exact: true }).fill('sponsors, alumni');
    await page.getByRole('button', { name: 'Add contact' }).click();

    const detail = page.getByRole('region', { name: 'contact detail' });
    await expect(detail.getByRole('heading', { name })).toBeVisible();
    await expect(detail).toContainText(`+254 711 ${suffix.slice(0, 3)} ${suffix.slice(3)}`);
    await expect(detail).toContainText('sponsors, alumni');

    await detail.getByRole('button', { name: 'Edit' }).click();
    await page.getByRole('textbox', { name: 'Relationship', exact: true }).fill('sponsor');
    await page.getByRole('button', { name: 'Save changes' }).click();
    await expect(detail.getByText('Edit: done.')).toBeVisible();
    await expect(detail).toContainText('sponsor');

    await detail.getByRole('button', { name: 'Archive' }).click();
    await detail.getByRole('button', { name: 'Archive' }).click();
    await expect(detail.getByText('Archived', { exact: true })).toBeVisible();

    await detail.getByRole('button', { name: 'Restore' }).click();
    await expect(detail.getByText('Active', { exact: true })).toBeVisible();
  });

  test('SMS templates: create and delete', async ({ page }) => {
    const name = `Closing notice ${Date.now()}`;
    await page.goto('/communications/templates');
    await page.getByRole('link', { name: 'New template' }).click();
    await page.getByRole('textbox', { name: 'Name', exact: true }).fill(name);
    await page.getByRole('textbox', { name: 'Message', exact: true }).fill('Dear {{parent_name}}, school closes on Friday.');
    await page.getByRole('button', { name: 'New template' }).click();

    const detail = page.getByRole('region', { name: 'template detail' });
    await expect(detail.getByRole('heading', { name })).toBeVisible();
    await expect(detail).toContainText('Parent name');

    await detail.getByRole('button', { name: 'Delete' }).click();
    await detail.getByRole('button', { name: 'Delete' }).click();
    await expect(page).toHaveURL(/\/communications\/templates$/);
    await expect(page.getByRole('link', { name })).toHaveCount(0);
  });

  test('cannot reach another school by editing the address', async ({ page }) => {
    await page.goto('/w/all/a0000000-0000-0000-0000-000000000002/communications/messages');
    // The address is corrected to the school the session belongs to.
    await expect(page).toHaveURL(/\/w\/all\/a0000000-0000-0000-0000-000000000001\/communications\/messages$/);
    await expect(page.getByRole('navigation', { name: 'Context' })).toContainText(SCHOOL);
  });

  test('the message screen passes axe WCAG A/AA', async ({ page }) => {
    await page.goto('/communications/messages');
    await expect(page.getByRole('heading', { name: 'Messages', level: 1 })).toBeVisible();
    const list = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
    expect(list.violations).toEqual([]);

    await page.getByRole('link', { name: 'New SMS' }).click();
    await expect(page.getByRole('textbox', { name: 'Message', exact: true })).toBeVisible();
    const form = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
    expect(form.violations).toEqual([]);
  });
});

test.describe('context switching', () => {
  test('a platform user chooses a school, and switching never carries a record across', async ({ page }) => {
    await signIn(page, 'ops@shule360.test');

    // No school chosen: only the list of schools.
    await expect(page).toHaveURL(/\/w\/all\/all\/platform\/schools$/);
    await expect(page.getByRole('link', { name: SCHOOL })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Baraka Academy' })).toBeVisible();

    await page.getByRole('link', { name: SCHOOL }).click();
    await page.getByRole('link', { name: 'Open this school' }).click();
    await expect(page.getByRole('heading', { name: 'Messages', level: 1 })).toBeVisible();
    await expect(page.getByRole('table', { name: 'Messages' })).toBeVisible();

    // Open a message, then switch school: back to a list, of the other school.
    await page.getByRole('table', { name: 'Messages' }).getByRole('link').first().click();
    await expect(page).toHaveURL(/\/communications\/messages\/[0-9a-f-]{36}$/);

    // The group crumb narrows the school list: a school of no group is not offered.
    const context = page.getByRole('navigation', { name: 'Context' });
    await context.getByRole('button', { name: /^School/ }).click();
    await expect(page.getByRole('option', { name: /Jua Kali Primary School/ })).toBeVisible();
    await expect(page.getByRole('option', { name: /Baraka Academy/ })).toHaveCount(0);
    await page.keyboard.press('Escape');

    // Leaving the group returns to the list of schools, not to the open message.
    await context.getByRole('button', { name: /^Group/ }).click();
    await page.getByRole('option', { name: /All groups/ }).getByRole('button').click();
    await expect(page).toHaveURL(/\/w\/all\/all\/platform\/schools$/);

    await context.getByRole('button', { name: /^School/ }).click();
    await page.getByRole('option', { name: /Baraka Academy/ }).getByRole('button').click();
    await expect(page).toHaveURL(/\/w\/all\/a0000000-0000-0000-0000-000000000002\/communications\/messages$/);
    await expect(page.getByText('No messages yet')).toBeVisible();
  });

  test('a group user sees only the schools of their group', async ({ page }) => {
    await signIn(page, 'trust@juakali.test');
    await expect(page).toHaveURL(/\/w\/b0000000-0000-0000-0000-000000000001\/all\/platform\/schools$/);
    await expect(page.getByRole('link', { name: SCHOOL })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Baraka Academy' })).toHaveCount(0);

    // A school outside the group answers like one that does not exist.
    await page.goto('/w/b0000000-0000-0000-0000-000000000001/a0000000-0000-0000-0000-000000000002/communications/messages');
    await expect(page.getByRole('main').getByRole('alert')).toContainText(/School not found|does not exist/);
    await expect(page.getByRole('table', { name: 'Messages' })).toHaveCount(0);
  });
});

test.describe('platform administration', () => {
  const suffix = String(Date.now()).slice(-7);
  const groupName = `Hilltop Trust ${suffix}`;
  const schoolName = `Hilltop Primary ${suffix}`;
  const userEmail = `director-${suffix}@example.test`;
  const userName = `Hilltop Director ${suffix}`;
  let password = '';

  test('adds a group, a school in it, and a user for that group', async ({ page }) => {
    await signIn(page, 'ops@shule360.test');
    await page.goto('/w/all/all/platform/groups');

    await page.getByRole('link', { name: 'Add group' }).click();
    await page.getByRole('textbox', { name: 'Group name', exact: true }).fill(groupName);
    await page.getByRole('button', { name: 'Add group' }).click();
    await expect(page.getByRole('region', { name: 'group detail' }).getByRole('heading', { name: groupName })).toBeVisible();

    await page.getByRole('navigation', { name: 'Platform sections' }).getByRole('link', { name: 'Schools' }).click();
    await page.getByRole('link', { name: 'Add school' }).click();
    await page.getByRole('textbox', { name: 'School name', exact: true }).fill(schoolName);
    await page.getByLabel('Group').selectOption({ label: groupName });
    await page.getByRole('button', { name: 'Add school' }).click();
    await expect(page.getByRole('region', { name: 'school detail' })).toContainText(groupName);

    await page.getByRole('navigation', { name: 'Platform sections' }).getByRole('link', { name: 'Users' }).click();
    await page.getByRole('link', { name: 'Add user' }).click();

    // A school staff member's email is refused: it would sign in as that staff member.
    await page.getByRole('textbox', { name: 'Email', exact: true }).fill('principal@juakali.sch.ke');
    await page.getByRole('textbox', { name: 'Full name', exact: true }).fill(userName);
    await page.getByLabel('Group').selectOption({ label: groupName });
    await page.getByRole('button', { name: 'Add user' }).click();
    await expect(page.getByRole('main').getByRole('alert')).toContainText('already a staff member of Jua Kali Primary School');

    await page.getByRole('textbox', { name: 'Email', exact: true }).fill(userEmail);
    await page.getByRole('button', { name: 'Add user' }).click();

    // The password is shown once.
    const detail = page.getByRole('region', { name: 'user detail' });
    await expect(detail.getByText(`Password for ${userName}`)).toBeVisible();
    password = (await detail.locator('code').nth(1).innerText()).trim();
    expect(password).toMatch(/^[A-Za-z2-9]{4}(-[A-Za-z2-9]{4}){3}$/);
    await detail.getByRole('button', { name: 'I have saved it' }).click();
    await expect(detail.locator('code')).toHaveCount(0);
    await page.reload();
    await expect(page.getByRole('region', { name: 'user detail' })).toContainText(userEmail);
    await expect(page.getByRole('region', { name: 'user detail' }).locator('code')).toHaveCount(0);
  });

  test('the new user signs in and reaches only their group; deactivating stops them at once', async ({ page }) => {
    const theirs = await request.newContext({ baseURL: BASE_URL });
    const login = await theirs.post('/api/v1/login', { data: { email: userEmail, password } });
    expect(login.status()).toBe(200);
    expect((await login.json()).session.scope).toBe('group');

    const visible = await (await theirs.get('/api/v1/schools')).json();
    expect(visible.map((school: { name: string }) => school.name)).toEqual([schoolName]);
    const hilltop = visible[0].id as string;

    // Inside their school, a screen that records who acted works for them.
    const added = await theirs.post('/api/v1/contacts', {
      headers: { 'X-School-ID': hilltop },
      data: { full_name: 'Group Office Contact', phone: `0722${suffix.slice(-6)}` },
    });
    expect(added.status()).toBe(201);
    // Another group's school, and the platform area, are closed to them.
    expect((await theirs.get('/api/v1/messages', { headers: { 'X-School-ID': 'a0000000-0000-0000-0000-000000000001' } })).status()).toBe(404);
    expect((await theirs.get('/api/v1/platform/users')).status()).toBe(403);

    await signIn(page, 'ops@shule360.test');
    await page.goto('/w/all/all/platform/users');
    await page.getByRole('table', { name: 'Users' }).getByRole('link', { name: userName }).click();
    const detail = page.getByRole('region', { name: 'user detail' });
    await detail.getByRole('button', { name: 'Deactivate' }).click();
    await detail.getByRole('button', { name: 'Deactivate' }).click();
    await expect(detail.getByText('Deactivated', { exact: true })).toBeVisible();

    // Their session is still unexpired, but the next request is refused.
    expect((await theirs.get('/api/v1/contacts', { headers: { 'X-School-ID': hilltop } })).status()).toBe(401);
    await theirs.dispose();
  });

  test('an administrator cannot deactivate their own account', async ({ page }) => {
    await signIn(page, 'ops@shule360.test');
    await page.goto('/w/all/all/platform/users');
    await page.getByRole('table', { name: 'Users' }).getByRole('link', { name: 'Platform Operator' }).click();
    const detail = page.getByRole('region', { name: 'user detail' });
    await detail.getByRole('button', { name: 'Deactivate' }).click();
    await detail.getByRole('button', { name: 'Deactivate' }).click();
    await expect(detail.getByRole('alert')).toContainText('cannot deactivate your own account');
  });

  test('a platform user can use the screens that have not moved yet', async ({ page }) => {
    await signIn(page, 'ops@shule360.test');
    await page.goto('/w/b0000000-0000-0000-0000-000000000001/a0000000-0000-0000-0000-000000000001/communications/messages');
    await expect(page.getByRole('heading', { name: 'Messages', level: 1 })).toBeVisible();

    // Settings is one of the screens that requires a staff identity.
    await page.getByRole('link', { name: 'Settings' }).click();
    await expect(page).toHaveURL(/\/settings$/);
    await expect(page.getByRole('navigation', { name: 'Context' })).toContainText(SCHOOL);
    await expect(page.getByRole('main').getByRole('alert')).toHaveCount(0);
    await expect(page.getByRole('main')).not.toContainText(/Staff ID not found|Unauthorized|Something went wrong/i);
  });
});
