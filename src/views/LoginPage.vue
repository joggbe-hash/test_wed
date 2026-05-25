<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  loginAccount,
  registerAccount,
  sendVerificationCode,
} from '../api/backendApi'

const router = useRouter()
const mode = ref<'login' | 'register'>('login')
const email = ref('')
const password = ref('')
const username = ref('')
const code = ref('')
const confirmPassword = ref('')
const isSubmitting = ref(false)
const statusMessage = ref('')
const errorMessage = ref('')

const canRegister = computed(() => password.value !== '' && password.value === confirmPassword.value)

function setStatus(message: string) {
  statusMessage.value = message
  errorMessage.value = ''
}

function setError(error: unknown) {
  statusMessage.value = ''
  errorMessage.value = error instanceof Error ? error.message : '操作失敗'
}

async function handleLogin() {
  isSubmitting.value = true
  try {
    await loginAccount(email.value, password.value)
    await router.push('/home')
  } catch (error) {
    setError(error)
  } finally {
    isSubmitting.value = false
  }
}

async function handleSendCode() {
  if (!email.value) {
    errorMessage.value = '請先輸入電子信箱'
    return
  }

  isSubmitting.value = true
  try {
    const response = await sendVerificationCode(email.value)
    setStatus(response.debug_code ? `驗證碼已送出：${response.debug_code}` : response.message)
  } catch (error) {
    setError(error)
  } finally {
    isSubmitting.value = false
  }
}

async function handleRegister() {
  if (!canRegister.value) {
    errorMessage.value = '兩次密碼不一致'
    return
  }

  isSubmitting.value = true
  try {
    await registerAccount({
      username: username.value,
      email: email.value,
      password: password.value,
      code: code.value,
    })
    setStatus('註冊成功，請登入')
    mode.value = 'login'
  } catch (error) {
    setError(error)
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <div class="auth-container">
    <div
      id="slider"
      class="form-slider"
      :class="mode === 'register' ? '-translate-x-1/2' : 'translate-x-0'"
    >
      <form id="loginForm" class="form-section" @submit.prevent="handleLogin">
        <div class="input-group">
          <div class="input-label">信箱</div>
          <input v-model="email" type="email" class="input-field" placeholder="example@email.com" autocomplete="email">
        </div>

        <div class="input-group">
          <div class="input-label">密碼</div>
          <input v-model="password" type="password" class="input-field" placeholder="請輸入密碼" autocomplete="current-password">
        </div>

        <div class="forgot-password">
          <a href="#">忘記密碼</a>
        </div>

        <div v-if="statusMessage || errorMessage" class="mb-5 w-full max-w-[550px] text-right text-sm font-bold text-white">
          {{ errorMessage || statusMessage }}
        </div>

        <div class="login-actions">
          <button type="button" class="btn btn-outline" @click="mode = 'register'">註冊</button>
          <button type="submit" class="btn btn-primary" :disabled="isSubmitting">
            {{ isSubmitting ? '登入中' : '登入' }}
          </button>
        </div>
      </form>

      <form id="registerForm" class="form-section register-form-section" @submit.prevent="handleRegister">
        <div class="reg-group">
          <div class="reg-label">電子信箱</div>
          <input v-model="email" type="email" class="reg-input" placeholder="example@email.com" autocomplete="email">
        </div>
        <div class="reg-group">
          <div class="reg-label">使用者名稱</div>
          <input v-model="username" type="text" class="reg-input" placeholder="請輸入暱稱" autocomplete="username">
        </div>
        <div class="reg-group">
          <div class="reg-label">密碼</div>
          <input v-model="password" type="password" class="reg-input" placeholder="至少 8 個字元" autocomplete="new-password">
        </div>
        <div class="reg-group">
          <div class="reg-label">確認密碼</div>
          <input v-model="confirmPassword" type="password" class="reg-input" placeholder="再次輸入密碼" autocomplete="new-password">
        </div>
        <div class="reg-group">
          <div class="reg-label">驗證碼</div>
          <input v-model="code" type="text" class="reg-input" placeholder="請輸入驗證碼">
        </div>

        <div class="flex w-full max-w-[650px] justify-center gap-5">
          <button type="button" class="reg-btn mt-0" :disabled="isSubmitting" @click="handleSendCode">取得驗證碼</button>
          <button type="submit" class="reg-btn mt-0" :disabled="isSubmitting || !canRegister">確認註冊</button>
        </div>

        <div v-if="statusMessage || errorMessage" class="mt-5 text-sm font-bold text-[#4a3320]">
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
