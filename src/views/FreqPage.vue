<script setup>
import { useRouter } from 'vue-router'
import AppNavbar from '../components/AppNavbar.vue'
import { usePageCss } from '../composables/usePageCss'

usePageCss('freq_page.css', { materialIcons: true })

const router = useRouter()
</script>

<template>
  <div class="app-container">
    <AppNavbar />
    <div class="main-layout">
      <div class="sidebar">
        <div class="time-card">時間</div>
        <div class="time-card">時間</div>
        <div class="time-card">時間</div>
      </div>

      <div class="feed-content">
        <div class="stat-item">使用者資料</div>
        <div class="stat-item">互動數據</div>
        <div class="stat-item">收藏紀錄</div>
        <div class="stat-item">設定項目</div>
        <div class="stat-item logout" @click="router.push('/login')">登出</div>
      </div>
    </div>
  </div>
</template>
