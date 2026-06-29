<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { deletePost } from '../api/backendApi'
import MainLayout from '../layouts/MainLayout.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import XPostCard from '../components/XPostCard.vue'
import SidebarWidgets from '../components/SidebarWidgets.vue'
import { openComposeModal } from '../composables/useComposeModal'
import { useFeedStore } from '../stores/useFeedStore'

const router = useRouter()
const feedStore = useFeedStore()
const errorMessage = ref('')
const openPostMenuId = ref<number | null>(null)
const imageViewer = ref<{ urls: string[], index: number } | null>(null)

function openImageViewer(urls: string[], index: number) {
  imageViewer.value = { urls, index }
}

function closeImageViewer() {
  imageViewer.value = null
}

function showPreviousViewerImage() {
  if (!imageViewer.value || imageViewer.value.urls.length <= 1) return
  imageViewer.value.index =
    imageViewer.value.index === 0 ? imageViewer.value.urls.length - 1 : imageViewer.value.index - 1
}

function showNextViewerImage() {
  if (!imageViewer.value || imageViewer.value.urls.length <= 1) return
  imageViewer.value.index = (imageViewer.value.index + 1) % imageViewer.value.urls.length
}

async function loadPosts() {
  try {
    await feedStore.loadPosts()
    errorMessage.value = ''
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '讀取貼文失敗'
  }
}

async function handleDeletePost(postId: number) {
  try {
    await deletePost(postId)
    feedStore.removePost(postId)
    openPostMenuId.value = null
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '刪除貼文失敗'
  }
}

onMounted(async () => {
  await loadPosts()
})
</script>

<template>
  <MainLayout active-nav="social">
    <template #sidebar>
      <SidebarWidgets />
    </template>

    <LoadingPanel v-if="feedStore.isLoading" />
    <template v-else>
      <div class="theme-banner">
        <div class="theme-banner-header">
          <div class="flex gap-[15px]">
            <div class="h-5 w-20 rounded bg-white/20"></div>
            <div class="h-5 w-[120px] rounded bg-white/20"></div>
          </div>
          <div class="flex gap-[15px]">
            <div class="h-[35px] w-20 rounded-full bg-white"></div>
            <div class="h-[35px] w-[30px] rounded bg-white/20"></div>
          </div>
        </div>
        <div class="theme-banner-bottom">
          <div class="h-[35px] w-[40%] rounded-lg bg-white/80"></div>
          <div class="h-[15px] w-full rounded bg-white/40"></div>
          <div class="h-[15px] w-[60%] rounded bg-white/40"></div>
        </div>
        <div class="absolute bottom-0 right-[50px] h-[100px] w-[100px] rounded-t-full bg-white/20"></div>
      </div>

      <button type="button" class="post-card show compose-entry" @click="openComposeModal">
        <span class="compose-entry-avatar">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12L12 2l10 10-10 10z"/></svg>
        </span>
        <span class="compose-entry-body">
          <span class="compose-entry-placeholder">有什麼新鮮事？</span>
          <span class="compose-entry-button">發佈</span>
        </span>
      </button>

      <div v-if="errorMessage" class="post-card show">
        <div class="post-body text-sm text-[#cc3333]">{{ errorMessage }}</div>
      </div>

      <XPostCard
        v-for="post in feedStore.posts"
        :key="'x-' + post.id"
        :post="post"
        :is-menu-open="openPostMenuId === post.id"
        @open-profile="router.push('/personal')"
        @toggle-menu="openPostMenuId = openPostMenuId === $event ? null : $event"
        @delete-post="handleDeletePost"
        @open-image="openImageViewer"
      />
    </template>

    <div
      v-if="imageViewer"
      class="fixed inset-0 z-[300] flex items-center justify-center bg-black/90"
      @click="closeImageViewer"
    >
      <button
        type="button"
        class="absolute left-5 top-5 flex size-11 items-center justify-center rounded-full bg-black/60 text-3xl text-white transition-colors hover:bg-white/15"
        aria-label="關閉圖片"
        @click.stop="closeImageViewer"
      >
        ×
      </button>

      <button
        v-if="imageViewer.urls.length > 1"
        type="button"
        class="absolute left-5 top-1/2 flex size-12 -translate-y-1/2 items-center justify-center rounded-full bg-black/60 text-3xl text-white transition-colors hover:bg-white/15"
        aria-label="上一張圖片"
        @click.stop="showPreviousViewerImage"
      >
        ‹
      </button>

      <img
        :src="imageViewer.urls[imageViewer.index]"
        alt=""
        class="max-h-[82dvh] max-w-[92vw] object-contain"
        @click.stop
      >

      <button
        v-if="imageViewer.urls.length > 1"
        type="button"
        class="absolute right-5 top-1/2 flex size-12 -translate-y-1/2 items-center justify-center rounded-full bg-black/60 text-3xl text-white transition-colors hover:bg-white/15"
        aria-label="下一張圖片"
        @click.stop="showNextViewerImage"
      >
        ›
      </button>

      <div class="absolute bottom-6 left-1/2 flex -translate-x-1/2 items-center gap-16 text-white">
        <button type="button" class="text-2xl" aria-label="回覆" @click.stop>♡</button>
        <button type="button" class="text-2xl" aria-label="轉發" @click.stop>↻</button>
        <button type="button" class="text-2xl" aria-label="喜歡" @click.stop>♥</button>
        <button type="button" class="text-2xl" aria-label="分享" @click.stop>⇧</button>
      </div>
    </div>
  </MainLayout>
</template>
