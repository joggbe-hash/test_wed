<script setup lang="ts">
interface Props {
  email: string
  isSubmitting: boolean
  statusMessage: string
  errorMessage: string
}

defineProps<Props>()

const code = defineModel<string>({ required: true })
const emit = defineEmits<{
  restart: []
}>()
</script>

<template>
  <section class="w-full max-w-[550px]" aria-labelledby="login-verification-title">
    <h2 id="login-verification-title" class="mb-3 text-right text-xl font-bold text-white">
      Email 登入驗證
    </h2>
    <p class="mb-5 text-right text-sm font-semibold text-white">
      驗證碼已寄到 {{ email }}，請在 5 分鐘內輸入。完成驗證前不會建立登入 Session。
    </p>

    <div class="input-group">
      <label class="input-label" for="login-verification-code">登入驗證碼</label>
      <input
        id="login-verification-code"
        v-model="code"
        name="login-verification-code"
        type="text"
        class="input-field"
        placeholder="請輸入 6 位數驗證碼"
        autocomplete="one-time-code"
        inputmode="numeric"
        pattern="[0-9]{6}"
        maxlength="6"
        spellcheck="false"
        required
        autofocus
      >
    </div>

    <div
      v-if="statusMessage || errorMessage"
      class="mb-5 text-right text-sm font-bold text-white"
      :role="errorMessage ? 'alert' : 'status'"
      :aria-live="errorMessage ? 'assertive' : 'polite'"
    >
      {{ errorMessage || statusMessage }}
    </div>

    <div class="login-actions">
      <button type="button" class="btn btn-outline" :disabled="isSubmitting" @click="emit('restart')">
        重新輸入帳密
      </button>
      <button type="submit" class="btn btn-primary" :disabled="isSubmitting">
        {{ isSubmitting ? '驗證中' : '驗證並登入' }}
      </button>
    </div>
  </section>
</template>
