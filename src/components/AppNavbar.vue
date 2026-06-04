<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const now = ref(new Date())
let clockTimer: number | undefined

const currentTime = computed(() =>
  new Intl.DateTimeFormat('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: true,
  }).format(now.value),
)

onMounted(() => {
  clockTimer = window.setInterval(() => {
    now.value = new Date()
  }, 1_000)
})

onBeforeUnmount(() => {
  if (clockTimer !== undefined) {
    window.clearInterval(clockTimer)
  }
})
</script>

<template>
  <div class="navbar">
    <div class="nav-logo" @click="router.push('/home')"></div>

    <div class="nav-actions">
      <button
        type="button"
        class="top-nav-icon top-nav-profile"
        aria-label="個人"
        title="個人"
        @click="router.push('/personal')"
      >
        <span class="material-symbols-outlined" aria-hidden="true">account_circle</span>
      </button>

      <button
        type="button"
        class="top-nav-icon top-nav-search"
        aria-label="探索"
        title="探索"
        @click="router.push('/explore')"
      >
        <span class="material-symbols-outlined" aria-hidden="true">explore</span>
      </button>

      <button
        type="button"
        class="top-nav-icon top-nav-notification"
        aria-label="通知"
        title="通知"
      >
        <span class="material-symbols-outlined" aria-hidden="true">notifications</span>
      </button>

      <button
        type="button"
        class="nav-time"
        aria-label="現在時間"
        title="現在時間"
        @click="router.push('/freq')"
      >
        {{ currentTime }}
      </button>
    </div>
  </div>
</template>
