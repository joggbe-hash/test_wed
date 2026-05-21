<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppNavbar from '../components/AppNavbar.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import PostActions from '../components/PostActions.vue'
import { showIntroduceModal } from '../composables/useModal'

import { fetchHomeFeed } from '../api/timedApi'

const route = useRoute()
const router = useRouter()
const isIframe = computed(() => route.query.isIframe === 'true')
const isPrivate = ref(false)
const isLoading = ref(true)
const feedPosts = ref<string[]>([])
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
          <div v-for="item in 16" :key="item" class="grid-icon-circle" @click="showIntroduceModal = true"></div>
        </div>
        <div class="sidebar-card"></div>
        <div class="sidebar-card"></div>
        <div class="sidebar-card"></div>
      </div>

      <div class="feed-content">
        <LoadingPanel v-if="isLoading && !isIframe" />

        <template v-else>
          <!-- Compact Composer -->
          <div class="post-card show !p-4">
            <div class="flex gap-4 w-full">
              <!-- Avatar -->
              <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-[#4a3320] text-white cursor-pointer hover:opacity-80 transition-opacity">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12L12 2l10 10-10 10z"/></svg>
              </div>
              
              <!-- Input and Actions -->
              <div class="flex flex-col w-full pt-1">
                <textarea v-model="shareText" placeholder="來吧，放什麼都行。" class="w-full resize-none bg-transparent text-lg text-[#333] placeholder-[#a59a91] outline-none min-h-[44px]" rows="1"></textarea>
                
                <hr class="my-3 border-[#eaddcf]" />
                
                <div class="flex items-center justify-between">
                  <!-- Toolbar Icons -->
                  <div class="flex shrink-0 gap-3 text-[#a59a91]">
                    <svg class="cursor-pointer hover:text-[#4a3320] transition-colors" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg>
                    <div class="flex h-[20px] w-[26px] items-center justify-center rounded border-[1.5px] border-current text-[9px] font-black cursor-pointer hover:text-[#4a3320] transition-colors">GIF</div>
                    <svg class="cursor-pointer hover:text-[#4a3320] transition-colors" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path></svg>
                    <svg class="cursor-pointer hover:text-[#4a3320] transition-colors" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"></line><line x1="8" y1="12" x2="21" y2="12"></line><line x1="8" y1="18" x2="21" y2="18"></line><line x1="3" y1="6" x2="3.01" y2="6"></line><line x1="3" y1="12" x2="3.01" y2="12"></line><line x1="3" y1="18" x2="3.01" y2="18"></line></svg>
                    <svg class="cursor-pointer hover:text-[#4a3320] transition-colors" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><path d="M8 14s1.5 2 4 2 4-2 4-2"></path><line x1="9" y1="9" x2="9.01" y2="9"></line><line x1="15" y1="9" x2="15.01" y2="9"></line></svg>
                  </div>
                  
                  <!-- Post Action -->
                  <div class="flex items-center gap-4">
                    <div class="cursor-pointer text-[13px] font-bold text-[#a59a91] hover:text-[#4a3320] transition-colors flex items-center gap-1" @click="isPrivate = !isPrivate">
                      {{ isPrivate ? '私人' : '所有人' }}
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>
                    </div>
                    <button class="rounded-full bg-[#4a3320] px-5 py-1.5 text-sm font-bold text-white shadow-sm hover:bg-[#382618] transition-colors opacity-90 hover:opacity-100">
                      張貼
                    </button>
                  </div>
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
