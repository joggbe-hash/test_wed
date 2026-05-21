<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppNavbar from '../components/AppNavbar.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import { fetchFreqData } from '../api/timedApi'

const router = useRouter()
const isLoading = ref(true)
const timeCards = ref<string[]>([])
const stats = ref<string[]>([])

onMounted(async () => {
  isLoading.value = true
  const response = await fetchFreqData()
  timeCards.value = response.data.timeCards
  stats.value = response.data.stats
  isLoading.value = false
})
</script>

<template>
  <div class="app-container">
    <AppNavbar />
    <div class="main-layout freq-layout">
      <div class="sidebar freq-sidebar">
        <div v-for="card in timeCards" :key="card" class="time-card">{{ card }}</div>
      </div>

      <div class="feed-content freq-feed">
        <LoadingPanel v-if="isLoading" />

        <template v-else>
          <div v-for="item in stats" :key="item" class="stat-item">{{ item }}</div>
          <div class="stat-item logout" @click="router.push('/login')">登出</div>
        </template>
      </div>
    </div>
  </div>
</template>
