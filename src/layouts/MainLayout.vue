<script setup lang="ts">
import AppNavbar from '../components/AppNavbar.vue'
import { openComposeModal } from '../composables/useComposeModal'

defineProps<{
  activeNav?: string
  layoutClass?: string
  sidebarClass?: string
  feedClass?: string
}>()
</script>

<template>
  <div class="app-container">
    <AppNavbar :active="activeNav" />
    <div class="main-layout" :class="layoutClass">
      <div class="sidebar" :class="sidebarClass">
        <slot name="sidebar" />
      </div>

      <div class="feed-content" :class="feedClass">
        <slot />
      </div>
    </div>
    <button type="button" class="fab" aria-label="新增貼文" @click="openComposeModal"></button>
  </div>
</template>
