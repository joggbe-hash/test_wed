import { computed, onBeforeMount, onBeforeUnmount, shallowRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ApiError,
  loginAccount,
  registerAccount,
  sendVerificationCode,
  verifyLoginAccount,
  verifyLoginOwnership,
} from '../../api/backendApi'
import {
  apiErrorMessage,
  loginErrorMessage,
  loginVerificationErrorMessage,
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
  const loginStage = shallowRef<'credentials' | 'ownership' | 'verification'>('credentials')
  const loginCode = shallowRef('')
  const loginChallengeId = shallowRef('')
  const loginChallengeEmail = shallowRef('')
  let loginChallengeRemember = false
  const ownershipCode = shallowRef('')
  const ownershipChallengeId = shallowRef('')
  const ownershipChallengeEmail = shallowRef('')
  let ownershipPassword = ''
  let ownershipRemember = false
  const passwordVerificationGrant = shallowRef('')
  const passwordVerificationGrantEmail = shallowRef('')
  const passwordVerificationGrantExpiresAt = shallowRef(0)
  const passwordVerificationGrantAttemptsRemaining = shallowRef(0)
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
  let loginAttemptController: AbortController | null = null

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
  const trimmedLoginCode = computed(() => loginCode.value.trim())
  const normalizedOwnershipCode = computed(() => ownershipCode.value.trim().toUpperCase())
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
    /^[A-HJ-NP-Z2-9]{16}$/.test(trimmedCode.value.toUpperCase()) &&
    isRegisterUsernameValid.value &&
    isRegisterPasswordStrong.value &&
    isRegisterPasswordMatched.value,
  )
  const hasActiveOwnershipGrant = computed(() =>
    passwordVerificationGrant.value.length > 0 &&
    passwordVerificationGrantEmail.value === normalizedEmail.value &&
    passwordVerificationGrantExpiresAt.value > Date.now() &&
    passwordVerificationGrantAttemptsRemaining.value > 0,
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
    if (loginChallengeEmail.value && value !== loginChallengeEmail.value) {
      resetLoginVerification()
    }
    if (
      (ownershipChallengeEmail.value && value !== ownershipChallengeEmail.value) ||
      (passwordVerificationGrantEmail.value && value !== passwordVerificationGrantEmail.value)
    ) {
      abortLoginAttempt()
      clearOwnershipChallenge()
      clearOwnershipGrant()
      loginStage.value = 'credentials'
    }
  })
  watch(mode, (value) => {
    if (value !== 'register') return
    abortLoginAttempt()
    loginPassword.value = ''
    clearLoginChallenge()
    clearOwnershipChallenge()
    clearOwnershipGrant()
    loginStage.value = 'credentials'
    statusMessage.value = ''
    errorMessage.value = ''
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

  function startLoginAttempt() {
    loginAttemptController?.abort()
    const controller = new AbortController()
    loginAttemptController = controller
    isSubmitting.value = true
    return controller
  }

  function abortLoginAttempt() {
    loginAttemptController?.abort()
    loginAttemptController = null
    isSubmitting.value = false
  }

  function isCurrentLoginAttempt(controller: AbortController) {
    return loginAttemptController === controller && !controller.signal.aborted
  }

  function finishLoginAttempt(controller: AbortController) {
    if (loginAttemptController !== controller) return
    loginAttemptController = null
    isSubmitting.value = false
  }

  function isAbortError(error: unknown) {
    return error instanceof DOMException && error.name === 'AbortError'
  }

  function clearOwnershipChallenge() {
    ownershipCode.value = ''
    ownershipChallengeId.value = ''
    ownershipChallengeEmail.value = ''
    ownershipPassword = ''
    ownershipRemember = false
  }

  function clearLoginChallenge() {
    loginCode.value = ''
    loginChallengeId.value = ''
    loginChallengeEmail.value = ''
    loginChallengeRemember = false
  }

  function clearOwnershipGrant() {
    passwordVerificationGrant.value = ''
    passwordVerificationGrantEmail.value = ''
    passwordVerificationGrantExpiresAt.value = 0
    passwordVerificationGrantAttemptsRemaining.value = 0
  }

  function usableOwnershipGrant(emailValue: string) {
    if (
      passwordVerificationGrantEmail.value !== emailValue ||
      passwordVerificationGrantExpiresAt.value <= Date.now() ||
      passwordVerificationGrantAttemptsRemaining.value <= 0
    ) {
      clearOwnershipGrant()
      return undefined
    }
    return passwordVerificationGrant.value || undefined
  }

  function enterOwnershipChallenge(
    error: unknown,
    emailValue: string,
    password = loginPassword.value,
    remember = rememberMe.value,
  ) {
    if (
      !(error instanceof ApiError) ||
      error.code !== 'LOGIN_EMAIL_OWNERSHIP_REQUIRED' ||
      !error.ownershipChallenge
    ) return false

    clearLoginChallenge()
    clearOwnershipGrant()
    ownershipChallengeId.value = error.ownershipChallenge.challenge_id
    ownershipChallengeEmail.value = emailValue
    ownershipPassword = password
    ownershipRemember = remember
    ownershipCode.value = ''
    loginStage.value = 'ownership'
    setStatus('16 位 Email 擁有權安全碼已寄出，請輸入最新一封信中的安全碼。')
    return true
  }

  function completePasswordLogin(
    challenge: Awaited<ReturnType<typeof loginAccount>>,
    emailValue: string,
    remember: boolean,
    controller: AbortController,
  ) {
    if (!isCurrentLoginAttempt(controller)) return
    loginChallengeId.value = challenge.challenge_id
    loginChallengeEmail.value = emailValue
    loginCode.value = ''
    loginChallengeRemember = remember
    loginPassword.value = ''
    clearOwnershipChallenge()
    clearOwnershipGrant()
    loginStage.value = 'verification'
    setStatus('登入驗證碼已寄出，請在 5 分鐘內輸入。')
  }

  function ownershipErrorMessage(error: unknown) {
    if (error instanceof ApiError && (error.status === 400 || error.status === 401)) {
      return 'Email 擁有權安全碼不正確、已過期，或不是最新一封。'
    }
    if (error instanceof ApiError && error.status === 429) {
      return '安全碼嘗試次數過多，請使用最新一封安全碼或稍後再試。'
    }
    return apiErrorMessage(error, 'Email 擁有權驗證失敗，請稍後再試。')
  }

  async function submitPasswordLogin(
    loginEmail: string,
    password: string,
    remember: boolean,
    grant: string | undefined,
    controller: AbortController,
  ) {
    try {
      const challenge = await loginAccount(loginEmail, password, remember, {
        ...(grant ? { passwordVerificationGrant: grant } : {}),
        signal: controller.signal,
      })
      completePasswordLogin(challenge, loginEmail.toLowerCase(), remember, controller)
    } catch (error) {
      if (!isCurrentLoginAttempt(controller) || isAbortError(error)) return
      if (enterOwnershipChallenge(error, loginEmail.toLowerCase(), password, remember)) return

      if (grant && error instanceof ApiError && error.status === 401) {
        passwordVerificationGrantAttemptsRemaining.value -= 1
        if (passwordVerificationGrantAttemptsRemaining.value <= 0) clearOwnershipGrant()
      }
      clearOwnershipChallenge()
      loginStage.value = 'credentials'
      setFormError(loginErrorMessage(error))
    }
  }

  async function handleLogin() {
    if (!trimmedEmail.value) return setFormError('請輸入電子信箱')
    if (!loginPassword.value) return setFormError('請輸入密碼')

    const loginEmail = trimmedEmail.value
    const normalizedLoginEmail = loginEmail.toLowerCase()
    const password = loginPassword.value
    const remember = rememberMe.value
    const grant = usableOwnershipGrant(normalizedLoginEmail)
    clearLoginChallenge()
    const controller = startLoginAttempt()
    try {
      await submitPasswordLogin(loginEmail, password, remember, grant, controller)
    } finally {
      finishLoginAttempt(controller)
    }
  }

  async function handleVerifyLoginOwnership() {
    if (!ownershipChallengeId.value || ownershipChallengeEmail.value !== normalizedEmail.value) {
      clearOwnershipChallenge()
      clearOwnershipGrant()
      loginStage.value = 'credentials'
      return setFormError('Email 擁有權驗證已失效，請重新輸入電子信箱與密碼。')
    }
    if (!/^[A-HJ-NP-Z2-9]{16}$/.test(normalizedOwnershipCode.value)) {
      return setFormError('請輸入 16 位 Base32 Email 擁有權安全碼。')
    }
    if (!ownershipPassword) {
      clearOwnershipChallenge()
      clearOwnershipGrant()
      loginStage.value = 'credentials'
      return setFormError('密碼已清除，請重新輸入帳密。')
    }

    const loginEmail = ownershipChallengeEmail.value
    const challengeId = ownershipChallengeId.value
    const submittedCode = normalizedOwnershipCode.value
    const password = ownershipPassword
    const remember = ownershipRemember
    const controller = startLoginAttempt()
    try {
      const response = await verifyLoginOwnership({
        email: loginEmail,
        challenge_id: challengeId,
        code: submittedCode,
      }, controller.signal)
      if (!isCurrentLoginAttempt(controller)) return

      passwordVerificationGrant.value = response.password_verification_grant
      passwordVerificationGrantEmail.value = loginEmail
      passwordVerificationGrantExpiresAt.value = Date.now() + response.expires_in_seconds * 1000
      passwordVerificationGrantAttemptsRemaining.value = response.max_attempts
      clearOwnershipChallenge()
      setStatus('Email 擁有權已確認，正在重新驗證密碼…')
      await submitPasswordLogin(loginEmail, password, remember, response.password_verification_grant, controller)
    } catch (error) {
      if (!isCurrentLoginAttempt(controller) || isAbortError(error)) return
      if (!enterOwnershipChallenge(error, loginEmail, password, remember)) {
        loginStage.value = 'ownership'
        setFormError(ownershipErrorMessage(error))
      }
    } finally {
      finishLoginAttempt(controller)
    }
  }

  function cancelLoginAttempt() {
    abortLoginAttempt()
    clearLoginChallenge()
    clearOwnershipChallenge()
    clearOwnershipGrant()
    loginStage.value = 'credentials'
    setFormError('已取消 Email 擁有權驗證')
  }

  async function handleVerifyLogin() {
    if (!loginChallengeId.value || loginChallengeEmail.value !== normalizedEmail.value) {
      return setFormError('登入驗證已失效，請重新輸入電子信箱與密碼。')
    }
    if (!/^\d{6}$/.test(trimmedLoginCode.value)) {
      return setFormError('請輸入 6 位數登入驗證碼。')
    }

    const loginEmail = loginChallengeEmail.value
    const submittedCode = trimmedLoginCode.value
    const challengeId = loginChallengeId.value
    const remember = loginChallengeRemember
    const controller = startLoginAttempt()
    try {
      const { user } = await verifyLoginAccount({
        email: loginEmail,
        code: submittedCode,
        challenge_id: challengeId,
        remember,
      }, controller.signal)
      if (!isCurrentLoginAttempt(controller)) return

      clearLoginChallenge()
      setCurrentSession(user)
      await router.replace(redirectAfterLogin())
    } catch (error) {
      if (!isCurrentLoginAttempt(controller) || isAbortError(error)) return
      setFormError(loginVerificationErrorMessage(error))
    } finally {
      finishLoginAttempt(controller)
    }
  }

  function handleLoginSubmit() {
    if (loginStage.value === 'ownership') return handleVerifyLoginOwnership()
    if (loginStage.value === 'verification') return handleVerifyLogin()
    return handleLogin()
  }

  function resetLoginVerification() {
    abortLoginAttempt()
    loginStage.value = 'credentials'
    clearLoginChallenge()
    clearOwnershipChallenge()
    clearOwnershipGrant()
    statusMessage.value = ''
    errorMessage.value = ''
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
        code: trimmedCode.value.toUpperCase(),
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

  onBeforeUnmount(() => {
    abortLoginAttempt()
    loginPassword.value = ''
    clearLoginChallenge()
    clearOwnershipChallenge()
    clearOwnershipGrant()
  })

  return {
    mode, email, loginPassword, loginStage, loginCode, loginChallengeId,
    ownershipCode, ownershipChallengeId, ownershipChallengeEmail, hasActiveOwnershipGrant,
    registerPassword, username, code, confirmPassword,
    isSubmitting, statusMessage, errorMessage, showLoginPassword, showRegisterPassword,
    showConfirmPassword, rememberMe, isCheckingSession, authBackgroundStyle,
    registerBackgroundStyle, showRegisterEmailFormatError, showRegisterUsernameError,
    canSendVerificationCode, hasLoginPassword, loginPasswordType, loginPasswordToggleLabel,
    hasRegisterPassword, registerPasswordType, registerPasswordToggleLabel,
    hasConfirmPassword, confirmPasswordType, confirmPasswordToggleLabel,
    showRegisterPasswordMismatch, showRegisterPasswordStrengthError, registerPasswordErrorMessage, canRegister,
    handleLogin, handleLoginSubmit, handleVerifyLoginOwnership, cancelLoginAttempt,
    handleVerifyLogin, resetLoginVerification, handleSendCode, handleRegister,
  }
}
