<script setup lang="ts">
defineProps<{
  isLoggingOut: boolean
  errorMessage: string
}>()

const emit = defineEmits<{
  logout: []
}>()
</script>

<template>
  <section id="account-session" class="account-session-panel" aria-labelledby="account-session-title">
    <div class="account-session-panel__copy">
      <span class="account-session-panel__eyebrow">SESSION</span>
      <h2 id="account-session-title" class="account-session-panel__title text-balance">
        帳號與工作階段
      </h2>
      <p class="account-session-panel__description text-pretty">
        在共用裝置完成使用後，請登出目前帳號以保護個人資料。
      </p>
    </div>

    <div class="account-session-panel__action">
      <p v-if="errorMessage" class="account-session-panel__error text-pretty" role="alert">
        {{ errorMessage }}
      </p>
      <button
        type="button"
        class="account-session-panel__logout"
        :disabled="isLoggingOut"
        :aria-busy="isLoggingOut"
        @click="emit('logout')"
      >
        <span>{{ isLoggingOut ? '登出中...' : '登出目前帳號' }}</span>
        <span aria-hidden="true">→</span>
      </button>
    </div>
  </section>
</template>

<style scoped>
@reference "../../style.css";

.account-session-panel {
  @apply flex items-center justify-between gap-10 rounded-2xl border border-[#e3c7c7] bg-card p-7 shadow-sm max-md:flex-col max-md:items-stretch max-md:gap-6 max-md:p-5;
}

.account-session-panel__copy {
  @apply max-w-2xl;
}

.account-session-panel__eyebrow {
  @apply text-xs font-bold text-[#a33c3c];
}

.account-session-panel__title {
  @apply mt-2 text-xl font-bold text-ink-warm;
}

.account-session-panel__description {
  @apply mt-2 text-sm leading-6 text-[#625a54];
}

.account-session-panel__action {
  @apply flex min-w-[230px] flex-col items-end gap-3 max-md:min-w-0 max-md:items-stretch;
}

.account-session-panel__error {
  @apply max-w-sm text-right text-xs leading-5 text-[#a33c3c] max-md:text-left;
}

.account-session-panel__logout {
  @apply flex min-h-11 w-full items-center justify-between gap-5 rounded-xl border border-[#cc8f8f] bg-[#fff7f7] px-5 py-3 text-sm font-bold text-danger-strong hover:border-danger-strong hover:bg-[#fff0f0] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-danger-strong disabled:cursor-not-allowed disabled:opacity-60;
  font: inherit;
}
</style>
