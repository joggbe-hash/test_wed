import { expect, test } from '@playwright/test'

for (const scenario of [
  { checked: false, expectedRemember: false },
  { checked: true, expectedRemember: true },
]) {
  test(`login verification preserves remember=${scenario.expectedRemember}`, async ({ page }) => {
    let loginRememberValue: boolean | undefined
    let verificationRememberValue: boolean | undefined
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
      loginRememberValue = body.remember
      await route.fulfill({
        status: 202,
        contentType: 'application/json',
        body: JSON.stringify({
          message: 'login verification code sent',
          challenge_id: 'remember-challenge-1',
          requires_verification: true,
          expires_in_seconds: 300,
        }),
      })
    })
    await page.route('**/api/auth/login/verify', async (route) => {
      const body = await route.request().postDataJSON()
      verificationRememberValue = body.remember
      expect(body).toMatchObject({
        email: 'remember@example.test',
        code: '123456',
        challenge_id: 'remember-challenge-1',
      })
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

    await page.getByLabel('登入驗證碼').fill('123456')
    await page.getByRole('button', { name: '驗證並登入', exact: true }).click()

    await expect(page).toHaveURL(/#\/home/)
    expect(loginRememberValue).toBe(scenario.expectedRemember)
    expect(verificationRememberValue).toBe(scenario.expectedRemember)
  })
}
