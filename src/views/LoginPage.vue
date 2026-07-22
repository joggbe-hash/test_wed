<script setup lang="ts">
import { computed, onBeforeMount, ref, shallowRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ApiError,
  loginAccount,
  registerAccount,
  sendVerificationCode,
} from '../api/backendApi'
import { useSession } from '../composables/useSession'
import { publicAsset } from '../utils/assets'

const router = useRouter()
const route = useRoute()
const {
  currentUser,
  isSessionInitialized,
  restoreCurrentSession,
  setCurrentSession,
} = useSession()
const mode = ref<'login' | 'register'>('login')
const email = ref('')
const loginPassword = ref('')
const registerPassword = ref('')
const username = ref('')
const code = ref('')
const confirmPassword = ref('')
const isSubmitting = ref(false)
const statusMessage = ref('')
const errorMessage = ref('')
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
const registerPasswordByteLength = computed(() => new TextEncoder().encode(registerPassword.value).length)
const isRegisterPasswordStrong = computed(() =>
  registerPasswordByteLength.value >= 8 &&
  registerPasswordByteLength.value <= 72 &&
  /\p{L}/u.test(registerPassword.value) &&
  /\p{N}/u.test(registerPassword.value),
)
const showRegisterPasswordStrengthError = computed(() => hasRegisterPassword.value && !isRegisterPasswordStrong.value)
const canRegister = computed(() =>
  isRegisterEmailValid.value &&
  /^\d{6}$/.test(trimmedCode.value) &&
  isRegisterUsernameValid.value &&
  isRegisterPasswordStrong.value &&
  isRegisterPasswordMatched.value,
)

watch(loginPassword, (value) => {
  if (value.length === 0) {
    showLoginPassword.value = false
  }
})

watch(registerPassword, (value) => {
  if (value.length === 0) {
    showRegisterPassword.value = false
  }
})

