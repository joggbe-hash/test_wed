<script setup lang="ts">
import { onMounted, ref } from 'vue'
import MainLayout from '../layouts/MainLayout.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import { deletePost } from '../api/backendApi'
import XPostCard from '../components/XPostCard.vue'
import SidebarWidgets from '../components/SidebarWidgets.vue'
import { useFeedStore } from '../stores/useFeedStore'

const feedStore = useFeedStore() // 引入 Pinia 狀態庫來讀取快取的貼文
const openPostMenuId = ref<number | null>(null) // 紀錄目前開啟的設定選單編號
const imageViewer = ref<{ urls: string[], index: number } | null>(null) // 控制全螢幕圖片檢視器的狀態

// 打開圖片檢視器
function openImageViewer(urls: string[], index: number) {
  imageViewer.value = { urls, index }
}

// 關閉圖片檢視器
function closeImageViewer() {
  imageViewer.value = null
}

// 切換至上一張圖片
function showPreviousViewerImage() {
  if (!imageViewer.value || imageViewer.value.urls.length <= 1) return
  imageViewer.value.index =
    imageViewer.value.index === 0 ? imageViewer.value.urls.length - 1 : imageViewer.value.index - 1
}

// 切換至下一張圖片
function showNextViewerImage() {
  if (!imageViewer.value || imageViewer.value.urls.length <= 1) return
  imageViewer.value.index = (imageViewer.value.index + 1) % imageViewer.value.urls.length
}

// 刪除貼文邏輯
async function handleDeletePost(postId: number) {
  try {
    await deletePost(postId)
    feedStore.removePost(postId) // 透過 Pinia 狀態庫移除該貼文
    openPostMenuId.value = null
  } catch (error) {
    console.error(error)
  }
}

// 元件掛載時，確認資料是否已載入，若無則自動讀取
onMounted(async () => {
  if (!feedStore.isLoaded) {
    await feedStore.loadPosts()
  }
})
</script>

<template>
  <MainLayout active-nav="personal">
    <!-- 個人頁也套用第二版左側工具欄，讓個人頁路由和首頁、社群頁的版面一致。 -->
    <template #sidebar>
      <SidebarWidgets />
    </template>

    <LoadingPanel v-if="feedStore.isLoading" />
    <template v-else>
      <XPostCard
        v-for="post in feedStore.posts"
            :key="'px-' + post.id"
            :post="post"
            :is-menu-open="openPostMenuId === post.id"
            @open-profile="() => {}"
            @toggle-menu="openPostMenuId = openPostMenuId === $event ? null : $event"
            @delete-post="handleDeletePost"
            @open-image="openImageViewer"
          />


    </template>
    
    <!-- 全螢幕圖片檢視器彈出視窗 -->
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
