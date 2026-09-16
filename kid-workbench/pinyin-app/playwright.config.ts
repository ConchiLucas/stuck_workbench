import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  use: {
    ...devices['Desktop Safari'],
    viewport: { width: 1024, height: 768 },
    baseURL: 'https://pinyin.test',
    serviceWorkers: 'block',
  },
  // e2e/fixture.ts serves dist and mocks every API in Playwright itself.
  // No webServer and no connection to the everyday :19112 app or database.
})
