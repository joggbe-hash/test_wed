<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import MainLayout from '../layouts/MainLayout.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import { fetchExploreData } from '../api/timedApi'
import type { ExploreCategory, ExploreRow } from '../api/timedApi'

const route = useRoute()
const cleanupHandlers: Array<() => void> = []
const isLoading = shallowRef(true)
const loadErrorMessage = shallowRef('')
const searchText = shallowRef('')
const categories = ref<ExploreCategory[]>([])
const rows = ref<ExploreRow[]>([])
const normalizedSearchText = computed(() => searchText.value.trim().toLowerCase())

const filteredCategories = computed(() => {
  const query = normalizedSearchText.value
  if (!query) return categories.value

  return categories.value.filter((category) => category.name.toLowerCase().includes(query))
})

function matchesExploreCard(card: ExploreRow['cards'][number], query: string) {
  return [
    card.title,
    card.tags,
    card.description,
    card.members,
  ].some((value) => value.toLowerCase().includes(query))
}

const filteredRows = computed(() => {
  const query = normalizedSearchText.value
  if (!query) return rows.value

  return rows.value
    .map((row) => ({
      ...row,
      cards: row.cards.filter((card) => matchesExploreCard(card, query)),
    }))
    .filter((row) => row.cards.length > 0)
})

function queryToText(query: unknown) {
  return typeof query === 'string' ? query : ''
}

function bindHorizontalScrollers() {
  cleanupHandlers.splice(0).forEach((cleanup) => cleanup())

  document.querySelectorAll('.horizontal-scroller').forEach((el) => {
    const slider = el as HTMLElement
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
      if (!isDown) return
      isDown = false
      slider.style.cursor = 'grab'
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
  searchText.value = queryToText(route.query.q)
  isLoading.value = true
  loadErrorMessage.value = ''
  try {
    const response = await fetchExploreData()
    categories.value = response.data.categories
    rows.value = response.data.rows
    await nextTick()
    bindHorizontalScrollers()
  } catch (error) {
    categories.value = []
    rows.value = []
    loadErrorMessage.value = error instanceof Error ? error.message : '探索資料載入失敗'
  } finally {
    isLoading.value = false
  }
})

onBeforeUnmount(() => {
  cleanupHandlers.splice(0).forEach((cleanup) => cleanup())
})

watch(
  () => route.query.q,
  (query) => {
    searchText.value = queryToText(query)
  },
)

watch(filteredRows, async () => {
  await nextTick()
  bindHorizontalScrollers()
})
</script>

<template>
  <MainLayout active-nav="explore" sidebar-class="explore-sidebar" feed-class="explore-feed">
    <template #sidebar>
      <div class="sidebar-search">
        <svg class="sidebar-search-icon" viewBox="0 0 24 24" fill="none" stroke="var(--color-muted)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
        <input
          v-model="searchText"
          type="search"
          name="explore-search"
          class="sidebar-search-input"
          aria-label="搜尋探索內容"
          autocomplete="off"
        >
        <button
          v-if="searchText"
          type="button"
          class="sidebar-search-clear"
          aria-label="清除搜尋"
          @click="searchText = ''"
        >
          <span aria-hidden="true">×</span>
        </button>
      </div>
      <div class="grid-icons gap-x-[15px]">
        <div v-for="item in filteredCategories" :key="item.id" class="grid-item">
          <div class="grid-icon-circle"></div>
          <div class="mt-[5px] text-sm text-text-default">{{ item.name }}</div>
        </div>
      </div>
    </template>

    <LoadingPanel v-if="isLoading" />

    <div v-else-if="loadErrorMessage" class="m-6 rounded-xl border border-red-300 bg-red-50 p-5 text-red-900" role="alert">
      {{ loadErrorMessage }}
    </div>

    <template v-else>
      <div v-for="row in filteredRows" :key="row.id" class="horizontal-scroller" :class="row.id === 1 ? 'mt-0' : 'mt-5'">
        <div v-for="card in row.cards" :key="card.id" class="explore-card">
          <div class="explore-header">
            <RouterLink to="/personal" class="explore-avatar" :aria-label="`查看 ${card.title} 的個人頁`" />
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
            <RouterLink to="/social" class="post-action-btn-large">進入</RouterLink>
          </div>
        </div>
      </div>
    </template>
  </MainLayout>
</template>
