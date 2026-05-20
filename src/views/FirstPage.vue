<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppNavbar from '../components/AppNavbar.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import PostActions from '../components/PostActions.vue'
import { ClosedEye, OpenEye } from '../components/Icons.vue'
import { fetchHomeFeed } from '../api/timedApi'
import { usePageCss } from '../composables/usePageCss'

usePageCss('first_page.css', { materialIcons: true })

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
          <div class="post-card show" style="opacity: 1; transform: translateY(0);">
            <div class="post-avatar" @click="router.push('/personal')"></div>
            <div class="post-body">
              <div class="post-user-id" style="font-size: 20px; font-family: &quot;Intel One Mono&quot;, monospace; font-optical-sizing: auto; font-weight: 500; font-style: normal; margin-bottom: 15px; color: #4A4A4A;">@yourid</div>
              <div style="font-size: 18px; color: #333; margin-bottom: 30px;">{{ shareText }}</div>
              <div style="display: flex; justify-content: space-between; align-items: center;">
                <div style="display: flex; gap: 15px; color: #7A7A7A;">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg>
                  <span style="font-size: 20px; font-weight: bold;">@</span>
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

          <div class="post-card show" style="opacity: 1; transform: translateY(0);">
            <div class="post-avatar" @click="router.push('/personal')"></div>
            <div class="post-body">
              <div style="font-size: 14px; color: #7A7A7A;">API 回傳時間：{{ apiReturnedAt }}</div>
            </div>
          </div>

          <div v-for="post in feedPosts" :key="post" class="post-card show" style="opacity: 1; transform: translateY(0);">
            <div class="post-avatar" @click="router.push('/personal')"></div>
            <div style="display: flex; flex-direction: column; flex: 1; min-width: 0;">
              <div class="post-body">
                <div style="font-size: 16px; color: #333; line-height: 1.6;">{{ post }}</div>
              </div>
              <PostActions />
            </div>
          </div>

          <div class="post-card show" style="opacity: 1; transform: translateY(0);">
            <div class="post-avatar" @click="router.push('/personal')"></div>
            <div style="display: flex; flex-direction: column; flex: 1; min-width: 0;">
              <div class="post-body" style="padding: 20px;">
                <div style="width: 100%; height: 350px; background-color: #CCCCCC; display: flex; justify-content: center; align-items: flex-end; padding-bottom: 20px; box-sizing: border-box; border-radius: 8px;">
                  <div style="background-color: #333333; border-radius: 20px; display: flex; padding: 8px 20px; gap: 20px; color: white; cursor: pointer;">
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
