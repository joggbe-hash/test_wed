<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { createPost, deletePost, fetchFeed } from '../api/backendApi'
import MainLayout from '../layouts/MainLayout.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import XPostCard from '../components/XPostCard.vue'
import { useFeedStore } from '../stores/useFeedStore'
import { showIntroduceModal } from '../composables/useModal'

interface ImagePreview {
  name: string
  url: string
}

const route = useRoute()
const router = useRouter()
const feedStore = useFeedStore()
const isIframe = computed(() => route.query.isIframe === 'true')
const isPrivate = ref(false)
const isSubmitting = ref(false)
const shareText = ref('')
const imageInput = ref<HTMLInputElement | null>(null)
const selectedImages = ref<File[]>([])
const imagePreviews = ref<ImagePreview[]>([])
const apiReturnedAt = ref('')
const errorMessage = ref('')
const openPostMenuId = ref<number | null>(null)
const imageViewer = ref<{ urls: string[], index: number } | null>(null)

function revokeImagePreviews() {
  imagePreviews.value.forEach((preview) => URL.revokeObjectURL(preview.url))
  imagePreviews.value = []
}

watch(selectedImages, (images) => {
  revokeImagePreviews()
  imagePreviews.value = images.slice(0, 4).map((image) => ({
    name: image.name,
    url: URL.createObjectURL(image),
  }))
})

function openImageViewer(urls: string[], index: number) {
  imageViewer.value = { urls, index }
}

function closeImageViewer() {
  imageViewer.value = null
}

function showPreviousViewerImage() {
  if (!imageViewer.value || imageViewer.value.urls.length <= 1) return
  imageViewer.value.index =
    imageViewer.value.index === 0 ? imageViewer.value.urls.length - 1 : imageViewer.value.index - 1
}

function showNextViewerImage() {
  if (!imageViewer.value || imageViewer.value.urls.length <= 1) return
  imageViewer.value.index = (imageViewer.value.index + 1) % imageViewer.value.urls.length
}

