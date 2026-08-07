import { defineConfig } from '@playwright/test';

const port = 8089;

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: 'list',
  use: {
    baseURL: `http://localhost:${port}`,
    // `on-first-retry` captures nothing while retries are 0, which is the state
    // this config has always been in — so a CI failure used to leave only the
    // list reporter's text behind. These assertions are geometry ("expected 8,
    // received 9.5"), which is near-undebuggable without a trace, and the suite
    // now gates publishing. Retries stay at 0: a flaky gate trains people to
    // re-run red builds, and the first real failure gets re-run with them.
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
  ],
});
