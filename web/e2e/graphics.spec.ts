import { expect, test, type Browser, type Page } from '@playwright/test'

import { signIn } from './dev-login'

/**
 * The terminal shows images. A tool such as terminal-browser first asks the
 * terminal whether it speaks the Kitty graphics protocol, and refuses to
 * start without an OK. These tests cover both halves: the answer to that
 * question, and an image reaching the screen.
 */

/** open signs one User in and puts them in a new Session, ready to type. */
async function open(browser: Browser): Promise<Page> {
  const page = await signIn(browser, 'ada')
  await page.getByRole('button', { name: 'New Session' }).click()
  await page.waitForURL(/#\/s\/.+/)
  await page.getByRole('list', { name: 'Connected Users' }).waitFor()
  await page.locator('.xterm-screen').click()
  return page
}

test('the terminal answers a Kitty graphics query', async ({ browser }) => {
  const page = await open(browser)

  // The shell runs printf, the terminal answers, and the answer goes back
  // into the shell as input, where the shell echoes it.
  await page.keyboard.type(String.raw`printf '\033_Gi=31,a=q,t=d,f=24,s=1,v=1;AAAA\033\\'`)
  await page.keyboard.press('Enter')

  await expect(page.locator('.xterm-screen')).toContainText('_Gi=31;OK')
})

test('a Kitty image reaches the screen', async ({ browser }) => {
  const page = await open(browser)

  // A 2 by 2 red image, transmitted and shown in one step.
  await page.keyboard.type(String.raw`printf '\033_Ga=T,f=24,s=2,v=2,c=8,r=4;/wAA/wAA/wAA/wAA\033\\'`)
  await page.keyboard.press('Enter')

  await expect.poll(() => page.evaluate(redPixels)).toBeGreaterThan(0)
})

/** redPixels counts the red pixels on the terminal's image layer. */
function redPixels(): number {
  let red = 0
  for (const canvas of document.querySelectorAll<HTMLCanvasElement>('.xterm-image-layer-top')) {
    const ctx = canvas.getContext('2d')
    if (!ctx) continue
    const px = ctx.getImageData(0, 0, canvas.width, canvas.height).data
    for (let i = 0; i < px.length; i += 4) {
      if (px[i] > 200 && px[i + 1] < 50 && px[i + 2] < 50 && px[i + 3] > 200) red++
    }
  }
  return red
}
