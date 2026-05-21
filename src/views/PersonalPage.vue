<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppNavbar from '../components/AppNavbar.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import PostActions from '../components/PostActions.vue'
import { fetchPersonalData } from '../api/timedApi'

const isLoading = ref(true)
const profile = ref({ id: '', bio: '' })
const posts = ref<string[]>([])
const timer = ref('')

onMounted(async () => {
  isLoading.value = true
  const response = await fetchPersonalData()
  profile.value = response.data.profile
  posts.value = response.data.posts
  timer.value = response.data.timer
  isLoading.value = false
})
</script>

<template>
  <div class="app-container">
    <AppNavbar active="personal" />
    <div class="main-layout">
      <div class="sidebar"></div>

      <div class="feed-content">
        <LoadingPanel v-if="isLoading" />

        <template v-else>
          <div class="post-card show">
            <div class="post-avatar"></div>
            <div class="post-body">
              <div class="post-user-id text-[#333333]">{{ profile.id }}</div>
              <div class="mb-[30px] text-base leading-[1.6] text-[#333333]">{{ profile.bio }}</div>
              <div class="flex gap-[15px]">
                <button class="profile-btn">編輯個人資料</button>
                <button class="profile-btn">分享個人資料</button>
              </div>
            </div>
          </div>

          <div v-for="post in posts" :key="post" class="post-card show">
            <div class="post-avatar"></div>
            <div class="flex min-w-0 flex-1 flex-col">
              <div class="post-body">
                <div class="text-base leading-[1.6] text-[#333333]">{{ post }}</div>
              </div>
              <PostActions />
            </div>
          </div>

          <div class="post-card show">
            <div class="post-avatar black"></div>
            <div class="flex min-w-0 flex-1 flex-col">
              <div class="post-body">
                <div class="mb-5 text-[32px] font-bold tracking-[2px]">{{ timer }}</div>
                <div class="h-[250px] w-full rounded bg-[#a0a0a0]"></div>
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
