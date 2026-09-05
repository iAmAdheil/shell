import type { Browser, Page } from '@playwright/test'

export const APP_BASE_URL = process.env.APP_BASE_URL ?? 'http://localhost:8081'
export const WEB_BASE_URL = process.env.WEB_BASE_URL ?? 'http://localhost:5173'
export const DEV_LOGIN_SECRET = process.env.DEV_LOGIN_SECRET ?? 'dev'

/** devLoginURL is the link that signs a browser in as one named User. */
export function devLoginURL(who: string): string {
  const q = new URLSearchParams({ u: who, k: DEV_LOGIN_SECRET })
  return `${APP_BASE_URL}/api/auth/dev/start?${q}`
}

/**
 * signIn opens a fresh browser context and logs it in as one named User. Each
 * User needs its own context, because being two Users at once means holding
 * two separate auth cookies.
 */
export async function signIn(browser: Browser, who: string): Promise<Page> {
  const context = await browser.newContext()
  const page = await context.newPage()
  await page.goto(devLoginURL(who))
  await page.waitForURL(`${WEB_BASE_URL}/**`)
  return page
}
