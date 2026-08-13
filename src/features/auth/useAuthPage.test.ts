import { defineComponent, h, nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../../api/backendApi'
import { useAuthPage } from './useAuthPage'

const mocks = vi.hoisted(() => ({
  loginAccount: vi.fn(),
  verifyLoginOwnership: vi.fn(),
  verifyLoginAccount: vi.fn(),
  registerAccount: vi.fn(),
  sendVerificationCode: vi.fn(),
  setCurrentSession: vi.fn(),
  replace: vi.fn(),
}))

vi.mock('../../api/backendApi', () => ({
  ApiError: class ApiError extends Error {
    code: string | undefined
    ownershipChallenge: unknown

    constructor(
      public status: number,
      message: string,
      details: { code?: string; ownershipChallenge?: unknown } = {},
    ) {
      super(message)
      this.code = details.code
      this.ownershipChallenge = details.ownershipChallenge
    }
  },
  loginAccount: mocks.loginAccount,
  verifyLoginOwnership: mocks.verifyLoginOwnership,
  verifyLoginAccount: mocks.verifyLoginAccount,
  registerAccount: mocks.registerAccount,
  sendVerificationCode: mocks.sendVerificationCode,
}))

vi.mock('../../composables/useSession', async () => {
  const { shallowRef } = await import('vue')
  return {
    useSession: () => ({
      currentUser: shallowRef(null),
      isSessionInitialized: shallowRef(false),
      restoreCurrentSession: vi.fn().mockResolvedValue(null),
      setCurrentSession: mocks.setCurrentSession,
    }),
  }
})

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ replace: mocks.replace }),
}))

vi.mock('../../utils/assets', () => ({ publicAsset: (path: string) => path }))

