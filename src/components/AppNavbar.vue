<script setup lang="ts">
import { useRouter } from 'vue-router'
import SideMenu from './SideMenu.vue'

const router = useRouter()
</script>

<template>
  <div class="navbar">
    <div class="nav-logo" @click="router.push('/home')"></div>
    <div class="nav-actions">
      <SideMenu class="side-menu-bar" compact />
      <button 
        type="button" 
        class="flex size-10 cursor-pointer items-center justify-center rounded-full border-0 bg-transparent text-white transition-colors hover:bg-brown-dark max-md:size-8" 
        @click="router.push('/personal')"
        title="個人資料"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="size-[26px]">
          <circle cx="12" cy="12" r="10"></circle>
          <circle cx="12" cy="9" r="3.5"></circle>
          <path d="M6.5 19.5v-1a4 4 0 0 1 4-4h3a4 4 0 0 1 4 4v1"></path>
        </svg>
      </button>
      <button type="button" class="nav-badge" @click="router.push('/freq')">999+</button>
    </div>
  </div>
</template>
