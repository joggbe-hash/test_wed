import { expect, test } from '@playwright/test'

test('user can log in and reach the home feed', async ({ page }) => {
  await page.route('**/api/auth/session', (route) =>
    route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'unauthorized' }),
    }),
  )
  await page.route('**/api/auth/login', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        user: { id: 1, username: '測試使用者', email: 'test@example.com' },
      }),
    }),
  )
  await page.route('**/api/feed', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ posts: [], next_cursor: '' }),
    }),
  )

  await page.goto('/#/login')
  const loginForm = page.locator('#loginForm')
  await loginForm.getByLabel('信箱', { exact: true }).fill('test@example.com')
  await loginForm.getByLabel('密碼', { exact: true }).fill('Password1')
  await loginForm.getByRole('button', { name: '登入', exact: true }).click()

  await expect(page).toHaveURL(/#\/home/)
})
