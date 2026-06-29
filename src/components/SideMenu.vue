<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { publicAsset } from '../utils/assets'

const props = defineProps<{
  compact?: boolean
}>()

const route = useRoute()
const router = useRouter()

const items = [
  { label: '首頁', icon: 'home', iconSrc: publicAsset('icons/home.png'), to: '/home', activePaths: ['/home'] },
  { label: '聊天', icon: 'chat', iconSrc: publicAsset('icons/chat.png'), to: '/social', activePaths: [] },
  { label: '搜尋', icon: 'search', iconSrc: publicAsset('icons/search.png'), to: '/explore', activePaths: [] },
  { label: '探索', icon: 'compass', iconSrc: publicAsset('icons/explore.png'), to: '/explore', activePaths: ['/explore'] },
  { label: '通知', icon: 'notifications', iconSrc: '', to: '/freq', activePaths: ['/freq'] },
  { label: '個人資料', icon: 'profile', iconSrc: '', to: '/personal', activePaths: ['/personal'] },
]

const activePath = computed(() => route.path)
const visibleItems = computed(() => (props.compact ? items.slice(0, 4) : items))

function isActive(paths: string[]) {
  return paths.includes(activePath.value)
}
</script>

<template>
  <nav class="side-menu" aria-label="主選單">
    <button
      v-for="item in visibleItems"
      :key="item.label"
      type="button"
      class="side-menu-item"
      :class="{ 'side-menu-item-active': isActive(item.activePaths) }"
      :aria-label="compact ? item.label : undefined"
      :title="compact ? item.label : undefined"
      @click="router.push(item.to)"
    >
      <span class="side-menu-icon" aria-hidden="true">
        <img
          v-if="compact && item.iconSrc"
          :src="item.iconSrc"
          :alt="item.label"
          class="side-menu-image-icon"
        >

        <svg
          v-else-if="item.icon === 'home'"
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

        <span v-else-if="item.icon === 'compass'" class="material-symbols-outlined">
          explore
        </span>

        <span v-else-if="item.icon === 'notifications'" class="material-symbols-outlined">
          notifications
        </span>

        <span v-else class="side-menu-avatar"></span>
      </span>
      <span v-if="!compact" class="side-menu-label">{{ item.label }}</span>
    </button>
  </nav>
</template>
