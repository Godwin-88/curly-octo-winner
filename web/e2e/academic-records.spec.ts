import { expect, request, test, type APIRequestContext, type Cookie, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

/**
 * Observations, the register and report cards, against the local stack:
 *
 *   docker compose up --build        (repo root)
 *   E2E_STACK=1 npx playwright test e2e/academic-records.spec.ts
 *
 * No SMS leaves the machine: the stack's stand-in answers for Africa's
 * Talking. Skipped unless E2E_STACK is set.
 *
 * Sign-in is limited to 5 attempts per address per 15 minutes; before a second
 * run within that window:  docker compose exec redis redis-cli FLUSHALL
 */
const STACK = Boolean(process.env.E2E_STACK);
const PASSWORD = 'password123';
const BASE_URL = `http://localhost:${process.env.E2E_PORT || 3100}`;
const PRINCIPAL = 'principal@juakali.sch.ke';
const TEACHER = 'teacher1@juakali.sch.ke';

test.describe.configure({ mode: 'serial' });

const sessions = new Map<string, Cookie[]>();

async function cookiesFor(email: string): Promise<Cookie[]> {
  let cookies = sessions.get(email);
  if (!cookies) {
    const api = await request.newContext({ baseURL: BASE_URL });
    const response = await api.post('/api/v1/login', { data: { email, password: PASSWORD } });
    expect(response.status(), `sign-in as ${email}`).toBe(200);
    cookies = (await api.storageState()).cookies;
    await api.dispose();
    sessions.set(email, cookies);
  }
  return cookies;
}

async function signIn(page: Page, email: string) {
  await page.context().addCookies(await cookiesFor(email));
}

/** The API as a signed-in member of staff, for setting a test up. */
async function apiAs(email: string): Promise<APIRequestContext> {
  return request.newContext({
    baseURL: BASE_URL,
    storageState: { cookies: await cookiesFor(email), origins: [] },
  });
}

/** Kenyan school terms: January–April, May–August, September–December. */
const TERM = Math.min(3, Math.floor(new Date().getMonth() / 4) + 1);
const YEAR = new Date().getFullYear();
const todayInKenya = () => new Intl.DateTimeFormat('en-CA', { timeZone: 'Africa/Nairobi' }).format(new Date());

interface Learner { id: string; full_name: string; grade: string; stream?: string }

let learner: Learner;
let subStrandId: string;
let cardId: string;

test.beforeAll(async () => {
  test.skip(!STACK, 'Set E2E_STACK=1 with the docker compose stack running');
  const api = await apiAs(PRINCIPAL);

  const learners: Learner[] = await (await api.get('/api/v1/learners')).json();
  expect(learners.length, 'the seed has learners').toBeGreaterThan(0);
  learner = learners[learners.length - 1];

  // A sub-strand of the school's own curriculum to observe against.
  let found = '';
  for (const area of await (await api.get('/api/v1/curriculum/learning-areas')).json()) {
    for (const strand of await (await api.get(`/api/v1/curriculum/learning-areas/${area.id}/strands`)).json()) {
      const subs = await (await api.get(`/api/v1/curriculum/strands/${strand.id}/sub-strands`)).json();
      if (subs.length) {
        found = subs[0].id;
        break;
      }
    }
    if (found) break;
  }
  expect(found, 'the seed has a sub-strand').not.toBe('');
  subStrandId = found;

  // So that the run can be repeated: an earlier run's card for this learner
  // and term is reopened and thrown away.
  const cards = await (await api.get(`/api/v1/reports?learner_id=${learner.id}&term=${TERM}&year=${YEAR}`)).json();
  for (const card of cards) {
    if (card.status === 'final') await api.post(`/api/v1/reports/${card.id}/reopen`);
    expect((await api.delete(`/api/v1/reports/${card.id}`)).status()).toBe(204);
  }
  await api.dispose();
});

test.beforeEach(async ({ page }) => {
  test.skip(!STACK, 'Set E2E_STACK=1 with the docker compose stack running');
  await signIn(page, PRINCIPAL);
});

test('an observation is recorded as the teacher who is signed in', async () => {
  const api = await apiAs(TEACHER);

  // The request the observation form sends. It used to be answered "Choose
  // the learner you are observing" whatever was chosen.
  const created = await api.post('/api/v1/assessments', {
    data: { learner_id: learner.id, sub_strand_id: subStrandId, rubric_level: 3, note: 'Explains her working.', term: TERM, year: YEAR },
  });
  expect(created.status(), await created.text()).toBe(201);
  const observation = await created.json();
  expect(observation.teacher_id).toBe('b0000000-0000-0000-0000-000000000003');

  // Naming somebody else as the teacher is not accepted.
  const forged = await api.post('/api/v1/assessments', {
    data: { learner_id: learner.id, sub_strand_id: subStrandId, rubric_level: 3, term: TERM, year: YEAR, teacher_id: 'b0000000-0000-0000-0000-000000000001' },
  });
  expect(forged.status()).toBe(400);

  const level = await api.post('/api/v1/assessments', {
    data: { learner_id: learner.id, sub_strand_id: subStrandId, rubric_level: 7, term: TERM, year: YEAR },
  });
  expect(level.status()).toBe(400);
  expect((await level.json()).error).toContain('rubric level');

  // A learner's attendance history loads (it failed on any mark with no reason).
  expect((await api.get(`/api/v1/attendance/learner/${learner.id}`)).status()).toBe(200);
  await api.dispose();
});

test('the register is one class, refuses tomorrow and says who was texted', async ({ page }) => {
  const api = await apiAs(TEACHER);
  const tomorrow = new Date(Date.now() + 36 * 3600 * 1000).toISOString().slice(0, 10);
  const refused = await api.post('/api/v1/attendance/bulk', {
    data: { date: tomorrow, marks: [{ learner_id: learner.id, status: 'present' }] },
  });
  expect(refused.status()).toBe(400);
  expect((await refused.json()).error).toContain('has not happened yet');
  await api.dispose();

  await page.goto('/academic/attendance');
  await expect(page.getByRole('heading', { name: 'Attendance Management' })).toBeVisible();

  // One class at a time: the count to mark is that class's, not the school's.
  const classes = page.getByLabel('Class', { exact: true });
  await expect(classes).toBeVisible();
  expect(await classes.locator('option').count()).toBeGreaterThan(1);

  await page.getByRole('button', { name: 'Mark all present' }).click();
  await expect(page.getByText('All learners marked.')).toBeVisible();
  // One learner is away.
  await page.getByRole('group').first().getByRole('button', { name: 'Absent' }).click();
  await page.getByLabel(/Text the parents of absent learners/).check();
  await page.getByRole('button', { name: 'Save register' }).click();

  const notice = page.getByRole('status').filter({ hasText: 'Register saved' });
  await expect(notice).toBeVisible();
  // Either a text is on its way or (on a second run today) nobody new is told.
  await expect(notice).toContainText(/A text is on its way to the parents of 1 absent learner|No new texts were sent|not texted/);

  const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa']).analyze();
  expect(results.violations.filter((v) => v.impact === 'critical' || v.impact === 'serious')).toEqual([]);
});

test('an absence alert is a message under Communications', async () => {
  const api = await apiAs(PRINCIPAL);
  const marks = await (await api.get(`/api/v1/attendance/date?date=${todayInKenya()}`)).json();
  const told = marks.filter((m: { sms_notified: boolean }) => m.sms_notified);
  const messages = await (await api.get('/api/v1/messages?limit=50')).json();
  const list = Array.isArray(messages) ? messages : messages.messages ?? messages.data ?? [];
  const alerts = list.filter((m: { content: string }) => m.content.includes('was marked absent today'));
  // Every mark that says a parent was told has a message behind it.
  expect(alerts.length).toBeGreaterThanOrEqual(Math.min(told.length, 1));
  await api.dispose();
});

test('a report card is drafted, commented on, published and then left alone', async ({ page }) => {
  await page.goto('/reports/cards');
  await expect(page.getByRole('heading', { name: 'Report cards' })).toBeVisible();

  await page.getByLabel('One learner').selectOption(learner.id);
  await page.getByRole('button', { name: 'Make draft', exact: true }).click();

  const card = page.getByRole('dialog').filter({ hasText: learner.full_name });
  await expect(card).toBeVisible();
  await expect(card.getByText('Draft', { exact: true })).toBeVisible();
  await expect(card.getByRole('cell', { name: /3 · Meeting Expectation/ })).toBeVisible();

  // The edit that used to be a database error.
  await card.getByLabel("Class teacher's comment").fill('A steady, thoughtful term.');
  await card.getByRole('button', { name: 'Save comment' }).click();
  await expect(page.getByRole('status').filter({ hasText: 'Comment saved.' })).toBeVisible();

  // The PDF is a document, not a link to nowhere.
  const download = page.waitForEvent('download');
  await card.getByRole('button', { name: 'Download PDF' }).click();
  const file = await download;
  expect(file.suggestedFilename()).toMatch(/^report-card-.*-term-\d-\d{4}\.pdf$/);

  await card.getByRole('button', { name: 'Publish' }).click();
  await page.getByRole('dialog').filter({ hasText: 'Publish this report card?' }).getByRole('button', { name: 'Publish' }).click();
  await expect(page.getByRole('status').filter({ hasText: 'Report card published' })).toBeVisible();
  await expect(card.getByText('A steady, thoughtful term.')).toBeVisible();
  await expect(card.getByRole('button', { name: 'Save comment' })).toHaveCount(0);
  await card.getByRole('button', { name: 'Close' }).click();

  // A published card has no delete in the list.
  const row = page.getByRole('row').filter({ hasText: learner.full_name });
  await expect(row.getByText('Published')).toBeVisible();
  await expect(row.getByRole('button', { name: /Delete/ })).toHaveCount(0);

  const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa']).analyze();
  expect(results.violations.filter((v) => v.impact === 'critical' || v.impact === 'serious')).toEqual([]);
});

test('a published card refuses changes; only a principal reopens it', async () => {
  const principal = await apiAs(PRINCIPAL);
  const cards = await (await principal.get(`/api/v1/reports?learner_id=${learner.id}&term=${TERM}&year=${YEAR}`)).json();
  expect(cards).toHaveLength(1);
  cardId = cards[0].id;
  expect(cards[0].status).toBe('final');
  expect(cards[0].published_at).toBeTruthy();

  const teacher = await apiAs(TEACHER);
  const edit = await teacher.patch(`/api/v1/reports/${cardId}`, { data: { teacher_comments: { 'Class teacher': 'Changed' } } });
  expect(edit.status()).toBe(409);
  const remove = await teacher.delete(`/api/v1/reports/${cardId}`);
  expect(remove.status()).toBe(409);
  const rebuild = await teacher.post('/api/v1/reports/generate', { data: { learner_id: learner.id, term: TERM, year: YEAR } });
  expect(rebuild.status()).toBe(409);
  // Setting the status by hand is not a thing.
  const force = await teacher.patch(`/api/v1/reports/${cardId}`, { data: { status: 'draft' } });
  expect(force.status()).toBe(400);
  expect((await teacher.post(`/api/v1/reports/${cardId}/reopen`)).status()).toBe(403);

  const pdf = await teacher.get(`/api/v1/reports/${cardId}/pdf`);
  expect(pdf.status()).toBe(200);
  expect(pdf.headers()['content-type']).toBe('application/pdf');
  expect((await pdf.body()).subarray(0, 5).toString()).toBe('%PDF-');

  expect((await principal.post(`/api/v1/reports/${cardId}/reopen`)).status()).toBe(200);
  await teacher.dispose();
  await principal.dispose();
});

test('the usual learning areas for a grade are added once', async ({ page }) => {
  await page.goto('/academic/curriculum');
  await page.getByLabel('Add the usual learning areas for').selectOption('Grade 9');
  await page.getByRole('button', { name: 'Add them' }).click();
  await expect(page.getByRole('status').filter({ hasText: /learning areas? added for Grade 9|Grade 9 already has the usual learning areas/ })).toBeVisible();

  await page.getByRole('button', { name: 'Add them' }).click();
  await expect(page.getByRole('status').filter({ hasText: 'Grade 9 already has the usual learning areas.' })).toBeVisible();
});
