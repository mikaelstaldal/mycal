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
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
  ],
  webServer: {
    command: `go build -o /tmp/mycal-e2e-server . && /tmp/mycal-e2e-server -addr :${port} -db /tmp/mycal-e2e-$$.db`,
    cwd: '..',
    port,
    reuseExistingServer: !process.env.CI,
    timeout: 30000,
  },
});
