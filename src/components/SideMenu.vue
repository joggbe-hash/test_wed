<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const items = [
  { label: '首頁', icon: 'home', to: '/home', activePaths: ['/home'] },
  { label: '訊息', icon: 'chat', to: '/social', activePaths: [] },
  { label: '搜尋', icon: 'search', to: '/explore', activePaths: [] },
  { label: '探索', icon: 'compass', to: '/explore', activePaths: ['/explore'] },
  { label: '通知', icon: 'notifications', to: '/freq', activePaths: ['/freq'] },
  { label: '個人檔案', icon: 'profile', to: '/personal', activePaths: ['/personal'] },
]

const activePath = computed(() => route.path)

function isActive(paths: string[]) {
  return paths.includes(activePath.value)
}
</script>

<template>
  <nav class="side-menu" aria-label="主要功能">
    <button
      v-for="item in items"
      :key="item.label"
      type="button"
      class="side-menu-item"
      :class="{ 'side-menu-item-active': isActive(item.activePaths) }"
      @click="router.push(item.to)"
    >
      <span class="side-menu-icon" aria-hidden="true">
        <svg
          v-if="item.icon === 'home'"
          viewBox="0 0 24 24"
          :fill="isActive(item.activePaths) ? 'currentColor' : 'none'"
          stroke="currentColor"
          stroke-width="2.2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M3 10.8 12 3l9 7.8V21a1 1 0 0 1-1 1h-5.2v-6.5H9.2V22H4a1 1 0 0 1-1-1z" />
        </svg>

        <span v-else-if="item.icon === 'chat'" class="material-symbols-outlined">
          chat
        </span>

        <svg
          v-else-if="item.icon === 'search'"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.1"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <circle cx="11" cy="11" r="7.5" />
          <path d="m16.5 16.5 4 4" />
        </svg>

        <svg
          v-else-if="item.icon === 'compass'"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.1"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <circle cx="12" cy="12" r="9" />
          <path d="m15.4 8.6-2.1 5.1-4.7 1.7 2.1-5.1z" />
        </svg>

        <span v-else-if="item.icon === 'notifications'" class="material-symbols-outlined">
          notifications
        </span>

        <span v-else class="side-menu-avatar"></span>
      </span>
      <span class="side-menu-label">{{ item.label }}</span>
    </button>
  </nav>
</template>
