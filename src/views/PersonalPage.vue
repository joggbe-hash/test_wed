<script setup>
import { onMounted, ref } from 'vue'
import AppNavbar from '../components/AppNavbar.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import PostActions from '../components/PostActions.vue'
import { fetchPersonalData } from '../api/timedApi'
import { usePageCss } from '../composables/usePageCss'

usePageCss('personal_page.css', { materialIcons: true })

const isLoading = ref(true)
const profile = ref({ id: '', bio: '' })
const posts = ref([])
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
          <div class="post-card">
            <div class="post-avatar"></div>
            <div class="post-body">
              <div class="post-user-id" style="font-size: 20px; font-family: &quot;Intel One Mono&quot;, monospace; font-optical-sizing: auto; font-weight: 500; font-style: normal; margin-bottom: 15px; color: #333;">{{ profile.id }}</div>
              <div style="font-size: 16px; color: #333; line-height: 1.6; margin-bottom: 30px;">{{ profile.bio }}</div>
              <div style="display: flex; gap: 15px;">
                <button class="profile-btn">編輯個人資料</button>
                <button class="profile-btn">分享個人資料</button>
              </div>
            </div>
          </div>

          <div v-for="post in posts" :key="post" class="post-card">
            <div class="post-avatar"></div>
            <div style="display: flex; flex-direction: column; flex: 1; min-width: 0;">
              <div class="post-body">
                <div style="font-size: 16px; color: #333; line-height: 1.6;">{{ post }}</div>
              </div>
              <PostActions />
            </div>
          </div>

          <div class="post-card">
            <div class="post-avatar black"></div>
            <div style="display: flex; flex-direction: column; flex: 1; min-width: 0;">
              <div class="post-body">
                <div style="font-size: 32px; font-weight: bold; font-family: &quot;Intel One Mono&quot;, monospace; font-optical-sizing: auto; font-style: normal; letter-spacing: 2px; margin-bottom: 20px;">{{ timer }}</div>
                <div style="width: 100%; height: 250px; background-color: #A0A0A0; border-radius: 4px;"></div>
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
