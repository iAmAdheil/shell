import { APP_BASE_URL, WEB_BASE_URL, devLoginURL } from './dev-login'

/**
 * These checks run before any test, so a stack that is not up fails once with
 * an instruction instead of once per test with a timeout.
 */
export default async function globalSetup() {
  await reachable(WEB_BASE_URL, 'the web app', 'run `make start`')
  await reachable(`${APP_BASE_URL}/api/health`, 'the server', 'run `make start`')

  // A 404 here means the dev routes were never registered.
  const res = await fetch(devLoginURL('probe'), { redirect: 'manual' })
  if (res.status === 404) {
    throw new Error(
      `The dev login is off. Set DEV_LOGIN_SECRET in .env and restart the server.\n` +
        `See .env.example. The tests use "dev" unless DEV_LOGIN_SECRET says otherwise.`,
    )
  }
}

async function reachable(url: string, what: string, fix: string) {
  try {
    await fetch(url)
  } catch {
    throw new Error(`Cannot reach ${what} at ${url}. To fix: ${fix}.`)
  }
}
