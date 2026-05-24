<script setup lang="ts">
import { useRouter } from 'vue-router'
import SideMenu from './SideMenu.vue'

const router = useRouter()
</script>

<template>
  <div class="navbar">
    <div class="nav-logo" @click="router.push('/home')"></div>
    <div class="nav-actions">
      <SideMenu class="side-menu-bar" />
      <button type="button" class="nav-badge" @click="router.push('/freq')">999+</button>
    </div>
  </div>
</template>
