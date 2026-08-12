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

test('user must verify the emailed code before reaching the home feed', async ({ page }) => {
  let hasVerifiedSession = false
  await page.route('**/api/auth/session', (route) => route.fulfill({
    status: hasVerifiedSession ? 200 : 401,
    contentType: 'application/json',
    body: JSON.stringify(hasVerifiedSession
      ? { user: { id: 1, username: '測試使用者' } }
      : { error: 'unauthorized' }),
  }))
  await page.route('**/api/auth/login', (route) =>
    route.fulfill({
      status: 202,
      contentType: 'application/json',
      body: JSON.stringify({
        message: 'login verification code sent',
        challenge_id: 'login-challenge-1',
        requires_verification: true,
        expires_in_seconds: 300,
      }),
    }),
  )
  await page.route('**/api/auth/login/verify', async (route) => {
    expect(await route.request().postDataJSON()).toEqual({
      email: 'test@example.com',
      code: '123456',
      challenge_id: 'login-challenge-1',
      remember: false,
    })
    hasVerifiedSession = true
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        user: { id: 1, username: '測試使用者' },
      }),
    })
  })
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

  await expect(page).toHaveURL(/#\/login/)
  await expect(loginForm.getByText('完成驗證前不會建立登入 Session')).toBeVisible()
  await loginForm.getByLabel('登入驗證碼').fill('123456')
  await loginForm.getByRole('button', { name: '驗證並登入', exact: true }).click()

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
      body: JSON.stringify({ message: '驗證碼已寄出', challenge_id: 'register-challenge-1' }),
    })
  })
  await page.route('**/api/auth/register', async (route) => {
    expect(await route.request().postDataJSON()).toEqual({
      email: 'new-user@example.com',
      code: 'ABCDEFGHJKLMNPQ2',
      username: '新使用者',
      password: 'Password1',
      challenge_id: 'register-challenge-1',
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
  await expect(registerForm.getByRole('status')).toContainText('驗證碼已送出')
  await registerForm.getByLabel('驗證碼').fill('ABCDEFGHJKLMNPQ2')
  await registerForm.getByLabel('使用者名稱').fill('新使用者')
  await registerForm.getByLabel('密碼', { exact: true }).fill('Password1')
  await registerForm.getByLabel('確認密碼').fill('Password1')
  await registerForm.getByRole('button', { name: '確認註冊', exact: true }).click()

  await expect(page.locator('#loginForm')).toBeVisible()
  await expect(page.locator('#loginForm').getByRole('status')).toContainText('註冊成功')
})
