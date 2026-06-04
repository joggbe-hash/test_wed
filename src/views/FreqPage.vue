<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import MainLayout from '../layouts/MainLayout.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import { logoutAccount } from '../api/backendApi'
import { fetchFreqData } from '../api/timedApi'
import { showIntroduceModal } from '../composables/useModal'
import { useFeedStore } from '../stores/useFeedStore'

const router = useRouter()
const feedStore = useFeedStore()
const isLoading = ref(true)
const isLoggingOut = ref(false)
const errorMessage = ref('')
const timeCards = ref<string[]>([])
const stats = ref<string[]>([])

function resetScreenState() {
  timeCards.value = []
  stats.value = []
  errorMessage.value = ''
  showIntroduceModal.value = false
  feedStore.reset()
}

async function handleLogout() {
  if (isLoggingOut.value) {
    return
  }

  isLoggingOut.value = true
  errorMessage.value = ''

  try {
    await logoutAccount()
    resetScreenState()
    await router.push('/login')
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '登出失敗，請稍後再試'
  } finally {
    isLoggingOut.value = false
  }
}

onMounted(async () => {
  isLoading.value = true
  try {
    const response = await fetchFreqData()
    timeCards.value = response.data.timeCards
    stats.value = response.data.stats
    errorMessage.value = ''
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '載入資料失敗'
  } finally {
    isLoading.value = false
  }
})
</script>

<template>
  <MainLayout
    active-nav="freq"
    layout-class="freq-layout"
    sidebar-class="freq-sidebar"
    feed-class="freq-feed"
  >
    <template #sidebar>
      <div v-for="card in timeCards" :key="card" class="time-card">{{ card }}</div>
    </template>

    <LoadingPanel v-if="isLoading" />

    <template v-else>
      <div v-for="item in stats" :key="item" class="stat-item">{{ item }}</div>
      <div v-if="errorMessage" class="stat-item">{{ errorMessage }}</div>
      <button
        type="button"
        class="stat-item logout"
        :disabled="isLoggingOut"
        @click="handleLogout"
      >
        {{ isLoggingOut ? '登出中...' : '登出' }}
      </button>
    </template>
  </MainLayout>
</template>
