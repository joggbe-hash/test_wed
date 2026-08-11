import { defineComponent, h, nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../../api/backendApi'
import { useAuthPage } from './useAuthPage'

const mocks = vi.hoisted(() => ({
  loginAccount: vi.fn(),
  verifyLoginAccount: vi.fn(),
  registerAccount: vi.fn(),
  sendVerificationCode: vi.fn(),
  setCurrentSession: vi.fn(),
  replace: vi.fn(),
}))

vi.mock('../../api/backendApi', () => ({
  ApiError: class ApiError extends Error {
    constructor(public status: number, message: string) {
      super(message)
    }
  },
  loginAccount: mocks.loginAccount,
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
    auth.code.value = '123456'
    auth.username.value = 'tester'
    auth.registerPassword.value = 'password1'
    auth.confirmPassword.value = 'password1'
    expect(auth.canRegister.value).toBe(true)

    auth.email.value = 'second@example.com'
    await nextTick()
    expect(auth.code.value).toBe('')
    expect(auth.canRegister.value).toBe(false)

    await auth.handleSendCode()
    auth.code.value = '654321'
    await auth.handleRegister()

    expect(mocks.registerAccount).toHaveBeenCalledWith(expect.objectContaining({
      email: 'second@example.com',
      code: '654321',
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

    expect(mocks.verifyLoginAccount).toHaveBeenCalledWith({
      email: 'user@example.com',
      code: '123456',
      challenge_id: 'login-challenge-1',
      remember: false,
    })
    expect(mocks.setCurrentSession).toHaveBeenCalledWith({ id: 7, username: 'tester' })
    expect(mocks.replace).toHaveBeenCalledWith('/home')
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
    auth.code.value = '123456'
    auth.username.value = 'tester'
    auth.registerPassword.value = 'password1'
    auth.confirmPassword.value = 'password1'
    await auth.handleRegister()
    expect(auth.errorMessage.value).toContain('最新一封的正確驗證碼')
  })
})
