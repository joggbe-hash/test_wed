<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { deletePost } from '../api/backendApi'
import AccessibleImageViewer from '../components/AccessibleImageViewer.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import SidebarWidgets from '../components/SidebarWidgets.vue'
import XPostCard from '../components/XPostCard.vue'
import MainLayout from '../layouts/MainLayout.vue'
import { useFeedStore } from '../stores/useFeedStore'

const feedStore = useFeedStore()
const openPostMenuId = ref<number | null>(null)
const imageViewer = ref<{ urls: string[], index: number } | null>(null)

function openImageViewer(urls: string[], index: number) {
  imageViewer.value = { urls, index }
}

async function handleDeletePost(postId: number) {
  try {
    await deletePost(postId)
    feedStore.removePost(postId)
    openPostMenuId.value = null
  } catch (error) {
    console.error('刪除貼文失敗', error)
  }
}

onMounted(async () => {
  if (!feedStore.isLoaded) await feedStore.loadPosts()
})
</script>

<template>
  <MainLayout active-nav="personal">
    <template #sidebar><SidebarWidgets /></template>

    <LoadingPanel v-if="feedStore.isLoading" />
    <template v-else>
      <XPostCard
        v-for="post in feedStore.posts"
        :key="`px-${post.id}`"
        :post="post"
        :is-menu-open="openPostMenuId === post.id"
        @open-profile="() => {}"
        @toggle-menu="openPostMenuId = openPostMenuId === $event ? null : $event"
        @delete-post="handleDeletePost"
        @open-image="openImageViewer"
      />
    </template>

    <AccessibleImageViewer v-if="imageViewer" v-model:index="imageViewer.index" :urls="imageViewer.urls" @close="imageViewer = null" />
  </MainLayout>
</template>