watch(confirmPassword, (value) => {
  if (value.length === 0) {
    showConfirmPassword.value = false
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
  statusMessage.value = ''
  errorMessage.value = error instanceof Error ? error.message : '操作失敗'
}

function setSessionCheckError(error: unknown) {
  if (
    route.query.sessionError === 'unavailable' ||
    (error instanceof ApiError && (error.status === 429 || error.status === 503))
  ) {
    setFormError('登入服務暫時無法使用，請稍後再試')
    return
  }

  setError(error)
}

function redirectAfterLogin() {
  const redirect = route.query.redirect
  return typeof redirect === 'string' && redirect.startsWith('/') && !redirect.startsWith('//')
    ? redirect
    : '/home'
}

async function handleLogin() {
  isSubmitting.value = true
  try {
    const { user } = await loginAccount(trimmedEmail.value, loginPassword.value, rememberMe.value)
    setCurrentSession(user)
    await router.replace(redirectAfterLogin())
  } catch (error) {
    setError(error)
  } finally {
    isSubmitting.value = false
  }
}

async function handleSendCode() {
  if (!trimmedEmail.value) {
    setFormError('請先輸入電子信箱')
    return
  }

  if (!isRegisterEmailValid.value) {
    setFormError('電子信箱格式不正確')
    return
  }

  isSubmitting.value = true
  try {
    const response = await sendVerificationCode(trimmedEmail.value)
    setStatus(response.message)
  } catch (error) {
    setError(error)
  } finally {
    isSubmitting.value = false
  }
}

async function handleRegister() {
  if (!isRegisterEmailValid.value) {
    setFormError('請輸入正確的電子信箱')
    return
  }

  if (!trimmedCode.value) {
    setFormError('請輸入驗證碼')
    return
  }

  if (!isRegisterUsernameValid.value) {
    setFormError('使用者名稱需為 2-20 個字，且不能有空白')
    return
  }

  if (!isRegisterPasswordStrong.value) {
    setFormError('密碼需為 8-72 bytes，且至少包含一個字母與一個數字')
    return
  }

  if (!isRegisterPasswordMatched.value) {
    setFormError('兩次密碼不一致')
    return
  }

  isSubmitting.value = true
  try {
    await registerAccount({
      username: trimmedUsername.value,
      email: trimmedEmail.value,
      password: registerPassword.value,
      code: trimmedCode.value,
    })
    setStatus('註冊成功，請登入')
    mode.value = 'login'
  } catch (error) {
    setError(error)
  } finally {
    isSubmitting.value = false
  }
}

onBeforeMount(async () => {
  try {
    const shouldForceRefresh = isSessionInitialized.value && currentUser.value !== null
    const user = await restoreCurrentSession({ force: shouldForceRefresh })
    if (user) {
      await router.replace(redirectAfterLogin())
    }
  } catch (error) {
    setSessionCheckError(error)
  } finally {
    isCheckingSession.value = false
  }
})
</script>

<template>
  <div class="auth-container" :style="authBackgroundStyle" :aria-busy="isCheckingSession">
    <p v-if="isCheckingSession" class="text-lg font-bold text-white" role="status" aria-live="polite">
      正在確認登入狀態…
    </p>
    <div
      v-else
      id="slider"
      class="form-slider"
      :class="mode === 'register' ? '-translate-x-1/2' : 'translate-x-0'"
    >
      <form
        id="loginForm"
        class="form-section"
        :inert="mode !== 'login'"
        :aria-hidden="mode !== 'login'"
        @submit.prevent="handleLogin"
      >
        <div class="input-group">
          <label class="input-label" for="login-email">信箱</label>
          <input
            id="login-email"
            v-model="email"
            name="email"
            type="email"
            class="input-field"
            placeholder="example@email.com"
            autocomplete="email"
            spellcheck="false"
          >
        </div>

        <div class="input-group">
          <label class="input-label" for="login-password">密碼</label>
          <input
            id="login-password"
            v-model="loginPassword"
            :type="loginPasswordType"
            class="input-field"
            placeholder="請輸入密碼"
            autocomplete="current-password"
            name="password"
          >
          <button
            v-if="hasLoginPassword"
            type="button"
            class="password-toggle"
            :aria-label="loginPasswordToggleLabel"
            :aria-pressed="showLoginPassword"
            @mousedown.prevent
            @click="showLoginPassword = !showLoginPassword"
          >
            <svg v-if="!showLoginPassword" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path d="M4 4L20 20"></path>
              <path d="M9.9 5.9C10.6 5.8 11.3 5.7 12 5.7C16 5.7 19.2 7.8 21.5 12C20.8 13.3 20 14.4 19 15.3"></path>
              <path d="M14.1 18.1C13.4 18.2 12.7 18.3 12 18.3C8 18.3 4.8 16.2 2.5 12C3.2 10.7 4 9.6 5 8.7"></path>
              <path d="M10.5 9.4C9.6 10 9 10.9 9 12C9 13.7 10.3 15 12 15C13.1 15 14 14.4 14.6 13.5"></path>
            </svg>
            <svg v-else viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path d="M2.5 12C4.8 7.8 8 5.7 12 5.7S19.2 7.8 21.5 12C19.2 16.2 16 18.3 12 18.3S4.8 16.2 2.5 12Z"></path>
              <circle cx="12" cy="12" r="3.1"></circle>
            </svg>
          </button>
        </div>

        <div class="login-options">
          <label class="remember-me">
            <input v-model="rememberMe" type="checkbox" autocomplete="off">
            <span>記住我</span>
          </label>

          <div class="forgot-password">
            <RouterLink to="/forgot-password">忘記密碼</RouterLink>
          </div>
        </div>

        <div
          v-if="statusMessage || errorMessage"
          class="mb-5 w-full max-w-[550px] text-right text-sm font-bold text-white"
          :role="errorMessage ? 'alert' : 'status'"
          :aria-live="errorMessage ? 'assertive' : 'polite'"
        >
          {{ errorMessage || statusMessage }}
        </div>

        <div class="login-actions">
          <button type="button" class="btn btn-outline" @click="mode = 'register'">註冊</button>
          <button type="submit" class="btn btn-primary" :disabled="isSubmitting">
            {{ isSubmitting ? '登入中' : '登入' }}
          </button>
        </div>
      </form>

      <form
        id="registerForm"
        class="form-section register-form-section"
        :style="registerBackgroundStyle"
        :inert="mode !== 'register'"
        :aria-hidden="mode !== 'register'"
        @submit.prevent="handleRegister"
      >
        <div class="reg-group">
          <label class="reg-label" for="register-email">電子信箱</label>
          <input
            id="register-email"
            v-model="email"
            name="register-email"
            type="email"
            class="reg-input"
            placeholder="example@email.com"
            autocomplete="email"
            spellcheck="false"
            :aria-invalid="showRegisterEmailFormatError"
            :aria-describedby="showRegisterEmailFormatError ? 'register-email-feedback' : statusMessage || errorMessage ? 'register-feedback' : undefined"
          >
          <button
            type="button"
            class="ml-3 shrink-0 rounded-full bg-accent-earth px-5 py-3 text-sm font-semibold text-white transition-all duration-300 hover:bg-nav disabled:cursor-not-allowed disabled:opacity-60 max-md:ml-0 max-md:mt-2 max-md:w-full"
            :disabled="!canSendVerificationCode"
            @click="handleSendCode"
          >
            取得驗證碼
          </button>
        </div>
        <p
          v-if="showRegisterEmailFormatError"
          id="register-email-feedback"
          class="mb-4 w-full max-w-[650px] text-right text-sm font-bold text-red-600"
        >
          電子信箱格式不正確
        </p>
        <div class="reg-group">
          <label class="reg-label" for="register-code">驗證碼</label>
          <input
            id="register-code"
            v-model="code"
            name="verification-code"
            type="text"
            class="reg-input"
            placeholder="請輸入 6 位驗證碼"
            autocomplete="one-time-code"
            inputmode="numeric"
            pattern="[0-9]{6}"
            maxlength="6"
            spellcheck="false"
          >
        </div>
        <div class="reg-group">
          <label class="reg-label" for="register-username">使用者名稱</label>
          <input
            id="register-username"
            v-model="username"
            name="username"
            type="text"
            class="reg-input"
            placeholder="請輸入暱稱"
            autocomplete="username"
            spellcheck="false"
            :aria-invalid="showRegisterUsernameError"
            :aria-describedby="showRegisterUsernameError ? 'register-username-feedback' : undefined"
          >
        </div>
        <p
          v-if="showRegisterUsernameError"
          id="register-username-feedback"
          class="mb-4 w-full max-w-[650px] text-right text-sm font-bold text-red-600"
        >
          使用者名稱需為 2-20 個字，且不能有空白
        </p>
        <div class="reg-group">
          <label class="reg-label" for="register-password">密碼</label>
          <input
            id="register-password"
            v-model="registerPassword"
            :type="registerPasswordType"
            class="reg-input"
            placeholder="至少 8 個字元"
            autocomplete="new-password"
            name="new-password"
            :aria-invalid="showRegisterPasswordStrengthError"
            :aria-describedby="showRegisterPasswordStrengthError ? 'register-password-strength-feedback' : undefined"
          >
          <button
            v-if="hasRegisterPassword"
            type="button"
            class="password-toggle"
            :aria-label="registerPasswordToggleLabel"
            :aria-pressed="showRegisterPassword"
            @mousedown.prevent
            @click="showRegisterPassword = !showRegisterPassword"
          >
            <svg v-if="!showRegisterPassword" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path d="M4 4L20 20"></path>
              <path d="M9.9 5.9C10.6 5.8 11.3 5.7 12 5.7C16 5.7 19.2 7.8 21.5 12C20.8 13.3 20 14.4 19 15.3"></path>
              <path d="M14.1 18.1C13.4 18.2 12.7 18.3 12 18.3C8 18.3 4.8 16.2 2.5 12C3.2 10.7 4 9.6 5 8.7"></path>
              <path d="M10.5 9.4C9.6 10 9 10.9 9 12C9 13.7 10.3 15 12 15C13.1 15 14 14.4 14.6 13.5"></path>
            </svg>
            <svg v-else viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path d="M2.5 12C4.8 7.8 8 5.7 12 5.7S19.2 7.8 21.5 12C19.2 16.2 16 18.3 12 18.3S4.8 16.2 2.5 12Z"></path>
              <circle cx="12" cy="12" r="3.1"></circle>
            </svg>
          </button>
        </div>
        <p
          v-if="showRegisterPasswordStrengthError"
          id="register-password-strength-feedback"
          class="mb-4 w-full max-w-[650px] text-right text-sm font-bold text-red-600"
        >
          密碼需為 8-72 bytes，且至少包含一個字母與一個數字
        </p>
        <div class="reg-group">
          <label class="reg-label" for="confirm-password">確認密碼</label>
          <input
            id="confirm-password"
            v-model="confirmPassword"
            :type="confirmPasswordType"
            class="reg-input"
            placeholder="再次輸入密碼"
            autocomplete="new-password"
            name="confirm-new-password"
            :aria-invalid="showRegisterPasswordMismatch"
            :aria-describedby="showRegisterPasswordMismatch ? 'register-password-feedback' : undefined"
          >
          <button
            v-if="hasConfirmPassword"
            type="button"
            class="password-toggle"
            :aria-label="confirmPasswordToggleLabel"
            :aria-pressed="showConfirmPassword"
            @mousedown.prevent
            @click="showConfirmPassword = !showConfirmPassword"
          >
            <svg v-if="!showConfirmPassword" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path d="M4 4L20 20"></path>
              <path d="M9.9 5.9C10.6 5.8 11.3 5.7 12 5.7C16 5.7 19.2 7.8 21.5 12C20.8 13.3 20 14.4 19 15.3"></path>
              <path d="M14.1 18.1C13.4 18.2 12.7 18.3 12 18.3C8 18.3 4.8 16.2 2.5 12C3.2 10.7 4 9.6 5 8.7"></path>
              <path d="M10.5 9.4C9.6 10 9 10.9 9 12C9 13.7 10.3 15 12 15C13.1 15 14 14.4 14.6 13.5"></path>
            </svg>
            <svg v-else viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path d="M2.5 12C4.8 7.8 8 5.7 12 5.7S19.2 7.8 21.5 12C19.2 16.2 16 18.3 12 18.3S4.8 16.2 2.5 12Z"></path>
              <circle cx="12" cy="12" r="3.1"></circle>
            </svg>
          </button>
        </div>
        <p
          v-if="showRegisterPasswordMismatch"
          id="register-password-feedback"
          class="mb-4 w-full max-w-[650px] text-right text-sm font-bold text-red-600"
        >
          兩次密碼不一致
        </p>

        <div class="flex w-full max-w-[650px] justify-center">
          <button type="submit" class="reg-btn mt-0" :disabled="isSubmitting || !canRegister">確認註冊</button>
        </div>

        <div
          v-if="statusMessage || errorMessage"
          id="register-feedback"
          class="mt-5 text-sm font-bold"
          :class="errorMessage || (statusMessage && statusMessage.includes('驗證碼已送出')) ? 'text-red-600' : 'text-brown'"
          :role="errorMessage ? 'alert' : 'status'"
          :aria-live="errorMessage ? 'assertive' : 'polite'"
        >
          {{ errorMessage || statusMessage }}
        </div>

        <div class="mt-[25px]">
          <a href="#" class="font-medium text-nav no-underline" @click.prevent="mode = 'login'">
            已有帳號，回到登入
          </a>
        </div>
      </form>
    </div>
  </div>
</template>
