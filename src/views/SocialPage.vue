<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppNavbar from '../components/AppNavbar.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import PostActions from '../components/PostActions.vue'
import { ClosedEye, OpenEye } from '../components/Icons.vue'
import { fetchSocialData } from '../api/timedApi'
import { usePageCss } from '../composables/usePageCss'

usePageCss('social_page.css', { materialIcons: true })

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
              <div style="display: flex; gap: 15px;">
                <div style="width: 80px; height: 20px; background-color: rgba(255,255,255,0.2); border-radius: 4px;"></div>
                <div style="width: 120px; height: 20px; background-color: rgba(255,255,255,0.2); border-radius: 4px;"></div>
              </div>
              <div style="display: flex; gap: 15px;">
                <div style="width: 80px; height: 35px; background-color: #FFFFFF; border-radius: 20px;"></div>
                <div style="width: 30px; height: 35px; background-color: rgba(255,255,255,0.2); border-radius: 4px;"></div>
              </div>
            </div>
            <div class="theme-banner-bottom">
              <div style="height: 35px; width: 40%; background-color: rgba(255,255,255,0.8); border-radius: 8px;"></div>
              <div style="height: 15px; width: 100%; background-color: rgba(255,255,255,0.4); border-radius: 4px;"></div>
              <div style="height: 15px; width: 60%; background-color: rgba(255,255,255,0.4); border-radius: 4px;"></div>
            </div>
            <div style="position: absolute; bottom: 0; right: 50px; width: 100px; height: 100px; background-color: rgba(255,255,255,0.2); border-radius: 50% 50% 0 0;"></div>
          </div>

          <div class="post-card show" style="opacity: 1; transform: translateY(0);">
            <div class="post-avatar" @click="router.push('/personal')"></div>
            <div class="post-body">
              <div class="post-user-id" style="font-size: 20px; font-family: &quot;Intel One Mono&quot;, monospace; font-optical-sizing: auto; font-weight: 500; font-style: normal; margin-bottom: 15px; color: #4A4A4A;">@yourid</div>
              <div style="font-size: 18px; color: #333; margin-bottom: 30px;">{{ composerText }}</div>
              <div style="display: flex; justify-content: space-between; align-items: center;">
                <div style="display: flex; gap: 15px; color: #7A7A7A; align-items: center;">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="cursor:pointer;"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg>
                  <span style="font-size: 20px; font-weight: bold; cursor:pointer;">@</span>
                </div>
                <div style="display: flex; align-items: center; gap: 15px; color: #7A7A7A;">
                  <div style="cursor: pointer; display: flex; align-items: center;" @click="isPrivate = !isPrivate">
                    <ClosedEye v-if="isPrivate" />
                    <OpenEye v-else />
                  </div>
                  <button class="post-action-btn" style="position: static; padding: 10px 25px; background-color: #4A3320; min-width: 120px;" @click="isPrivate = !isPrivate">
                    {{ isPrivate ? '私人發布' : '公開分享' }}
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div v-for="post in posts" :key="post.id" class="post-card show">
            <div class="post-avatar" @click="router.push('/personal')"></div>
            <div style="display: flex; flex-direction: column; flex: 1; min-width: 0;">
              <div class="post-body">
                <div v-if="post.type === 'text'" style="font-size: 16px; color: #333; line-height: 1.6;">{{ post.text }}</div>
                <div v-else style="width: 100%; height: 350px; background-color: var(--hover-bg); border-radius: 4px;"></div>
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
