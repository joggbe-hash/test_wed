<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, shallowRef } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const now = shallowRef(new Date())
const navSearchText = shallowRef('')
let clockTimer: number | undefined

const currentTime = computed(() =>
  new Intl.DateTimeFormat('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: true,
  }).format(now.value),
)

function submitNavSearch() {
  const query = navSearchText.value.trim()
  router.push(query ? { path: '/explore', query: { q: query } } : '/explore')
}

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

    <form class="nav-search-form" role="search" @submit.prevent="submitNavSearch">
      <svg class="nav-search-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <circle cx="11" cy="11" r="7.5"></circle>
        <line x1="16.5" y1="16.5" x2="21" y2="21"></line>
      </svg>
      <input
        v-model="navSearchText"
        type="search"
        class="nav-search-input"
        aria-label="搜尋"
        autocomplete="off"
      >
    </form>

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
