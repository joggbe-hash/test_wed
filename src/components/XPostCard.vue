<script setup lang="ts">
import type { BackendPost } from '../api/backendApi'
import PostActions from './PostActions.vue'

const props = defineProps<{
  post: BackendPost
  isMenuOpen: boolean
}>()

const emit = defineEmits<{
  openProfile: []
  toggleMenu: [postId: number]
  deletePost: [postId: number]
  openImage: [urls: string[], index: number]
}>()

function visibleImages(post: BackendPost) {
  return post.image_urls?.slice(0, 4) ?? []
}
</script>

<template>
  <article class="x-post">
    <button type="button" class="x-post-avatar" aria-label="查看個人頁" @click="emit('openProfile')">
      <span>{{ props.post.username.slice(0, 1).toUpperCase() }}</span>
    </button>

    <div class="x-post-main">
      <div class="x-post-header">
        <div class="min-w-0">
          <div class="x-post-name">
            <span class="truncate">@{{ props.post.username || 'joggbe' }}</span>
            <span class="x-post-handle">· 9m</span>
          </div>
        </div>

        <div class="relative">
          <button
            type="button"
            class="x-post-menu-btn"
            aria-label="貼文設定"
            @click.stop="emit('toggleMenu', props.post.id)"
          >
            <img src="/icons/settings.png" alt="" class="size-5 opacity-70">
          </button>
          <div v-if="props.isMenuOpen" class="x-post-menu">
            <button type="button" class="x-post-delete" @click.stop="emit('deletePost', props.post.id)">
              刪除貼文
            </button>
          </div>
        </div>
      </div>

      <p v-if="props.post.content" class="x-post-text">{{ props.post.content }}</p>

      <div
        v-if="props.post.image_urls?.length"
        class="x-media-grid"
        :class="`x-media-grid-${Math.min(props.post.image_urls.length, 4)}`"
      >
        <button
          v-for="(url, index) in visibleImages(props.post)"
          :key="`${props.post.id}-${url}`"
          type="button"
          class="x-media-tile"
          :aria-label="`開啟第 ${index + 1} 張圖片`"
          @click="emit('openImage', props.post.image_urls ?? [], index)"
        >
          <img :src="url" alt="" loading="lazy">
          <span v-if="index === 3 && (props.post.image_urls?.length ?? 0) > 4" class="x-media-more">
            +{{ (props.post.image_urls?.length ?? 0) - 4 }}
          </span>
        </button>
      </div>

      <div v-else-if="props.post.image_status === 'processing'" class="x-media-processing">
        圖片處理中
      </div>

      <PostActions />
    </div>
  </article>
</template>
