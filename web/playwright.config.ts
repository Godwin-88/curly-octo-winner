import { defineConfig, devices } from '@playwright/test';

// Smoke-test suite. These tests deliberately run WITHOUT the Go API: the
// same-origin rewrite proxy returns 5xx for /api/v1/* and the app degrades
// gracefully (session hydration fails -> guards redirect to sign-in). That
// keeps the suite fast and DB-free while still covering routing, rendering,
// redirect guards, and accessibility on the public surface. Authenticated
// flows are gated behind env vars for when a staging API is available.
//
// IMPORTANT: this suite serves a PRODUCTION build from web/.next. `next dev`
// writes to that same directory, so do NOT keep a dev server running while
// executing these tests: the dev server overwrites the production chunks and
// the server then 500s on them ("This page hit a problem" / ChunkLoadError).
// Stop `next dev` (or use a separate checkout) before running Playwright.

const PORT = Number(process.env.E2E_PORT || 3100);

export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  expect: { timeout: 10_000 },
  fullyParallel: true,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: `http://localhost:${PORT}`,
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        ...(process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH
          ? { launchOptions: { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH } }
          : {}),
      },
    },
  ],
  webServer: {
    command: 'npm run start -- --port ' + PORT,
    url: `http://localhost:${PORT}/auth/login`,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
});
