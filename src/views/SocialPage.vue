<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppNavbar from '../components/AppNavbar.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import PostActions from '../components/PostActions.vue'
import { showIntroduceModal } from '../composables/useModal'
import { fetchSocialData } from '../api/timedApi'
import type { SocialPost } from '../api/timedApi'

const router = useRouter()
const isPrivate = ref(false)
const isLoading = ref(true)
const composerText = ref('')
const posts = ref<SocialPost[]>([])

onMounted(async () => {
  isLoading.value = true
  const response = await fetchSocialData()
  composerText.value = response.data.composerText
  posts.value = response.data.posts
  isLoading.value = false
})
</script>

<template>
  <div class="app-container">
    <AppNavbar />
    <div class="main-layout">
      <div class="sidebar">
        <div class="sidebar-card"></div>
        <div class="sidebar-card"></div>
        <div class="sidebar-card"></div>
        <div class="grid-icons">
          <div v-for="item in 16" :key="item" class="grid-icon-circle" @click="showIntroduceModal = true"></div>
        </div>
      </div>

      <div class="feed-content">
        <LoadingPanel v-if="isLoading" />

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
            <div class="absolute right-[50px] bottom-0 h-[100px] w-[100px] rounded-t-full bg-white/20"></div>
          </div>

          <div class="post-card show !p-4">
            <div class="flex w-full gap-4">
              <div class="flex h-12 w-12 shrink-0 cursor-pointer items-center justify-center rounded-full bg-[#4a3320] text-white transition-opacity hover:opacity-80">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12L12 2l10 10-10 10z"/></svg>
              </div>

              <div class="flex w-full flex-col pt-1">
                <textarea
                  v-model="composerText"
                  placeholder="想分享什麼？"
                  class="min-h-[44px] w-full resize-none bg-transparent text-lg text-[#333] placeholder-[#a59a91] outline-none"
                  rows="1"
                ></textarea>

                <hr class="my-3 border-[#eaddcf]">

                <div class="flex items-center justify-between">
                  <div class="flex shrink-0 gap-3 text-[#a59a91]">
                    <svg class="cursor-pointer transition-colors hover:text-[#4a3320]" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg>
                    <div class="flex h-[20px] w-[26px] cursor-pointer items-center justify-center rounded border-[1.5px] border-current text-[9px] font-black transition-colors hover:text-[#4a3320]">GIF</div>
                    <svg class="cursor-pointer transition-colors hover:text-[#4a3320]" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path></svg>
                    <svg class="cursor-pointer transition-colors hover:text-[#4a3320]" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"></line><line x1="8" y1="12" x2="21" y2="12"></line><line x1="8" y1="18" x2="21" y2="18"></line><line x1="3" y1="6" x2="3.01" y2="6"></line><line x1="3" y1="12" x2="3.01" y2="12"></line><line x1="3" y1="18" x2="3.01" y2="18"></line></svg>
                    <svg class="cursor-pointer transition-colors hover:text-[#4a3320]" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><path d="M8 14s1.5 2 4 2 4-2 4-2"></path><line x1="9" y1="9" x2="9.01" y2="9"></line><line x1="15" y1="9" x2="15.01" y2="9"></line></svg>
                  </div>

                  <div class="flex items-center gap-4">
                    <div class="flex cursor-pointer items-center gap-1 text-[13px] font-bold text-[#a59a91] transition-colors hover:text-[#4a3320]" @click="isPrivate = !isPrivate">
                      {{ isPrivate ? '私人' : '所有人' }}
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>
                    </div>
                    <button class="rounded-full bg-[#4a3320] px-5 py-1.5 text-sm font-bold text-white opacity-90 shadow-sm transition-colors hover:bg-[#382618] hover:opacity-100">
                      張貼
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-for="post in posts" :key="post.id" class="post-card show">
            <div class="post-avatar" @click="router.push('/personal')"></div>
            <div class="flex min-w-0 flex-1 flex-col">
              <div class="post-body">
                <div v-if="post.type === 'text'" class="text-base leading-[1.6] text-[#333333]">{{ post.text }}</div>
                <div v-else class="h-[350px] w-full rounded bg-[#e0e0e0]"></div>
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
