<script setup lang="ts">
import { computed, shallowRef } from 'vue'
import { RouterLink } from 'vue-router'
import { publicAsset } from '../utils/assets'

const email = shallowRef('')
const statusMessage = shallowRef('')
const errorMessage = shallowRef('')
const isSubmitting = shallowRef(false)
const forgotPasswordBackgroundStyle = {
  '--auth-background-image': `url("${publicAsset('picture/meme_background.jpg')}")`,
}

const trimmedEmail = computed(() => email.value.trim())
const isEmailValid = computed(() => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmedEmail.value))
const showEmailError = computed(() => email.value.length > 0 && !isEmailValid.value)
const emailDescriptionId = computed(() => {
  if (showEmailError.value) {
    return 'forgot-password-email-feedback'
  }

  return statusMessage.value || errorMessage.value ? 'forgot-password-status' : undefined
})
const canSubmit = computed(() => isEmailValid.value && !isSubmitting.value)

async function handleSubmit() {
  statusMessage.value = ''
  errorMessage.value = ''

  if (!trimmedEmail.value) {
    errorMessage.value = '請先輸入電子信箱'
    return
  }

  if (!isEmailValid.value) {
    errorMessage.value = '電子信箱格式不正確'
    return
  }

  isSubmitting.value = true
  try {
    await new Promise((resolve) => window.setTimeout(resolve, 350))
    statusMessage.value = '已收到申請；後端接上重設密碼 API 後，會寄送重設連結。'
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <main class="forgot-password-page" :style="forgotPasswordBackgroundStyle">
    <form class="forgot-password-panel" @submit.prevent="handleSubmit">
      <div class="forgot-password-header">
        <h1>忘記密碼</h1>
        <RouterLink class="forgot-password-back" to="/login">返回登入</RouterLink>
      </div>

      <div class="forgot-password-field">
        <label for="forgot-password-email">電子信箱</label>
        <input
          id="forgot-password-email"
          v-model="email"
          type="email"
          placeholder="example@email.com"
          autocomplete="email"
          :aria-invalid="showEmailError"
          :aria-describedby="emailDescriptionId"
        >
      </div>

      <p
        v-if="showEmailError"
        id="forgot-password-email-feedback"
        class="forgot-password-error"
      >
        電子信箱格式不正確
      </p>

      <button type="submit" class="forgot-password-submit" :disabled="!canSubmit">
        {{ isSubmitting ? '送出中' : '送出重設連結' }}
      </button>

      <p
        v-if="statusMessage || errorMessage"
        id="forgot-password-status"
        class="forgot-password-status"
        :class="errorMessage ? 'forgot-password-status-error' : 'forgot-password-status-success'"
        aria-live="polite"
      >
        {{ errorMessage || statusMessage }}
      </p>
    </form>
  </main>
</template>
