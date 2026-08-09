import { computed, onBeforeMount, shallowRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ApiError,
  loginAccount,
  registerAccount,
  sendVerificationCode,
} from '../../api/backendApi'
import {
  apiErrorMessage,
  loginErrorMessage,
  registrationErrorMessage,
  verificationSendErrorMessage,
} from '../../api/errors'
import { useSession } from '../../composables/useSession'
import { publicAsset } from '../../utils/assets'
import { registrationPasswordError } from './passwordValidation'

export function useAuthPage() {
  const router = useRouter()
  const route = useRoute()
  const {
    currentUser,
    isSessionInitialized,
    restoreCurrentSession,
    setCurrentSession,
  } = useSession()

  const mode = shallowRef<'login' | 'register'>('login')
  const email = shallowRef('')
  const loginPassword = shallowRef('')
  const registerPassword = shallowRef('')
  const username = shallowRef('')
  const code = shallowRef('')
  const verificationChallengeId = shallowRef('')
  const verificationChallengeEmail = shallowRef('')
  const confirmPassword = shallowRef('')
  const isSubmitting = shallowRef(false)
  const statusMessage = shallowRef('')
  const errorMessage = shallowRef('')
  const showLoginPassword = shallowRef(false)
  const showRegisterPassword = shallowRef(false)
  const showConfirmPassword = shallowRef(false)
  const rememberMe = shallowRef(false)
  const isCheckingSession = shallowRef(true)

  const authBackgroundStyle = {
    '--auth-background-image': `url("${publicAsset('picture/meme_background.jpg')}")`,
  }
  const registerBackgroundStyle = {
    '--register-background-image': `url("${publicAsset('picture/register_round.jpg')}")`,
  }

  const trimmedEmail = computed(() => email.value.trim())
  const normalizedEmail = computed(() => trimmedEmail.value.toLowerCase())
  const trimmedUsername = computed(() => username.value.trim())
  const trimmedCode = computed(() => code.value.trim())
  const isRegisterEmailValid = computed(() => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmedEmail.value))
  const showRegisterEmailFormatError = computed(() => email.value.length > 0 && !isRegisterEmailValid.value)
  const isRegisterUsernameValid = computed(() =>
    trimmedUsername.value.length >= 2 &&
    trimmedUsername.value.length <= 20 &&
    !/\s/.test(trimmedUsername.value),
  )
  const showRegisterUsernameError = computed(() => username.value.length > 0 && !isRegisterUsernameValid.value)
  const canSendVerificationCode = computed(() => isRegisterEmailValid.value && !isSubmitting.value)
  const hasLoginPassword = computed(() => loginPassword.value.length > 0)
  const loginPasswordType = computed(() => showLoginPassword.value && hasLoginPassword.value ? 'text' : 'password')
  const loginPasswordToggleLabel = computed(() => showLoginPassword.value ? '隱藏密碼' : '顯示密碼')
  const hasRegisterPassword = computed(() => registerPassword.value.length > 0)
  const registerPasswordType = computed(() => showRegisterPassword.value && hasRegisterPassword.value ? 'text' : 'password')
  const registerPasswordToggleLabel = computed(() => showRegisterPassword.value ? '隱藏密碼' : '顯示密碼')
  const hasConfirmPassword = computed(() => confirmPassword.value.length > 0)
  const confirmPasswordType = computed(() => showConfirmPassword.value && hasConfirmPassword.value ? 'text' : 'password')
  const confirmPasswordToggleLabel = computed(() => showConfirmPassword.value ? '隱藏密碼' : '顯示密碼')
  const isRegisterPasswordMatched = computed(() =>
    registerPassword.value.length > 0 &&
    confirmPassword.value.length > 0 &&
    registerPassword.value === confirmPassword.value,
  )
  const showRegisterPasswordMismatch = computed(() =>
    confirmPassword.value.length > 0 && registerPassword.value !== confirmPassword.value,
  )
  const registerPasswordErrorMessage = computed(() => registrationPasswordError(registerPassword.value))
  const isRegisterPasswordStrong = computed(() => registerPasswordErrorMessage.value === '')
  const showRegisterPasswordStrengthError = computed(() => hasRegisterPassword.value && !isRegisterPasswordStrong.value)
  const canRegister = computed(() =>
    isRegisterEmailValid.value &&
    verificationChallengeId.value.length > 0 &&
    verificationChallengeEmail.value === normalizedEmail.value &&
    /^\d{6}$/.test(trimmedCode.value) &&
    isRegisterUsernameValid.value &&
    isRegisterPasswordStrong.value &&
    isRegisterPasswordMatched.value,
  )

  watch(loginPassword, (value) => {
    if (value.length === 0) showLoginPassword.value = false
  })
  watch(registerPassword, (value) => {
    if (value.length === 0) showRegisterPassword.value = false
  })
  watch(confirmPassword, (value) => {
    if (value.length === 0) showConfirmPassword.value = false
  })
  watch(normalizedEmail, (value) => {
    if (verificationChallengeEmail.value && value !== verificationChallengeEmail.value) {
      verificationChallengeId.value = ''
      verificationChallengeEmail.value = ''
      code.value = ''
    }
  })

  function setStatus(message: string) {
    statusMessage.value = message
    errorMessage.value = ''
  }

  function setFormError(message: string) {
    statusMessage.value = ''
    errorMessage.value = message
  }

  function setError(error: unknown) {
    setFormError(apiErrorMessage(error, '驗證失敗，請稍後再試。'))
  }

  function redirectAfterLogin() {
    const redirect = route.query.redirect
    return typeof redirect === 'string' && redirect.startsWith('/') && !redirect.startsWith('//')
      ? redirect
      : '/home'
  }

  async function handleLogin() {
    if (!trimmedEmail.value) return setFormError('請輸入電子信箱')
    if (!loginPassword.value) return setFormError('請輸入密碼')

    isSubmitting.value = true
    try {
      const { user } = await loginAccount(trimmedEmail.value, loginPassword.value, rememberMe.value)
      setCurrentSession(user)
      await router.replace(redirectAfterLogin())
    } catch (error) {
      setFormError(loginErrorMessage(error))
    } finally {
      isSubmitting.value = false
    }
  }

  async function handleSendCode() {
    if (!trimmedEmail.value) return setFormError('請先輸入電子信箱')
    if (!isRegisterEmailValid.value) return setFormError('電子信箱格式不正確')
    isSubmitting.value = true
    try {
      const response = await sendVerificationCode(trimmedEmail.value)
      verificationChallengeId.value = response.challenge_id
      verificationChallengeEmail.value = normalizedEmail.value
      code.value = ''
      setStatus('驗證碼已送出，請查看最新一封信。')
    } catch (error) {
      setFormError(verificationSendErrorMessage(error))
    } finally {
      isSubmitting.value = false
    }
  }

  async function handleRegister() {
    if (isRegisterEmailValid.value && (!verificationChallengeId.value || verificationChallengeEmail.value !== normalizedEmail.value)) {
      return setFormError('請先取得這個電子信箱的驗證碼')
    }
    if (!isRegisterEmailValid.value) return setFormError('請輸入正確的電子信箱')
    if (!trimmedCode.value) return setFormError('請輸入驗證碼')
    if (!isRegisterUsernameValid.value) return setFormError('使用者名稱需為 2-20 個字，且不能有空白')
    if (!isRegisterPasswordStrong.value) return setFormError(registerPasswordErrorMessage.value)
    if (!isRegisterPasswordMatched.value) return setFormError('兩次密碼不一致')
    isSubmitting.value = true
    try {
      await registerAccount({
        username: trimmedUsername.value,
        email: trimmedEmail.value,
        password: registerPassword.value,
        code: trimmedCode.value,
        challenge_id: verificationChallengeId.value,
      })
      setStatus('註冊成功，請登入')
      mode.value = 'login'
    } catch (error) {
      setFormError(registrationErrorMessage(error))
    } finally {
      isSubmitting.value = false
    }
  }

  onBeforeMount(async () => {
    try {
      const force = isSessionInitialized.value && currentUser.value !== null
      const user = await restoreCurrentSession({ force })
      if (user) await router.replace(redirectAfterLogin())
    } catch (error) {
      if (
        route.query.sessionError === 'unavailable' ||
        (error instanceof ApiError && (error.status === 429 || error.status === 503))
      ) {
        setFormError('登入服務暫時無法使用，請稍後再試')
      } else {
        setError(error)
      }
    } finally {
      isCheckingSession.value = false
    }
  })

  return {
    mode, email, loginPassword, registerPassword, username, code, confirmPassword,
    isSubmitting, statusMessage, errorMessage, showLoginPassword, showRegisterPassword,
    showConfirmPassword, rememberMe, isCheckingSession, authBackgroundStyle,
    registerBackgroundStyle, showRegisterEmailFormatError, showRegisterUsernameError,
    canSendVerificationCode, hasLoginPassword, loginPasswordType, loginPasswordToggleLabel,
    hasRegisterPassword, registerPasswordType, registerPasswordToggleLabel,
    hasConfirmPassword, confirmPasswordType, confirmPasswordToggleLabel,
    showRegisterPasswordMismatch, showRegisterPasswordStrengthError, registerPasswordErrorMessage, canRegister,
    handleLogin, handleSendCode, handleRegister,
  }
}
