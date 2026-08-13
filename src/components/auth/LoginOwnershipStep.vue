<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  email: string
  isSubmitting: boolean
  statusMessage: string
  errorMessage: string
}

defineProps<Props>()

const code = defineModel<string>({ required: true })
const emit = defineEmits<{
  cancel: []
}>()

const normalizedCode = computed({
  get: () => code.value,
  set: (value: string) => {
    code.value = value.toUpperCase().replace(/[^A-HJ-NP-Z2-9]/g, '').slice(0, 16)
  },
})
</script>

<template>
  <section class="w-full max-w-[550px]" aria-labelledby="login-ownership-title">
    <h2 id="login-ownership-title" class="mb-3 text-right text-xl font-bold text-white">
      確認 Email 擁有權
    </h2>
    <p class="mb-5 text-right text-sm font-semibold text-white">
      安全碼已寄到 {{ email }}。請輸入最新一封信中的 16 位安全碼；完成前不會建立登入 Session。
    </p>

    <div class="input-group">
      <label class="input-label" for="login-ownership-code">Email 擁有權安全碼</label>
      <input
        id="login-ownership-code"
        v-model="normalizedCode"
        name="login-ownership-code"
        type="text"
        class="input-field"
        placeholder="請輸入 16 位安全碼"
        autocomplete="one-time-code"
        autocapitalize="characters"
        inputmode="text"
        pattern="[A-HJ-NP-Z2-9]{16}"
        maxlength="16"
        spellcheck="false"
        :disabled="isSubmitting"
        :aria-invalid="Boolean(errorMessage)"
        :aria-describedby="statusMessage || errorMessage ? 'login-ownership-feedback' : undefined"
        required
        autofocus
      >
    </div>

    <div
      v-if="statusMessage || errorMessage"
      id="login-ownership-feedback"
      class="mb-5 text-right text-sm font-bold text-white"
      :role="errorMessage ? 'alert' : 'status'"
      :aria-live="errorMessage ? 'assertive' : 'polite'"
    >
      {{ errorMessage || statusMessage }}
    </div>

    <div class="login-actions">
      <button
        type="button"
        class="btn btn-outline"
        data-testid="cancel-ownership"
        @click="emit('cancel')"
      >
        取消並返回帳密
      </button>
      <button type="submit" class="btn btn-primary" :disabled="isSubmitting">
        {{ isSubmitting ? '驗證中' : '驗證並繼續' }}
      </button>
    </div>
  </section>
</template>
