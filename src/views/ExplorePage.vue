<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppNavbar from '../components/AppNavbar.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import { fetchExploreData } from '../api/timedApi'
import type { ExploreCategory, ExploreRow } from '../api/timedApi'

const router = useRouter()
const cleanupHandlers: Array<() => void> = []
const isLoading = ref(true)
const searchText = ref('')
const categories = ref<ExploreCategory[]>([])
const rows = ref<ExploreRow[]>([])

function bindHorizontalScrollers() {
  cleanupHandlers.splice(0).forEach((cleanup) => cleanup())

  document.querySelectorAll('.horizontal-scroller').forEach((el) => { const slider = el as HTMLElement;
    let isDown = false
    let startX = 0
    let scrollLeft = 0

    const onMouseDown = (event: MouseEvent) => {
      isDown = true
      slider.style.cursor = 'grabbing'
      startX = event.pageX - slider.offsetLeft
      scrollLeft = slider.scrollLeft
      event.preventDefault()
    }

    const onMouseUp = () => {
      if (isDown) {
        isDown = false
        slider.style.cursor = 'grab'
      }
    }

    const onMouseMove = (event: MouseEvent) => {
      if (!isDown) return
      event.preventDefault()
      const x = event.pageX - slider.offsetLeft
      slider.scrollLeft = scrollLeft - (x - startX)
    }

    const onWheel = (event: WheelEvent) => {
      event.preventDefault()
      slider.scrollLeft += event.deltaY
    }

    slider.addEventListener('mousedown', onMouseDown)
    window.addEventListener('mouseup', onMouseUp)
    window.addEventListener('mousemove', onMouseMove)
    slider.addEventListener('wheel', onWheel, { passive: false })

    cleanupHandlers.push(() => {
      slider.removeEventListener('mousedown', onMouseDown)
      window.removeEventListener('mouseup', onMouseUp)
      window.removeEventListener('mousemove', onMouseMove)
      slider.removeEventListener('wheel', onWheel)
    })
  })
}

onMounted(async () => {
  isLoading.value = true
  const response = await fetchExploreData()
  categories.value = response.data.categories
  rows.value = response.data.rows
  isLoading.value = false
  await nextTick()
  bindHorizontalScrollers()
})

onBeforeUnmount(() => {
  cleanupHandlers.splice(0).forEach((cleanup) => cleanup())
})
</script>

<template>
  <div class="app-container">
    <AppNavbar />
    <div class="main-layout">
      <div class="sidebar">
        <div class="sidebar-search">
          <svg class="sidebar-search-icon" viewBox="0 0 24 24" fill="none" stroke="#7A7A7A" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
          <input v-model="searchText" type="text" class="sidebar-search-input" placeholder="">
          <button
            v-if="searchText"
            type="button"
            class="sidebar-search-clear"
            aria-label="清除搜尋"
            @click="searchText = ''"
          >
            <span class="material-symbols-outlined">cancel</span>
          </button>
        </div>
        <div class="grid-icons gap-x-[15px]">
          <div v-for="item in categories" :key="item.id" class="grid-item">
            <div class="grid-icon-circle"></div>
            <div class="mt-[5px] text-sm text-[#333333]">{{ item.name }}</div>
          </div>
        </div>
      </div>

      <div class="feed-content explore-feed">
        <LoadingPanel v-if="isLoading" />

        <template v-else>
          <div v-for="row in rows" :key="row.id" class="horizontal-scroller" :class="row.id === 1 ? 'mt-0' : 'mt-5'">
            <div v-for="card in row.cards" :key="card.id" class="explore-card">
              <div class="explore-header">
                <div class="explore-avatar" @click="router.push('/personal')"></div>
                <div class="explore-title-area">
                  <div class="explore-title">{{ card.title }}</div>
                  <div class="explore-tags">{{ card.tags }}</div>
                </div>
              </div>
              <div class="explore-details">
                <div class="explore-desc">{{ card.description }}</div>
                <div class="explore-members">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path></svg>
                  <span>{{ card.members }}</span>
                </div>
                <button class="post-action-btn-large" @click="router.push('/social')">進入</button>
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>