async function loadFeed() {
  try {
    await feedStore.loadPosts()
    apiReturnedAt.value = new Date().toLocaleTimeString('zh-TW', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
    errorMessage.value = ''
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '讀取貼文失敗'
  }
}

async function handleCreatePost() {
  const content = shareText.value.trim()
  if ((!content && selectedImages.value.length === 0) || isSubmitting.value) {
    return
  }

  isSubmitting.value = true
  try {
    const imagesToUpload = selectedImages.value.slice(0, 4)
    const result = await createPost(content, imagesToUpload)

    if (imagesToUpload.length > 0) {
      let isProcessed = false
      let attempts = 0
      while (!isProcessed && attempts < 20) {
        await new Promise((resolve) => setTimeout(resolve, 1000))
        const response = await fetchFeed()
        const newPost = response.posts.find((post) => post.id === result.post_id)
        if (!newPost || newPost.image_status !== 'processing') {
          isProcessed = true
        }
        attempts++
      }
    }

    shareText.value = ''
    selectedImages.value = []
    if (imageInput.value) {
      imageInput.value.value = ''
    }
    await feedStore.loadPosts(true)
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '發文失敗'
  } finally {
    isSubmitting.value = false
  }
}

function openImagePicker() {
  imageInput.value?.click()
}

function handleImageChange(event: Event) {
  const input = event.target as HTMLInputElement
  selectedImages.value = Array.from(input.files ?? []).slice(0, 4)
}

function removeSelectedImage(imageIndex: number) {
  selectedImages.value = selectedImages.value.filter((_, index) => index !== imageIndex)
  if (selectedImages.value.length === 0 && imageInput.value) {
    imageInput.value.value = ''
  }
}

async function handleDeletePost(postId: number) {
  try {
    await deletePost(postId)
    feedStore.removePost(postId)
    openPostMenuId.value = null
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '刪除貼文失敗'
  }
}

onMounted(async () => {
  await loadFeed()
})

onBeforeUnmount(() => {
  revokeImagePreviews()
})
</script>

<template>
  <MainLayout active-nav="home">
    <template #sidebar>
      <div class="grid-icons">
        <div v-for="item in 16" :key="item" class="grid-icon-circle" @click="showIntroduceModal = true"></div>
      </div>
      <div class="sidebar-card"></div>
      <div class="sidebar-card"></div>
      <div class="sidebar-card"></div>
    </template>

    <LoadingPanel v-if="feedStore.isLoading && !isIframe" />
    <template v-else>
      <div class="post-card show !p-4">
        <div class="flex w-full gap-4">
          <div class="flex h-12 w-12 shrink-0 cursor-pointer items-center justify-center rounded-full bg-[#4a3320] text-white transition-opacity hover:opacity-80">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12L12 2l10 10-10 10z"/></svg>
          </div>

          <div class="flex w-full flex-col pt-1">
            <textarea
              v-model="shareText"
              placeholder="分享你的想法"
              class="min-h-[44px] w-full resize-none bg-transparent text-lg text-[#333] placeholder-[#a59a91] outline-none"
              rows="1"
            ></textarea>

            <hr class="my-3 border-[#eaddcf]">

            <div
              v-if="imagePreviews.length"
              class="compose-media-grid"
              :class="`compose-media-grid-${Math.min(imagePreviews.length, 4)}`"
            >
              <div
                v-for="(image, index) in imagePreviews"
                :key="`${image.name}-${index}`"
                class="compose-media-tile"
              >
                <img :src="image.url" :alt="image.name">
                <button type="button" class="compose-edit-btn">Edit</button>
                <button
                  type="button"
                  class="compose-remove-btn"
                  :aria-label="`移除第 ${index + 1} 張圖片`"
                  @click="removeSelectedImage(index)"
                >
                  ×
                </button>
              </div>
            </div>

            <div class="flex items-center justify-between">
              <div class="flex shrink-0 gap-3 text-[#a59a91]">
                <button
                  type="button"
                  class="cursor-pointer border-0 bg-transparent p-0 text-current transition-colors hover:text-[#4a3320]"
                  aria-label="選擇圖片"
                  @click="openImagePicker"
                >
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg>
                </button>
                <input
                  ref="imageInput"
                  type="file"
                  class="sr-only"
                  accept="image/*"
                  multiple
                  @change="handleImageChange"
                >
                <div class="flex h-[20px] w-[26px] cursor-pointer items-center justify-center rounded border-[1.5px] border-current text-[9px] font-black transition-colors hover:text-[#4a3320]">GIF</div>
                <svg class="cursor-pointer transition-colors hover:text-[#4a3320]" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path></svg>
              </div>

              <div class="flex items-center gap-4">
                <div class="flex cursor-pointer items-center gap-1 text-[13px] font-bold text-[#a59a91] transition-colors hover:text-[#4a3320]" @click="isPrivate = !isPrivate">
                  {{ isPrivate ? '私人' : '公開' }}
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>
                </div>
                <button
                  class="rounded-full bg-[#4a3320] px-5 py-1.5 text-sm font-bold text-white opacity-90 shadow-sm transition-colors hover:bg-[#382618] hover:opacity-100 disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="isSubmitting || (!shareText.trim() && selectedImages.length === 0)"
                  @click="handleCreatePost"
                >
                  {{ isSubmitting ? '發佈中' : '發文' }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="errorMessage" class="post-card show">
        <div class="post-body text-sm text-[#cc3333]">{{ errorMessage }}</div>
      </div>

      <div class="post-card show">
        <div class="post-avatar" @click="router.push('/personal')"></div>
        <div class="post-body">
          <div class="text-sm text-muted">API 回應時間：{{ apiReturnedAt || '尚未讀取' }}</div>
        </div>
      </div>

      <XPostCard
        v-for="post in feedStore.posts"
        :key="'x-' + post.id"
        :post="post"
        :is-menu-open="openPostMenuId === post.id"
        @open-profile="router.push('/personal')"
        @toggle-menu="openPostMenuId = openPostMenuId === $event ? null : $event"
        @delete-post="handleDeletePost"
        @open-image="openImageViewer"
      />
    </template>

    <div
      v-if="imageViewer"
      class="fixed inset-0 z-[300] flex items-center justify-center bg-black/90"
      @click="closeImageViewer"
    >
      <button
        type="button"
        class="absolute left-5 top-5 flex size-11 items-center justify-center rounded-full bg-black/60 text-3xl text-white transition-colors hover:bg-white/15"
        aria-label="關閉圖片"
        @click.stop="closeImageViewer"
      >
        ×
      </button>

      <button
        v-if="imageViewer.urls.length > 1"
        type="button"
        class="absolute left-5 top-1/2 flex size-12 -translate-y-1/2 items-center justify-center rounded-full bg-black/60 text-3xl text-white transition-colors hover:bg-white/15"
        aria-label="上一張圖片"
        @click.stop="showPreviousViewerImage"
      >
        ‹
      </button>

      <img
        :src="imageViewer.urls[imageViewer.index]"
        alt=""
        class="max-h-[82dvh] max-w-[92vw] object-contain"
        @click.stop
      >

      <button
        v-if="imageViewer.urls.length > 1"
        type="button"
        class="absolute right-5 top-1/2 flex size-12 -translate-y-1/2 items-center justify-center rounded-full bg-black/60 text-3xl text-white transition-colors hover:bg-white/15"
        aria-label="下一張圖片"
        @click.stop="showNextViewerImage"
      >
        ›
      </button>

      <div class="absolute bottom-6 left-1/2 flex -translate-x-1/2 items-center gap-16 text-white">
        <button type="button" class="text-2xl" aria-label="回覆" @click.stop>♡</button>
        <button type="button" class="text-2xl" aria-label="轉發" @click.stop>↻</button>
        <button type="button" class="text-2xl" aria-label="喜歡" @click.stop>♥</button>
        <button type="button" class="text-2xl" aria-label="分享" @click.stop>⇧</button>
      </div>
    </div>
  </MainLayout>
</template>