describe('useAuthPage verification challenge', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.registerAccount.mockResolvedValue({ message: 'registered', user_id: 1 })
  })

  it('uses only the latest challenge and invalidates it when the email changes', async () => {
    mocks.sendVerificationCode
      .mockResolvedValueOnce({ message: 'sent', challenge_id: 'challenge-1' })
      .mockResolvedValueOnce({ message: 'sent', challenge_id: 'challenge-2' })

    let auth!: ReturnType<typeof useAuthPage>
    mount(defineComponent({
      setup() {
        auth = useAuthPage()
        return () => h('div')
      },
    }))
    await flushPromises()

    auth.email.value = 'first@example.com'
    await auth.handleSendCode()
    auth.code.value = 'ABCDEFGHJKLMNPQR'
    auth.username.value = 'tester'
    auth.registerPassword.value = 'password1'
    auth.confirmPassword.value = 'password1'
    expect(auth.canRegister.value).toBe(true)

    auth.email.value = 'second@example.com'
    await nextTick()
    expect(auth.code.value).toBe('')
    expect(auth.canRegister.value).toBe(false)

    await auth.handleSendCode()
    auth.code.value = '23456789ABCDEFGH'
    await auth.handleRegister()

    expect(mocks.registerAccount).toHaveBeenCalledWith(expect.objectContaining({
      email: 'second@example.com',
      code: '23456789ABCDEFGH',
      challenge_id: 'challenge-2',
    }))
  })

  it('requires the email code before creating the browser session', async () => {
    mocks.loginAccount.mockResolvedValueOnce({
      message: 'login verification code sent',
      challenge_id: 'login-challenge-1',
      requires_verification: true,
      expires_in_seconds: 300,
    })
    mocks.verifyLoginAccount.mockResolvedValueOnce({
      user: { id: 7, username: 'tester' },
    })

    let auth!: ReturnType<typeof useAuthPage>
    mount(defineComponent({
      setup() {
        auth = useAuthPage()
        return () => h('div')
      },
    }))
    await flushPromises()

    auth.email.value = 'user@example.com'
    auth.loginPassword.value = 'Password1'
    await auth.handleLogin()

    expect(auth.loginStage.value).toBe('verification')
    expect(auth.loginChallengeId.value).toBe('login-challenge-1')
    expect(auth.loginPassword.value).toBe('')
    expect(mocks.setCurrentSession).not.toHaveBeenCalled()
    expect(mocks.replace).not.toHaveBeenCalled()

    auth.loginCode.value = '123456'
    await auth.handleVerifyLogin()

    expect(mocks.verifyLoginAccount).toHaveBeenCalledWith(
      {
        email: 'user@example.com',
        code: '123456',
        challenge_id: 'login-challenge-1',
        remember: false,
      },
      expect.any(AbortSignal),
    )
    expect(mocks.setCurrentSession).toHaveBeenCalledWith({ id: 7, username: 'tester' })
    expect(mocks.replace).toHaveBeenCalledWith('/home')
  })

  it('keeps the verification challenge retryable when the login response is invalid', async () => {
    mocks.loginAccount.mockResolvedValueOnce({
      message: 'login verification code sent',
      challenge_id: 'login-challenge-1',
      requires_verification: true,
      expires_in_seconds: 300,
    })
    mocks.verifyLoginAccount.mockRejectedValueOnce(new ApiError(200, 'Invalid login response'))

    let auth!: ReturnType<typeof useAuthPage>
    mount(defineComponent({
      setup() {
        auth = useAuthPage()
        return () => h('div')
      },
    }))
    await flushPromises()

    auth.email.value = 'user@example.com'
    auth.loginPassword.value = 'Password1'
    await auth.handleLogin()
    auth.loginCode.value = '123456'
    await auth.handleVerifyLogin()

    expect(auth.loginStage.value).toBe('verification')
    expect(auth.loginChallengeId.value).toBe('login-challenge-1')
    expect(auth.loginCode.value).toBe('123456')
    expect(mocks.setCurrentSession).not.toHaveBeenCalled()
    expect(mocks.replace).not.toHaveBeenCalled()
  })

  it('verifies email ownership and retries with the same in-memory credentials', async () => {
    const ownershipChallenge = {
      challenge_id: 'ownership-challenge-1',
      code_format: 'base32-16-v1' as const,
      expires_in_seconds: 86400,
    }
    const grant = 'A'.repeat(43)
    mocks.loginAccount
      .mockRejectedValueOnce(new ApiError(429, 'email ownership required', {
        code: 'LOGIN_EMAIL_OWNERSHIP_REQUIRED',
        ownershipChallenge,
      }))
      .mockResolvedValueOnce({
        message: 'login verification code sent',
        challenge_id: 'login-challenge-after-ownership',
        requires_verification: true,
        expires_in_seconds: 300,
      })
    mocks.verifyLoginOwnership.mockResolvedValueOnce({
      password_verification_grant: grant,
      expires_in_seconds: 300,
      max_attempts: 3,
    })
    mocks.verifyLoginAccount.mockResolvedValueOnce({
      user: { id: 7, username: 'tester' },
    })

    let auth!: ReturnType<typeof useAuthPage>
    mount(defineComponent({
      setup() {
        auth = useAuthPage()
        return () => h('div')
      },
    }))
    await flushPromises()

    auth.email.value = 'user@example.com'
    auth.loginPassword.value = 'Password1'
    auth.rememberMe.value = true
    await auth.handleLogin()

    expect(auth.loginStage.value).toBe('ownership')
    expect(auth.ownershipChallengeId.value).toBe('ownership-challenge-1')
    expect(auth.loginPassword.value).toBe('Password1')
    expect(mocks.setCurrentSession).not.toHaveBeenCalled()

    auth.rememberMe.value = false
    auth.ownershipCode.value = 'abcdefghjklmnpq2'
    await auth.handleLoginSubmit()

    expect(mocks.verifyLoginOwnership).toHaveBeenCalledWith({
      email: 'user@example.com',
      challenge_id: 'ownership-challenge-1',
      code: 'ABCDEFGHJKLMNPQ2',
    }, expect.any(AbortSignal))
    expect(mocks.loginAccount).toHaveBeenNthCalledWith(
      1,
      'user@example.com',
      'Password1',
      true,
      { signal: expect.any(AbortSignal) },
    )
    expect(mocks.loginAccount).toHaveBeenNthCalledWith(
      2,
      'user@example.com',
      'Password1',
      true,
      { passwordVerificationGrant: grant, signal: expect.any(AbortSignal) },
    )
    expect(auth.loginStage.value).toBe('verification')
    expect(auth.loginChallengeId.value).toBe('login-challenge-after-ownership')
    expect(auth.loginPassword.value).toBe('')
    expect(auth.hasActiveOwnershipGrant.value).toBe(false)
    expect(mocks.setCurrentSession).not.toHaveBeenCalled()

    auth.loginCode.value = '123456'
    await auth.handleVerifyLogin()

    expect(mocks.verifyLoginAccount).toHaveBeenCalledWith(
      {
        email: 'user@example.com',
        code: '123456',
        challenge_id: 'login-challenge-after-ownership',
        remember: true,
      },
      expect.any(AbortSignal),
    )
  })

  it('does not commit a deferred OTP response after restarting the login flow', async () => {
    mocks.loginAccount.mockResolvedValueOnce({
      message: 'login verification code sent',
      challenge_id: 'login-challenge-1',
      requires_verification: true,
      expires_in_seconds: 300,
    })
    let resolveVerification!: (value: { user: { id: number; username: string } }) => void
    let verificationSignal: AbortSignal | undefined
    mocks.verifyLoginAccount.mockImplementationOnce((_payload: unknown, signal?: AbortSignal) => {
      verificationSignal = signal
      return new Promise((resolve) => {
        resolveVerification = resolve
      })
    })

    let auth!: ReturnType<typeof useAuthPage>
    mount(defineComponent({
      setup() {
        auth = useAuthPage()
        return () => h('div')
      },
    }))
    await flushPromises()

    auth.email.value = 'user@example.com'
    auth.loginPassword.value = 'Password1'
    await auth.handleLogin()
    auth.loginCode.value = '123456'
    const pendingVerification = auth.handleVerifyLogin()
    await flushPromises()

    auth.resetLoginVerification()
    expect(verificationSignal?.aborted).toBe(true)
    resolveVerification({ user: { id: 7, username: 'tester' } })
    await pendingVerification

    expect(auth.loginStage.value).toBe('credentials')
    expect(mocks.setCurrentSession).not.toHaveBeenCalled()
    expect(mocks.replace).not.toHaveBeenCalled()
  })

  it('does not commit a deferred OTP response after the auth page unmounts', async () => {
    mocks.loginAccount.mockResolvedValueOnce({
      message: 'login verification code sent',
      challenge_id: 'login-challenge-1',
      requires_verification: true,
      expires_in_seconds: 300,
    })
    let resolveVerification!: (value: { user: { id: number; username: string } }) => void
    let verificationSignal: AbortSignal | undefined
    mocks.verifyLoginAccount.mockImplementationOnce((_payload: unknown, signal?: AbortSignal) => {
      verificationSignal = signal
      return new Promise((resolve) => {
        resolveVerification = resolve
      })
    })

    let auth!: ReturnType<typeof useAuthPage>
    const wrapper = mount(defineComponent({
      setup() {
        auth = useAuthPage()
        return () => h('div')
      },
    }))
    await flushPromises()

    auth.email.value = 'user@example.com'
    auth.loginPassword.value = 'Password1'
    await auth.handleLogin()
    auth.loginCode.value = '123456'
    const pendingVerification = auth.handleVerifyLogin()
    await flushPromises()

    wrapper.unmount()
    expect(verificationSignal?.aborted).toBe(true)
    resolveVerification({ user: { id: 7, username: 'tester' } })
    await pendingVerification

    expect(mocks.setCurrentSession).not.toHaveBeenCalled()
    expect(mocks.replace).not.toHaveBeenCalled()
  })

  it('returns to credentials after a 401 and reuses a still-valid grant', async () => {
    const ownershipChallenge = {
      challenge_id: 'ownership-challenge-1',
      code_format: 'base32-16-v1' as const,
      expires_in_seconds: 86400,
    }
    const grant = 'B'.repeat(43)
    mocks.loginAccount
      .mockRejectedValueOnce(new ApiError(429, 'email ownership required', {
        code: 'LOGIN_EMAIL_OWNERSHIP_REQUIRED',
        ownershipChallenge,
      }))
      .mockRejectedValueOnce(new ApiError(401, 'invalid credentials'))
      .mockResolvedValueOnce({
        message: 'login verification code sent',
        challenge_id: 'login-challenge-after-correction',
        requires_verification: true,
        expires_in_seconds: 300,
      })
    mocks.verifyLoginOwnership.mockResolvedValueOnce({
      password_verification_grant: grant,
      expires_in_seconds: 300,
      max_attempts: 3,
    })

    let auth!: ReturnType<typeof useAuthPage>
    mount(defineComponent({
      setup() {
        auth = useAuthPage()
        return () => h('div')
      },
    }))
    await flushPromises()

    auth.email.value = 'user@example.com'
    auth.loginPassword.value = 'WrongPassword1'
    await auth.handleLogin()
    auth.ownershipCode.value = 'ABCDEFGHJKLMNPQ2'
    await auth.handleLoginSubmit()

    expect(auth.loginStage.value).toBe('credentials')
    expect(auth.loginPassword.value).toBe('WrongPassword1')
    expect(auth.hasActiveOwnershipGrant.value).toBe(true)
    expect(auth.errorMessage.value).toContain('電子信箱或密碼不正確')

    auth.loginPassword.value = 'CorrectPassword1'
    await auth.handleLogin()

    expect(mocks.loginAccount).toHaveBeenNthCalledWith(
      3,
      'user@example.com',
      'CorrectPassword1',
      false,
      { passwordVerificationGrant: grant, signal: expect.any(AbortSignal) },
    )
    expect(auth.loginStage.value).toBe('verification')
    expect(auth.loginChallengeId.value).toBe('login-challenge-after-correction')
    expect(auth.hasActiveOwnershipGrant.value).toBe(false)
  })

  it('replaces an invalid or expired grant with the new structured challenge', async () => {
    const firstChallenge = {
      challenge_id: 'ownership-challenge-1',
      code_format: 'base32-16-v1' as const,
      expires_in_seconds: 86400,
    }
    const replacementChallenge = {
      challenge_id: 'ownership-challenge-2',
      code_format: 'base32-16-v1' as const,
      expires_in_seconds: 86400,
    }
    mocks.loginAccount
      .mockRejectedValueOnce(new ApiError(429, 'email ownership required', {
        code: 'LOGIN_EMAIL_OWNERSHIP_REQUIRED',
        ownershipChallenge: firstChallenge,
      }))
      .mockRejectedValueOnce(new ApiError(429, 'grant expired', {
        code: 'LOGIN_EMAIL_OWNERSHIP_REQUIRED',
        ownershipChallenge: replacementChallenge,
      }))
    mocks.verifyLoginOwnership.mockResolvedValueOnce({
      password_verification_grant: 'C'.repeat(43),
      expires_in_seconds: 300,
      max_attempts: 3,
    })

    let auth!: ReturnType<typeof useAuthPage>
    mount(defineComponent({
      setup() {
        auth = useAuthPage()
        return () => h('div')
      },
    }))
    await flushPromises()

    auth.email.value = 'user@example.com'
    auth.loginPassword.value = 'Password1'
    await auth.handleLogin()
    auth.ownershipCode.value = 'ABCDEFGHJKLMNPQ2'
    await auth.handleLoginSubmit()

    expect(mocks.loginAccount).toHaveBeenCalledTimes(2)
    expect(auth.loginStage.value).toBe('ownership')
    expect(auth.ownershipChallengeId.value).toBe('ownership-challenge-2')
    expect(auth.ownershipCode.value).toBe('')
    expect(auth.loginPassword.value).toBe('Password1')
    expect(auth.hasActiveOwnershipGrant.value).toBe(false)
  })

  it('clears a retained grant when the email changes', async () => {
    const ownershipChallenge = {
      challenge_id: 'ownership-challenge-1',
      code_format: 'base32-16-v1' as const,
      expires_in_seconds: 86400,
    }
    mocks.loginAccount
      .mockRejectedValueOnce(new ApiError(429, 'email ownership required', {
        code: 'LOGIN_EMAIL_OWNERSHIP_REQUIRED',
        ownershipChallenge,
      }))
      .mockRejectedValueOnce(new ApiError(401, 'invalid credentials'))
    mocks.verifyLoginOwnership.mockResolvedValueOnce({
      password_verification_grant: 'D'.repeat(43),
      expires_in_seconds: 300,
      max_attempts: 3,
    })

    let auth!: ReturnType<typeof useAuthPage>
    mount(defineComponent({
      setup() {
        auth = useAuthPage()
        return () => h('div')
      },
    }))
    await flushPromises()

    auth.email.value = 'user@example.com'
    auth.loginPassword.value = 'WrongPassword1'
    await auth.handleLogin()
    auth.ownershipCode.value = 'ABCDEFGHJKLMNPQ2'
    await auth.handleLoginSubmit()
    expect(auth.hasActiveOwnershipGrant.value).toBe(true)

    auth.email.value = 'other@example.com'
    await nextTick()

    expect(auth.hasActiveOwnershipGrant.value).toBe(false)
  })

  it('clears the password and retained grant when switching to registration', async () => {
    const ownershipChallenge = {
      challenge_id: 'ownership-challenge-1',
      code_format: 'base32-16-v1' as const,
      expires_in_seconds: 86400,
    }
    mocks.loginAccount
      .mockRejectedValueOnce(new ApiError(429, 'email ownership required', {
        code: 'LOGIN_EMAIL_OWNERSHIP_REQUIRED',
        ownershipChallenge,
      }))
      .mockRejectedValueOnce(new ApiError(401, 'invalid credentials'))
    mocks.verifyLoginOwnership.mockResolvedValueOnce({
      password_verification_grant: 'F'.repeat(43),
      expires_in_seconds: 300,
      max_attempts: 3,
    })

    let auth!: ReturnType<typeof useAuthPage>
    mount(defineComponent({
      setup() {
        auth = useAuthPage()
        return () => h('div')
      },
    }))
    await flushPromises()

    auth.email.value = 'user@example.com'
    auth.loginPassword.value = 'WrongPassword1'
    await auth.handleLogin()
    auth.ownershipCode.value = 'ABCDEFGHJKLMNPQ2'
    await auth.handleLoginSubmit()
    expect(auth.hasActiveOwnershipGrant.value).toBe(true)

    auth.mode.value = 'register'
    await nextTick()

    expect(auth.loginPassword.value).toBe('')
    expect(auth.hasActiveOwnershipGrant.value).toBe(false)
    expect(auth.ownershipChallengeId.value).toBe('')
    expect(auth.loginStage.value).toBe('credentials')
  })

  it('keeps a new login attempt authoritative after cancelling an older ownership request', async () => {
    const ownershipChallenge = {
      challenge_id: 'ownership-challenge-old',
      code_format: 'base32-16-v1' as const,
      expires_in_seconds: 86400,
    }
    let resolveOldOwnership!: (value: {
      password_verification_grant: string
      expires_in_seconds: number
      max_attempts: number
    }) => void
    mocks.loginAccount
      .mockRejectedValueOnce(new ApiError(429, 'email ownership required', {
        code: 'LOGIN_EMAIL_OWNERSHIP_REQUIRED',
        ownershipChallenge,
      }))
      .mockResolvedValueOnce({
        message: 'login verification code sent',
        challenge_id: 'new-login-challenge',
        requires_verification: true,
        expires_in_seconds: 300,
      })
    mocks.verifyLoginOwnership.mockImplementationOnce(() => new Promise((resolve) => {
      resolveOldOwnership = resolve
    }))

    let auth!: ReturnType<typeof useAuthPage>
    mount(defineComponent({
      setup() {
        auth = useAuthPage()
        return () => h('div')
      },
    }))
    await flushPromises()

    auth.email.value = 'old@example.com'
    auth.loginPassword.value = 'OldPassword1'
    await auth.handleLogin()
    auth.ownershipCode.value = 'ABCDEFGHJKLMNPQ2'
    const oldAttempt = auth.handleLoginSubmit()
    await flushPromises()
    expect(auth.isSubmitting.value).toBe(true)

    auth.cancelLoginAttempt()
    auth.email.value = 'new@example.com'
    auth.loginPassword.value = 'NewPassword1'
    const newAttempt = auth.handleLogin()
    await newAttempt

    resolveOldOwnership({
      password_verification_grant: 'E'.repeat(43),
      expires_in_seconds: 300,
      max_attempts: 3,
    })
    await oldAttempt

    expect(mocks.loginAccount).toHaveBeenCalledTimes(2)
    expect(auth.loginStage.value).toBe('verification')
    expect(auth.loginChallengeId.value).toBe('new-login-challenge')
    expect(auth.errorMessage.value).toBe('')
    expect(auth.isSubmitting.value).toBe(false)
    expect(auth.hasActiveOwnershipGrant.value).toBe(false)
  })

  it('shows actionable messages for each authentication rate limit', async () => {
    let auth!: ReturnType<typeof useAuthPage>
    mount(defineComponent({
      setup() {
        auth = useAuthPage()
        return () => h('div')
      },
    }))
    await flushPromises()

    mocks.loginAccount.mockRejectedValueOnce(new ApiError(429, 'too many login attempts'))
    auth.email.value = 'user@example.com'
    auth.loginPassword.value = 'password1'
    await auth.handleLogin()
    expect(auth.errorMessage.value).toContain('最長約 5 分鐘')
    expect(mocks.verifyLoginOwnership).not.toHaveBeenCalled()

    mocks.sendVerificationCode.mockRejectedValueOnce(new ApiError(429, 'send limit reached'))
    await auth.handleSendCode()
    expect(auth.errorMessage.value).toContain('間隔 60 秒')

    mocks.sendVerificationCode.mockResolvedValueOnce({
      message: 'verification code sent',
      challenge_id: 'challenge-1',
    })
    await auth.handleSendCode()
    expect(auth.statusMessage.value).toBe('驗證碼已送出，請查看最新一封信。')

    mocks.registerAccount.mockRejectedValueOnce(new ApiError(429, 'too many verification attempts'))
    auth.code.value = 'ABCDEFGHJKLMNPQR'
    auth.username.value = 'tester'
    auth.registerPassword.value = 'password1'
    auth.confirmPassword.value = 'password1'
    await auth.handleRegister()
    expect(auth.errorMessage.value).toContain('最新一封的正確驗證碼')
  })
})
