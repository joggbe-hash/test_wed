import { expect, test } from '@playwright/test'

test('logout revokes only the current browser session and returns to login', async ({ page }) => {
  let logoutRequest: { method: string; browserRequestHeader: string | null } | null = null

  await page.route('**/api/auth/session', (route) => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({
      user: {
        id: 7,
        username: 'security-check',
        email: 'must-not-render@example.test',
      },
    }),
  }))
  await page.route('**/api/auth/logout', async (route) => {
    const request = route.request()
    logoutRequest = {
      method: request.method(),
      browserRequestHeader: request.headers()['x-type-wsp-request'] ?? null,
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ message: 'logged out' }),
    })
  })

  await page.goto('/#/freq')
  await expect(page.locator('body')).not.toContainText('must-not-render@example.test')
  await expect(page.locator('[data-logout-all]')).toHaveCount(0)
  await page.locator('[data-logout-current]').click()

  await expect(page).toHaveURL(/#\/login$/)
  expect(logoutRequest).toEqual({ method: 'POST', browserRequestHeader: '1' })
})
