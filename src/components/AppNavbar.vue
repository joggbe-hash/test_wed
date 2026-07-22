<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, shallowRef } from 'vue'
import { RouterLink } from 'vue-router'
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
    <RouterLink to="/home" class="nav-logo" aria-label="回到首頁" />

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
      <RouterLink
        to="/personal"
        class="top-nav-icon top-nav-profile"
        aria-label="個人"
        title="個人"
      >
        <span class="material-symbols-outlined" aria-hidden="true">account_circle</span>
      </RouterLink>

      <RouterLink
        to="/explore"
        class="top-nav-icon top-nav-search"
        aria-label="探索"
        title="探索"
      >
        <span class="material-symbols-outlined" aria-hidden="true">explore</span>
      </RouterLink>

      <button
        type="button"
        class="top-nav-icon top-nav-notification"
        aria-label="通知"
        title="通知"
      >
        <span class="material-symbols-outlined" aria-hidden="true">notifications</span>
      </button>

      <RouterLink
        to="/freq"
        class="nav-time"
        aria-label="現在時間"
        title="現在時間"
      >
        {{ currentTime }}
      </RouterLink>
    </div>
  </div>
</template>
