<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { deletePost } from '../api/backendApi'
import AccessibleImageViewer from '../components/AccessibleImageViewer.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import SidebarWidgets from '../components/SidebarWidgets.vue'
import XPostCard from '../components/XPostCard.vue'
import { openComposeModal } from '../composables/useComposeModal'
import MainLayout from '../layouts/MainLayout.vue'
import { useFeedStore } from '../stores/useFeedStore'

const router = useRouter()
const feedStore = useFeedStore()
const errorMessage = ref('')
const openPostMenuId = ref<number | null>(null)
const imageViewer = ref<{ urls: string[], index: number } | null>(null)

function openImageViewer(urls: string[], index: number) {
  imageViewer.value = { urls, index }
}

async function loadPosts() {
  try {
    await feedStore.loadPosts()
    errorMessage.value = ''
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '無法載入貼文'
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

onMounted(loadPosts)
</script>

<template>
  <MainLayout active-nav="social">
    <template #sidebar><SidebarWidgets /></template>

    <LoadingPanel v-if="feedStore.isLoading" />
    <template v-else>
      <div class="theme-banner" aria-hidden="true">
        <div class="theme-banner-header">
          <div class="flex gap-[15px]"><div class="h-5 w-20 rounded bg-white/20"></div><div class="h-5 w-[120px] rounded bg-white/20"></div></div>
          <div class="flex gap-[15px]"><div class="h-[35px] w-20 rounded-full bg-white"></div><div class="h-[35px] w-[30px] rounded bg-white/20"></div></div>
        </div>
        <div class="theme-banner-bottom">
          <div class="h-[35px] w-[40%] rounded-lg bg-white/80"></div><div class="h-[15px] w-full rounded bg-white/40"></div><div class="h-[15px] w-[60%] rounded bg-white/40"></div>
        </div>
        <div class="absolute bottom-0 right-[50px] h-[100px] w-[100px] rounded-t-full bg-white/20"></div>
      </div>

      <button type="button" class="post-card show compose-entry" @click="openComposeModal">
        <span class="compose-entry-avatar" aria-hidden="true"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 12 12 2l10 10-10 10z" /></svg></span>
        <span class="compose-entry-body"><span class="compose-entry-placeholder">有什麼新鮮事？</span><span class="compose-entry-button">發佈</span></span>
      </button>

      <div v-if="errorMessage" class="post-card show" role="alert"><div class="post-body text-sm text-[#cc3333]">{{ errorMessage }}</div></div>
      <XPostCard
        v-for="post in feedStore.posts"
        :key="`x-${post.id}`"
        :post="post"
        :is-menu-open="openPostMenuId === post.id"
        @open-profile="router.push('/personal')"
        @toggle-menu="openPostMenuId = openPostMenuId === $event ? null : $event"
        @delete-post="handleDeletePost"
        @open-image="openImageViewer"
      />
    </template>

    <AccessibleImageViewer v-if="imageViewer" v-model:index="imageViewer.index" :urls="imageViewer.urls" @close="imageViewer = null" />
  </MainLayout>
</template>
