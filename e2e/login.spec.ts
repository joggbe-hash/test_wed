import { expect, test } from '@playwright/test'

test('empty login form is rejected before calling the login API', async ({ page }) => {
  let loginRequestCount = 0

  await page.route('**/api/auth/session', (route) =>
    route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'unauthorized' }),
    }),
  )
  await page.route('**/api/auth/login', (route) => {
    loginRequestCount += 1
    return route.fulfill({
      status: 400,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'missing credentials' }),
    })
  })

  await page.goto('/#/login')
  const loginForm = page.locator('#loginForm')
  const emailInput = loginForm.getByLabel('信箱', { exact: true })
  await loginForm.getByRole('button', { name: '登入', exact: true }).click()

  await expect(emailInput).toBeFocused()
  await expect(emailInput).toHaveJSProperty('validity.valueMissing', true)
  await expect(loginForm.getByRole('alert')).toHaveCount(0)
  expect(loginRequestCount).toBe(0)
})

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

test('user can request a verification code and register', async ({ page }) => {
  await page.route('**/api/auth/session', (route) =>
    route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'unauthorized' }),
    }),
  )
  await page.route('**/api/auth/send-code', async (route) => {
    expect((await route.request().postDataJSON()).email).toBe('new-user@example.com')
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ message: '驗證碼已寄出' }),
    })
  })
  await page.route('**/api/auth/register', async (route) => {
    expect(await route.request().postDataJSON()).toEqual({
      email: 'new-user@example.com',
      code: '123456',
      username: '新使用者',
      password: 'Password1',
    })
    await route.fulfill({
      status: 201,
      contentType: 'application/json',
      body: JSON.stringify({ message: 'registered' }),
    })
  })

  await page.goto('/#/login')
  await page.locator('#loginForm').getByRole('button', { name: '註冊', exact: true }).click()
  const registerForm = page.locator('#registerForm')
  await registerForm.getByLabel('電子信箱').fill('new-user@example.com')
  await registerForm.getByRole('button', { name: '取得驗證碼' }).click()
  await expect(registerForm.getByRole('status')).toContainText('驗證碼已寄出')
  await registerForm.getByLabel('驗證碼').fill('123456')
  await registerForm.getByLabel('使用者名稱').fill('新使用者')
  await registerForm.getByLabel('密碼', { exact: true }).fill('Password1')
  await registerForm.getByLabel('確認密碼').fill('Password1')
  await registerForm.getByRole('button', { name: '確認註冊', exact: true }).click()

  await expect(page.locator('#loginForm')).toBeVisible()
  await expect(page.locator('#loginForm').getByRole('status')).toContainText('註冊成功')
})
