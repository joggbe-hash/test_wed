import { expect, test } from '@playwright/test'

const runtimeUser = {
  id: 9001,
  username: 'runtime-check',
  email: 'runtime-check@example.test',
}

test('protected pages render without runtime demo data', async ({ page }) => {
  const requestedUrls: string[] = []
  const consoleProblems: string[] = []
  page.on('request', (request) => requestedUrls.push(request.url()))
  page.on('console', (message) => {
    if (message.type() === 'error' || message.type() === 'warning') {
      consoleProblems.push(`${message.type()}: ${message.text()}`)
    }
  })
  page.on('pageerror', (error) => consoleProblems.push(`pageerror: ${error.message}`))

  await page.route('**/api/auth/session', (route) => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ user: runtimeUser }),
  }))
  await page.route('**/api/schedule', (route) => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ tasks: [], reminders: [] }),
  }))

  await page.goto('/#/explore')
  const skipDailyTaskPrompt = page.getByRole('button', { name: '跳過' })
  if (await skipDailyTaskPrompt.isVisible()) await skipDailyTaskPrompt.click()
  await expect(page.getByRole('heading', { name: '目前沒有探索資料' })).toBeVisible()
  await expect(page.getByText('探索功能尚未連接真實後端資料來源')).toBeVisible()

  await page.goto('/#/freq')
  await expect(page.getByText(runtimeUser.username, { exact: true })).toBeVisible()
  await expect(page.locator('body')).not.toContainText(runtimeUser.email)

  await page.goto('/#/introduce')
  await expect(page.getByRole('heading', { name: `@${runtimeUser.username}` })).toBeVisible()
  await expect(page.locator('body')).not.toContainText(runtimeUser.email)

  expect(requestedUrls.filter((url) => /timed|mock|demo/i.test(url))).toEqual([])
  expect(consoleProblems).toEqual([])
  await expect(page.locator('body')).not.toContainText('demo.type-wsp.local')
})
