<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppNavbar from '../components/AppNavbar.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import PostActions from '../components/PostActions.vue'
import { ClosedEye, OpenEye } from '../components/Icons.vue'
import { fetchHomeFeed } from '../api/timedApi'

const route = useRoute()
const router = useRouter()
const isIframe = computed(() => route.query.isIframe === 'true')
const isPrivate = ref(false)
const isLoading = ref(true)
const feedPosts = ref([])
const shareText = ref('')
const apiReturnedAt = ref('')

onMounted(async () => {
  isLoading.value = true
  const response = await fetchHomeFeed()
  feedPosts.value = response.data.posts
  shareText.value = response.data.shareText
  apiReturnedAt.value = response.returnedAt.toLocaleTimeString('zh-TW', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
  isLoading.value = false
})
</script>

<template>
  <div class="app-container">
    <AppNavbar />
    <div class="main-layout">
      <div class="sidebar">
        <div class="grid-icons">
          <div v-for="item in 16" :key="item" class="grid-icon-circle" @click="router.push('/introduce')"></div>
        </div>
        <div class="sidebar-card"></div>
        <div class="sidebar-card"></div>
        <div class="sidebar-card"></div>
      </div>

      <div class="feed-content">
        <LoadingPanel v-if="isLoading && !isIframe" />

        <template v-else>
          <div class="post-card show">
            <div class="post-avatar" @click="router.push('/personal')"></div>
            <div class="post-body">
              <div class="post-user-id">@yourid</div>
              <div class="mb-[30px] text-lg text-[#333333]">{{ shareText }}</div>
              <div class="flex items-center justify-between">
                <div class="flex gap-[15px] text-muted">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg>
                  <span class="text-xl font-bold">@</span>
                </div>
                <div class="flex items-center gap-[15px] text-muted">
                  <div class="flex cursor-pointer items-center" @click="isPrivate = !isPrivate">
                    <ClosedEye v-if="isPrivate" />
                    <OpenEye v-else />
                  </div>
                  <button class="post-action-btn min-w-[120px] px-[25px]" @click="isPrivate = !isPrivate">
                    {{ isPrivate ? '設為公開' : '設為私人' }}
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div class="post-card show">
            <div class="post-avatar" @click="router.push('/personal')"></div>
            <div class="post-body">
              <div class="text-sm text-muted">API 回傳時間：{{ apiReturnedAt }}</div>
            </div>
          </div>

          <div v-for="post in feedPosts" :key="post" class="post-card show">
            <div class="post-avatar" @click="router.push('/personal')"></div>
            <div class="flex min-w-0 flex-1 flex-col">
              <div class="post-body">
                <div class="text-base leading-[1.6] text-[#333333]">{{ post }}</div>
              </div>
              <PostActions />
            </div>
          </div>

          <div class="post-card show">
            <div class="post-avatar" @click="router.push('/personal')"></div>
            <div class="flex min-w-0 flex-1 flex-col">
              <div class="post-body p-5">
                <div class="flex h-[350px] w-full items-end justify-center rounded-lg bg-[#cccccc] pb-5">
                  <div class="flex cursor-pointer gap-5 rounded-full bg-[#333333] px-5 py-2 text-white">
                    <span>&lt;</span>
                    <span>&gt;</span>
                  </div>
                </div>
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
