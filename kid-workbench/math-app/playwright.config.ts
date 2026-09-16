import { defineConfig, devices } from '@playwright/test'

// Tests serve the existing dist build through request interception only.
// Run `npm run build` before this suite; no webServer or live backend is used.
export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  use: {
    baseURL: 'https://math.test',
    serviceWorkers: 'block',
    trace: 'retain-on-failure',
  },
  projects: [
    { name: 'tablet', use: { ...devices['iPad (gen 7) landscape'], browserName: 'webkit' } },
    { name: 'mobile', use: { ...devices['iPhone 13'], browserName: 'webkit' } },
  ],
})
