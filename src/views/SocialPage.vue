<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppNavbar from '../components/AppNavbar.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import PostActions from '../components/PostActions.vue'
import { ClosedEye, OpenEye } from '../components/Icons.vue'
import { fetchSocialData } from '../api/timedApi'

const router = useRouter()
const isPrivate = ref(false)
const isLoading = ref(true)
const composerText = ref('')
const posts = ref([])

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
          <div v-for="item in 16" :key="item" class="grid-icon-circle" @click="router.push('/introduce')"></div>
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

          <div class="post-card show">
            <div class="post-avatar" @click="router.push('/personal')"></div>
            <div class="post-body">
              <div class="post-user-id">@yourid</div>
              <div class="mb-[30px] text-lg text-[#333333]">{{ composerText }}</div>
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-[15px] text-muted">
                  <svg class="cursor-pointer" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg>
                  <span class="cursor-pointer text-xl font-bold">@</span>
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
