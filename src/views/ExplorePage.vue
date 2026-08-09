<script setup lang="ts">
import { shallowRef, watch } from 'vue'
import { useRoute } from 'vue-router'
import MainLayout from '../layouts/MainLayout.vue'

const route = useRoute()
const searchText = shallowRef('')

function queryToText(query: unknown) {
  return typeof query === 'string' ? query : ''
}

watch(
  () => route.query.q,
  (query) => {
    searchText.value = queryToText(query)
  },
  { immediate: true },
)
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
          disabled
        >
      </div>
    </template>

    <section class="m-6 rounded-2xl border border-border-soft bg-white p-8 text-center" aria-labelledby="explore-empty-title">
      <h1 id="explore-empty-title" class="text-2xl font-bold text-ink-warm">目前沒有探索資料</h1>
      <p class="mx-auto mt-3 max-w-xl text-sm leading-6 text-text-muted">
        探索功能尚未連接真實後端資料來源，因此不顯示示範內容。
      </p>
    </section>
  </MainLayout>
</template>
