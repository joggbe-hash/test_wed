import { describe, expect, it } from 'vitest'
import { ApiError } from './backendApi'
import {
  apiErrorMessage,
  loginErrorMessage,
  registrationErrorMessage,
  verificationSendErrorMessage,
} from './errors'

describe('apiErrorMessage', () => {
  it('maps known HTTP statuses to stable user-facing messages', () => {
    expect(apiErrorMessage(new ApiError(413, 'raw backend message'))).toContain('大小限制')
    expect(apiErrorMessage(new ApiError(429, 'raw backend message'))).toContain('太頻繁')
  })

  it('uses the caller fallback for unknown errors', () => {
    expect(apiErrorMessage(new Error('internal detail'), '自訂訊息')).toBe('自訂訊息')
  })

  it('distinguishes invalid login credentials from an expired session', () => {
    const unauthorized = new ApiError(401, 'invalid email or password')

    expect(loginErrorMessage(unauthorized)).toBe('電子信箱或密碼不正確')
    expect(apiErrorMessage(unauthorized)).toBe('登入狀態已失效，請重新登入。')
  })

  it('explains the temporary login limit without exposing account state', () => {
    const limited = new ApiError(429, 'too many login attempts')

    expect(loginErrorMessage(limited)).toBe('登入嘗試次數過多，請稍後再試（最長約 5 分鐘）。')
  })

  it('explains verification send cooldowns and longer send limits', () => {
    const limited = new ApiError(429, 'send limit reached')

    expect(verificationSendErrorMessage(limited)).toBe(
      '驗證碼傳送太頻繁。每次需間隔 60 秒；若已達寄送上限，請稍後再試。',
    )
  })

  it('explains invalid and rate-limited registration codes', () => {
    expect(registrationErrorMessage(new ApiError(400, 'invalid verification code'))).toBe(
      '驗證碼不正確、已過期，或不是最新一封，請確認後再試。',
    )
    expect(registrationErrorMessage(new ApiError(429, 'too many verification attempts'))).toBe(
      '驗證碼錯誤次數過多，請使用最新一封的正確驗證碼，或重新取得驗證碼。',
    )
  })
})
