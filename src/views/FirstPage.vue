<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiError, createPost, fetchFeed } from '../api/backendApi'
import type { BackendPost } from '../api/backendApi'
import AppNavbar from '../components/AppNavbar.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import PostActions from '../components/PostActions.vue'
import { showIntroduceModal } from '../composables/useModal'

const route = useRoute()
const router = useRouter()
const isIframe = computed(() => route.query.isIframe === 'true')
const isPrivate = ref(false)
const isLoading = ref(true)
const isSubmitting = ref(false)
const feedPosts = ref<BackendPost[]>([])
const shareText = ref('')
const apiReturnedAt = ref('')
const errorMessage = ref('')

async function loadFeed() {
  try {
    const response = await fetchFeed()
    feedPosts.value = response.posts
    apiReturnedAt.value = new Date().toLocaleTimeString('zh-TW', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
    errorMessage.value = ''
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      await router.push('/login')
      return
    }
    errorMessage.value = error instanceof Error ? error.message : '讀取貼文失敗'
  }
}

async function handleCreatePost() {
  const content = shareText.value.trim()
  if (!content || isSubmitting.value) {
    return
  }

  isSubmitting.value = true
  try {
    await createPost(content)
    shareText.value = ''
    await loadFeed()
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      await router.push('/login')
      return
    }
    errorMessage.value = error instanceof Error ? error.message : '發文失敗'
  } finally {
    isSubmitting.value = false
  }
}

onMounted(async () => {
  isLoading.value = true
  await loadFeed()
  isLoading.value = false
})
</script>

<template>
  <div class="app-container">
    <AppNavbar />
    <div class="main-layout">
      <div class="sidebar">
        <div class="grid-icons">
          <div v-for="item in 16" :key="item" class="grid-icon-circle" @click="showIntroduceModal = true"></div>
        </div>
        <div class="sidebar-card"></div>
        <div class="sidebar-card"></div>
        <div class="sidebar-card"></div>
      </div>

      <div class="feed-content">
        <LoadingPanel v-if="isLoading && !isIframe" />

        <template v-else>
          <div class="post-card show !p-4">
            <div class="flex w-full gap-4">
              <div class="flex h-12 w-12 shrink-0 cursor-pointer items-center justify-center rounded-full bg-[#4a3320] text-white transition-opacity hover:opacity-80">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12L12 2l10 10-10 10z"/></svg>
              </div>

              <div class="flex w-full flex-col pt-1">
                <textarea
                  v-model="shareText"
                  placeholder="分享你的想法"
                  class="min-h-[44px] w-full resize-none bg-transparent text-lg text-[#333] placeholder-[#a59a91] outline-none"
                  rows="1"
                ></textarea>

                <hr class="my-3 border-[#eaddcf]">

                <div class="flex items-center justify-between">
                  <div class="flex shrink-0 gap-3 text-[#a59a91]">
                    <svg class="cursor-pointer transition-colors hover:text-[#4a3320]" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg>
                    <div class="flex h-[20px] w-[26px] cursor-pointer items-center justify-center rounded border-[1.5px] border-current text-[9px] font-black transition-colors hover:text-[#4a3320]">GIF</div>
                    <svg class="cursor-pointer transition-colors hover:text-[#4a3320]" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path></svg>
                  </div>

                  <div class="flex items-center gap-4">
                    <div class="flex cursor-pointer items-center gap-1 text-[13px] font-bold text-[#a59a91] transition-colors hover:text-[#4a3320]" @click="isPrivate = !isPrivate">
                      {{ isPrivate ? '私人' : '公開' }}
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>
                    </div>
                    <button
                      class="rounded-full bg-[#4a3320] px-5 py-1.5 text-sm font-bold text-white opacity-90 shadow-sm transition-colors hover:bg-[#382618] hover:opacity-100 disabled:cursor-not-allowed disabled:opacity-50"
                      :disabled="isSubmitting || !shareText.trim()"
                      @click="handleCreatePost"
                    >
                      {{ isSubmitting ? '發佈中' : '發文' }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="errorMessage" class="post-card show">
            <div class="post-body text-sm text-[#cc3333]">{{ errorMessage }}</div>
          </div>

          <div class="post-card show">
            <div class="post-avatar" @click="router.push('/personal')"></div>
            <div class="post-body">
              <div class="text-sm text-muted">API 回傳時間：{{ apiReturnedAt || '尚未讀取' }}</div>
            </div>
          </div>

          <div v-for="post in feedPosts" :key="post.id" class="post-card show">
            <div class="post-avatar" @click="router.push('/personal')"></div>
            <div class="flex min-w-0 flex-1 flex-col">
              <div class="post-body">
                <div class="post-user-id">{{ post.username }}</div>
                <div v-if="post.content" class="text-base leading-[1.6] text-[#333333]">{{ post.content }}</div>
                <div v-if="post.image_urls?.length" class="mt-5 flex flex-col gap-3">
                  <img
                    v-for="imageUrl in post.image_urls"
                    :key="imageUrl"
                    :src="imageUrl"
                    alt=""
                    class="max-h-[420px] w-full rounded-lg object-cover"
                  >
                </div>
                <div v-if="post.image_status === 'processing'" class="mt-5 text-sm text-muted">圖片處理中</div>
              </div>
              <PostActions />
            </div>
          </div>
        </template>
      </div>
    </div>
    <div class="fab"></div>
  </div>
</template>
