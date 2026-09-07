import { expect, test, type Browser, type Page } from '@playwright/test'

import { signIn } from './dev-login'

/**
 * Smoke tests for the whole point of the app: two people, one shell. The Go
 * tests already prove this at the handler level. These prove the parts Go
 * cannot see — xterm rendering, the roster, and the share link.
 */

/** share opens one Session and puts two signed-in Users inside it. */
async function share(browser: Browser): Promise<{ ada: Page; grace: Page }> {
  const ada = await signIn(browser, 'ada')
  const grace = await signIn(browser, 'grace')

  await ada.getByRole('button', { name: 'New Session' }).click()
  await ada.waitForURL(/#\/s\/.+/)

  const code = new URL(ada.url()).hash.replace('#/s/', '')
  expect(code).not.toBe('')
  await grace.goto(`/#/s/${code}`)

  await ada.getByRole('list', { name: 'Connected Users' }).waitFor()
  await grace.getByRole('list', { name: 'Connected Users' }).waitFor()
  return { ada, grace }
}

test('two Users share one Session', async ({ browser }) => {
  const { ada, grace } = await share(browser)

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

test('each User has their own colour', async ({ browser }) => {
  const { ada } = await share(browser)

  await expect(ada.locator('li.who')).toHaveCount(2)
  const colors = await ada
    .locator('li.who')
    .evaluateAll((items) =>
      items.map((item) => getComputedStyle(item).getPropertyValue('--user').trim()),
    )

  expect(colors.every((color) => color !== '')).toBe(true)
  expect(new Set(colors).size).toBe(2)
})

test('the roster marks whoever is typing', async ({ browser }) => {
  const { ada, grace } = await share(browser)

  // One shell has one cursor, and nobody owns it until somebody types.
  await expect(ada.locator('li.who.typing')).toHaveCount(0)
  await expect(grace.locator('li.who.typing')).toHaveCount(0)

  await ada.locator('.xterm-screen').click()
  await ada.keyboard.type('a')
  await expect(grace.locator('li.who.typing .who-name')).toHaveText('ada')
  await expect(ada.locator('li.who.typing .who-name')).toHaveText('you')

  // The cursor passes to whoever typed last.
  await grace.locator('.xterm-screen').click()
  await grace.keyboard.type('g')
  await expect(ada.locator('li.who.typing .who-name')).toHaveText('grace')
  await expect(ada.locator('li.who.typing')).toHaveCount(1)

  // And the one shared cursor takes that User's colour, on everyone's screen.
  const graceColor = await ada
    .locator('li.who.typing')
    .evaluate((item) => getComputedStyle(item).getPropertyValue('--user').trim())

  await expect(ada.locator('.xterm-cursor')).toHaveCSS('background-color', hexToRgb(graceColor))
})

/** hexToRgb writes a #rrggbb colour the way getComputedStyle reports it. */
function hexToRgb(hex: string): string {
  const n = Number.parseInt(hex.slice(1), 16)
  return `rgb(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255})`
}
