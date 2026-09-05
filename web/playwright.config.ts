import { defineConfig, devices } from '@playwright/test'

/**
 * These tests drive the real app. They do not start it, because the stack is
 * four things — Postgres, the Session image, the Go server and Vite — and
 * booting all four from a test runner turns every unrelated failure into a
 * failed test. Global setup checks the stack is up and says what to run.
 */
export default defineConfig({
  testDir: './e2e',
  globalSetup: './e2e/global-setup.ts',
  timeout: 60_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  workers: 1,
  reporter: 'list',
  use: {
    baseURL: process.env.WEB_BASE_URL ?? 'http://localhost:5173',
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
})
