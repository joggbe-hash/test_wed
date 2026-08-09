<script setup lang="ts">
import { onMounted, shallowRef } from 'vue'
import { deletePost } from '../api/backendApi'
import { apiErrorMessage } from '../api/errors'
import AccessibleImageViewer from '../components/AccessibleImageViewer.vue'
import FeedLoadMoreButton from '../components/FeedLoadMoreButton.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import SidebarWidgets from '../components/SidebarWidgets.vue'
import XPostCard from '../components/XPostCard.vue'
import { useSession } from '../composables/useSession'
import { usePersonalPosts } from '../features/posts/usePersonalPosts'
import MainLayout from '../layouts/MainLayout.vue'
import { useFeedStore } from '../stores/useFeedStore'

const feedStore = useFeedStore()
const { currentUser } = useSession()
const {
  posts: personalPostItems,
  isLoading: isLoadingPersonalPosts,
  isLoadingMore: isLoadingMorePersonalPosts,
  hasMore: hasMorePersonalPosts,
  loadPosts: loadPersonalPosts,
  loadMore: loadMorePersonalPosts,
  removePost: removePersonalPost,
} = usePersonalPosts()
const errorMessage = shallowRef('')
const openPostMenuId = shallowRef<number | null>(null)
const imageViewer = shallowRef<{ urls: string[], index: number } | null>(null)

function openImageViewer(urls: string[], index: number) {
  imageViewer.value = { urls, index }
}

async function handleDeletePost(postId: number) {
  try {
    await deletePost(postId)
    removePersonalPost(postId)
    feedStore.removePost(postId)
    openPostMenuId.value = null
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '無法刪除貼文，請稍後再試。')
  }
}

async function handleLoadMore() {
  try {
    await loadMorePersonalPosts()
    errorMessage.value = ''
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '無法載入更多貼文，請稍後再試。')
  }
}

onMounted(async () => {
  try {
    await loadPersonalPosts()
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '無法載入個人貼文，請稍後再試。')
  }
})
</script>

<template>
  <MainLayout active-nav="personal">
    <template #sidebar><SidebarWidgets /></template>

    <LoadingPanel v-if="isLoadingPersonalPosts" />
    <template v-else>
      <div v-if="errorMessage" class="post-card show" role="alert">
        <div class="post-body text-sm text-[#cc3333]">{{ errorMessage }}</div>
      </div>
      <XPostCard
        v-for="post in personalPostItems"
        :key="`px-${post.id}`"
        :post="post"
        :is-menu-open="openPostMenuId === post.id"
        :can-delete="post.user_id === currentUser?.id"
        @open-profile="() => {}"
        @toggle-menu="openPostMenuId = openPostMenuId === $event ? null : $event"
        @delete-post="handleDeletePost"
        @open-image="openImageViewer"
      />
      <FeedLoadMoreButton
        :has-more="hasMorePersonalPosts"
        :is-loading="isLoadingMorePersonalPosts"
        @load-more="handleLoadMore"
      />
    </template>

    <AccessibleImageViewer v-if="imageViewer" v-model:index="imageViewer.index" :urls="imageViewer.urls" @close="imageViewer = null" />
  </MainLayout>
</template>
