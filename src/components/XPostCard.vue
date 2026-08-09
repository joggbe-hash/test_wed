<script setup lang="ts">
import { computed, shallowRef } from 'vue'
import type { BackendPost } from '../api/backendApi'
import { useSharedMinuteNow } from '../composables/useSharedMinuteNow'
import { publicAsset } from '../utils/assets'
import PostActions from './PostActions.vue'
import TaskDeleteConfirmDialog from './TaskDeleteConfirmDialog.vue'

const props = defineProps<{
  post: BackendPost
  isMenuOpen: boolean
  canDelete: boolean
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

const isDeleteConfirmOpen = shallowRef(false)
const postMenuId = computed(() => `post-menu-${props.post.id}`)
const postMenuTriggerId = computed(() => `post-menu-trigger-${props.post.id}`)
const deleteDialogItem = computed(() => ({ id: props.post.id, title: '這篇貼文' }))

const minuteMilliseconds = 60 * 1000
const hourMilliseconds = 60 * minuteMilliseconds
const dayMilliseconds = 24 * hourMilliseconds
const now = useSharedMinuteNow()

const postTimeLabel = computed(() => {
  const createdAt = Date.parse(props.post.created_at)
  if (Number.isNaN(createdAt)) return ''

  const elapsed = Math.max(0, now.value - createdAt)
  if (elapsed < minuteMilliseconds) return '剛剛'
  if (elapsed < hourMilliseconds) return `${Math.floor(elapsed / minuteMilliseconds)} 分鐘`
  if (elapsed < dayMilliseconds) return `${Math.floor(elapsed / hourMilliseconds)} 小時`
  return `${Math.floor(elapsed / dayMilliseconds)} 天`
})

const postTimeTitle = computed(() => {
  const createdAt = new Date(props.post.created_at)
  if (Number.isNaN(createdAt.getTime())) return undefined

  return new Intl.DateTimeFormat('zh-TW', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(createdAt)
})

function requestDeletePost() {
  if (!props.canDelete) return
  isDeleteConfirmOpen.value = true
}

function closeDeleteConfirmation() {
  isDeleteConfirmOpen.value = false
}

function confirmDeletePost() {
  if (!props.canDelete) return
  isDeleteConfirmOpen.value = false
  emit('deletePost', props.post.id)
}

const settingsIconUrl = publicAsset('icons/settings.png')
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
            <time
              v-if="postTimeLabel"
              class="x-post-handle"
              :datetime="props.post.created_at"
              :title="postTimeTitle"
            >
              · {{ postTimeLabel }}
            </time>
          </div>
        </div>

        <div v-if="props.canDelete" class="relative">
          <button
            :id="postMenuTriggerId"
            type="button"
            class="x-post-menu-btn"
            aria-label="貼文設定"
            :aria-expanded="props.isMenuOpen"
            :aria-controls="postMenuId"
            @click.stop="emit('toggleMenu', props.post.id)"
          >
            <img :src="settingsIconUrl" alt="" width="20" height="20" class="size-5 opacity-70">
          </button>
          <div v-if="props.isMenuOpen" :id="postMenuId" class="x-post-menu">
            <button
              type="button"
              class="x-post-delete"
              aria-haspopup="dialog"
              @click.stop="requestDeletePost"
            >
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
          <img :src="url" :alt="`貼文圖片 ${index + 1}`" width="1600" height="900" loading="lazy">
          <span v-if="index === 3 && (props.post.image_urls?.length ?? 0) > 4" class="x-media-more">
            +{{ (props.post.image_urls?.length ?? 0) - 4 }}
          </span>
        </button>
      </div>

      <div v-else-if="props.post.image_status === 'processing'" class="x-media-processing">
        圖片處理中
      </div>

      <div v-else-if="props.post.image_status === 'failed'" class="x-media-processing x-media-failed" role="status">
        圖片處理失敗，請刪除貼文後重新上傳。
      </div>

      <PostActions />
    </div>
  </article>

  <TaskDeleteConfirmDialog
    v-if="props.canDelete && isDeleteConfirmOpen"
    :item="deleteDialogItem"
    kind="post"
    :return-focus-id="postMenuTriggerId"
    @cancel="closeDeleteConfirmation"
    @confirm="confirmDeletePost"
  />
</template>
