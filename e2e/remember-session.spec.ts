import { expect, test } from '@playwright/test'

for (const scenario of [
  { checked: false, expectedRemember: false },
  { checked: true, expectedRemember: true },
]) {
  test(`login sends remember=${scenario.expectedRemember}`, async ({ page }) => {
    let rememberValue: boolean | undefined
    let isLoggedIn = false

    await page.route('**/api/auth/session', (route) => route.fulfill(isLoggedIn
      ? {
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ user: { id: 7, username: 'remember-check' } }),
        }
      : {
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'unauthorized' }),
        }))
    await page.route('**/api/auth/login', async (route) => {
      const body = await route.request().postDataJSON()
      rememberValue = body.remember
      isLoggedIn = true
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ user: { id: 7, username: 'remember-check' } }),
      })
    })
    await page.route('**/api/feed', (route) => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ posts: [], next_cursor: '' }),
    }))

    await page.goto('/#/login')
    await expect(page.getByText('30 天未使用將自動登出')).toBeVisible()
    await page.locator('#login-email').fill('remember@example.test')
    await page.locator('#login-password').fill('Password1')
    if (scenario.checked) {
      await page.locator('#loginForm input[type="checkbox"]').check()
    }
    await page.locator('#loginForm button[type="submit"]').click()

    await expect(page).toHaveURL(/#\/home/)
    expect(rememberValue).toBe(scenario.expectedRemember)
  })
}
