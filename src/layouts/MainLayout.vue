<script setup lang="ts">
import AppNavbar from '../components/AppNavbar.vue'

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
    <div class="fab"></div>
  </div>
</template>
