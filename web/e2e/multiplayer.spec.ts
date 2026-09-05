import { expect, test } from '@playwright/test'

import { signIn } from './dev-login'

/**
 * The smoke test for the whole point of the app: two people, one shell. The
 * Go tests already prove this at the handler level. This one proves the parts
 * Go cannot see — xterm rendering, the roster, and the share link.
 */
test('two Users share one Session', async ({ browser }) => {
  const ada = await signIn(browser, 'ada')
  const grace = await signIn(browser, 'grace')

  await ada.getByRole('button', { name: 'New Session' }).click()

  const code = await sessionCode(ada)
  await grace.goto(`/#/s/${code}`)

  const adaRoster = ada.getByRole('list', { name: 'Connected Users' })
  const graceRoster = grace.getByRole('list', { name: 'Connected Users' })

  // Each User reads as "you", so Ada sees herself plus Grace, and Grace the
  // other way round.
  await expect(adaRoster).toContainText('you')
  await expect(adaRoster).toContainText('grace')
  await expect(graceRoster).toContainText('you')
  await expect(graceRoster).toContainText('ada')

  // One input stream: what Ada types reaches Grace's screen.
  await ada.locator('.xterm-screen').click()
  await ada.keyboard.type('hello-from-ada')

  await expect(grace.locator('.xterm-screen')).toContainText('hello-from-ada')
})

/** sessionCode reads the Code the app puts in the URL hash. */
async function sessionCode(page: import('@playwright/test').Page): Promise<string> {
  await page.waitForURL(/#\/s\/.+/)
  const code = new URL(page.url()).hash.replace('#/s/', '')
  expect(code).not.toBe('')
  return code
}
