<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ApiError, createPost, deletePost, fetchFeed } from '../api/backendApi'
import type { BackendPost } from '../api/backendApi'
import AppNavbar from '../components/AppNavbar.vue'
import LoadingPanel from '../components/LoadingPanel.vue'
import PostActions from '../components/PostActions.vue'
import { showIntroduceModal } from '../composables/useModal'

const router = useRouter()
const isPrivate = ref(false)
const isLoading = ref(true)
const isSubmitting = ref(false)
const composerText = ref('')
const imageInput = ref<HTMLInputElement | null>(null)
const selectedImages = ref<File[]>([])
const activeImageIndex = ref(0)
const posts = ref<BackendPost[]>([])
const errorMessage = ref('')
const openPostMenuId = ref<number | null>(null)
const currentImageIndices = ref<Record<number, number>>({})
const imageViewer = ref<{ urls: string[], index: number } | null>(null)

function nextImage(post: BackendPost) {
  if (!post.image_urls || post.image_urls.length <= 1) return
  const current = currentImageIndices.value[post.id] || 0
  currentImageIndices.value[post.id] = (current + 1) % post.image_urls.length
}

function prevImage(post: BackendPost) {
  if (!post.image_urls || post.image_urls.length <= 1) return
  const current = currentImageIndices.value[post.id] || 0
  currentImageIndices.value[post.id] = current === 0 ? post.image_urls.length - 1 : current - 1
}

function setImage(post: BackendPost, index: number) {
  currentImageIndices.value[post.id] = index
}

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

const imagePreviews = computed(() =>
  selectedImages.value.map((image) => ({
    name: image.name,
    url: URL.createObjectURL(image),
  })),
)
const activeImagePreview = computed(() => imagePreviews.value[activeImageIndex.value])

async function loadPosts() {
  try {
    const response = await fetchFeed()
    posts.value = response.posts
    errorMessage.value = ''
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      await router.push('/login')
      return
    }
    errorMessage.value = error instanceof Error ? error.message : '讀取貼文失敗'
  }
}

async function handleCreatePost() {
  const content = composerText.value.trim()
  if ((!content && selectedImages.value.length === 0) || isSubmitting.value) {
    return
  }

  isSubmitting.value = true
  try {
    const result = await createPost(content, selectedImages.value)
    
    // 如果有上傳圖片，等待後端處理完成再發文
    if (selectedImages.value.length > 0) {
      let isProcessed = false
      let attempts = 0
      while (!isProcessed && attempts < 20) {
        await new Promise(resolve => setTimeout(resolve, 1000))
        const response = await fetchFeed()
        const newPost = response.posts.find(p => p.id === result.post_id)
        if (!newPost || newPost.image_status !== 'processing') {
          isProcessed = true
        }
        attempts++
      }
    }

    composerText.value = ''
    selectedImages.value = []
    activeImageIndex.value = 0
    if (imageInput.value) {
      imageInput.value.value = ''
    }
    await loadPosts()
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      await router.push('/login')
      return
    }
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
  selectedImages.value = Array.from(input.files ?? [])
  activeImageIndex.value = 0
}

function showPreviousImage() {
  if (selectedImages.value.length <= 1) {
    return
  }
  activeImageIndex.value =
    activeImageIndex.value === 0 ? selectedImages.value.length - 1 : activeImageIndex.value - 1
}

function showNextImage() {
  if (selectedImages.value.length <= 1) {
    return
  }
  activeImageIndex.value = (activeImageIndex.value + 1) % selectedImages.value.length
}

function removeActiveImage() {
  selectedImages.value = selectedImages.value.filter((_, index) => index !== activeImageIndex.value)
  if (activeImageIndex.value >= selectedImages.value.length) {
    activeImageIndex.value = Math.max(selectedImages.value.length - 1, 0)
  }
  if (selectedImages.value.length === 0 && imageInput.value) {
    imageInput.value.value = ''
  }
}

async function handleDeletePost(postId: number) {
  try {
    await deletePost(postId)
    posts.value = posts.value.filter((post) => post.id !== postId)
    openPostMenuId.value = null
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      await router.push('/login')
      return
    }
    errorMessage.value = error instanceof Error ? error.message : '刪除貼文失敗'
  }
}

onMounted(async () => {
  isLoading.value = true
  await loadPosts()
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
                  placeholder="想說些什麼"
                  class="min-h-[44px] w-full resize-none bg-transparent text-lg text-[#333] placeholder-[#a59a91] outline-none"
                  rows="1"
                ></textarea>

                <hr class="my-3 border-[#eaddcf]">

                <div v-if="activeImagePreview" class="mb-3">
                  <div class="relative flex w-full max-w-[640px] items-center justify-center overflow-hidden rounded-[16px] border border-[#d8d1ca] bg-white">
                    <img :src="activeImagePreview.url" :alt="activeImagePreview.name" class="block h-auto max-h-[520px] max-w-full object-contain">
                    <button
                      v-if="selectedImages.length > 1"
                      type="button"
                      class="absolute left-4 top-1/2 flex size-9 -translate-y-1/2 items-center justify-center rounded-full bg-[#333333]/80 text-xl font-bold text-white"
                      aria-label="上一張圖片"
                      @click="showPreviousImage"
                    >
                      &lt;
                    </button>
                    <button
                      v-if="selectedImages.length > 1"
                      type="button"
                      class="absolute right-4 top-1/2 flex size-9 -translate-y-1/2 items-center justify-center rounded-full bg-[#333333]/80 text-xl font-bold text-white"
                      aria-label="下一張圖片"
                      @click="showNextImage"
                    >
                      &gt;
                    </button>
                    <button
                      type="button"
                      class="absolute right-4 top-4 flex size-8 items-center justify-center rounded-full bg-[#4a3320] text-sm font-bold text-white"
                      aria-label="移除圖片"
                      @click="removeActiveImage"
                    >
                      x
                    </button>
                  </div>
                  <div class="mt-2 flex max-w-[520px] justify-center gap-2">
                    <button
                      v-for="(_, index) in selectedImages"
                      :key="index"
                      type="button"
                      class="h-3 w-3 border-0"
                      :class="index === activeImageIndex ? 'bg-[#4a3320]' : 'bg-[#d9d9d9]'"
                      :aria-label="`第 ${index + 1} 張圖片`"
                      @click="activeImageIndex = index"
                    ></button>
                    <span class="ml-2 text-sm font-bold text-muted">
                      {{ activeImageIndex + 1 }} / {{ selectedImages.length }}
                    </span>
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
                      :disabled="isSubmitting || (!composerText.trim() && selectedImages.length === 0)"
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

          <div v-for="post in posts" :key="post.id" class="post-card show">
            <div class="post-avatar" @click="router.push('/personal')"></div>
            <div class="flex min-w-0 flex-1 flex-col">
              <div class="post-body">
                <div class="absolute right-4 top-4">
                  <button
                    type="button"
                    class="flex size-8 items-center justify-center rounded-full border-0 bg-transparent transition-colors hover:bg-[#f4efea]"
                    aria-label="貼文設定"
                    @click.stop="openPostMenuId = openPostMenuId === post.id ? null : post.id"
                  >
                    <img src="/icons/settings.png" alt="" class="size-5 opacity-75">
                  </button>
                  <div
                    v-if="openPostMenuId === post.id"
                    class="absolute right-0 top-9 z-20 w-32 overflow-hidden rounded bg-white shadow-[0_8px_20px_rgba(0,0,0,0.15)]"
                  >
                    <button
                      type="button"
                      class="w-full border-0 bg-white px-4 py-3 text-left text-sm font-bold text-[#cc3333] hover:bg-[#f5d5d5]"
                      @click.stop="handleDeletePost(post.id)"
                    >
                      刪除貼文
                    </button>
                  </div>
                </div>
                <div class="post-user-id">{{ post.username }}</div>
                <div v-if="post.content" class="text-base leading-[1.6] text-[#333333]">{{ post.content }}</div>
                <div v-if="post.image_urls?.length" class="relative mt-5 flex w-full items-center justify-center overflow-hidden rounded-[16px] border border-[#d8d1ca] bg-white">
                  <img
                    :src="post.image_urls[currentImageIndices[post.id] || 0]"
                    alt=""
                    class="block h-auto max-h-[680px] max-w-full cursor-zoom-in object-contain transition-opacity duration-300"
                    @error="($event.target as HTMLImageElement).style.opacity = '0'"
                    @load="($event.target as HTMLImageElement).style.opacity = '1'"
                    @click="openImageViewer(post.image_urls, currentImageIndices[post.id] || 0)"
                  >
                  
                  <!-- 上一頁按鈕 -->
                  <button
                    v-if="post.image_urls.length > 1"
                    type="button"
                    class="absolute left-3 top-1/2 flex size-[30px] -translate-y-1/2 items-center justify-center rounded-full bg-white/90 text-gray-700 shadow-sm transition-colors hover:bg-white"
                    @click.stop="prevImage(post)"
                  >
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>
                  </button>
                  
                  <!-- 下一頁按鈕 -->
                  <button
                    v-if="post.image_urls.length > 1"
                    type="button"
                    class="absolute right-3 top-1/2 flex size-[30px] -translate-y-1/2 items-center justify-center rounded-full bg-white/90 text-gray-700 shadow-sm transition-colors hover:bg-white"
                    @click.stop="nextImage(post)"
                  >
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>
                  </button>
                  
                  <!-- 右上角數字標示 -->
                  <div
                    v-if="post.image_urls.length > 1"
                    class="absolute right-4 top-4 rounded-full bg-black/60 px-2.5 py-1 text-xs font-medium text-white backdrop-blur-sm"
                  >
                    {{ (currentImageIndices[post.id] || 0) + 1 }} / {{ post.image_urls.length }}
                  </div>

                  <!-- 底部小圓點 -->
                  <div v-if="post.image_urls.length > 1" class="absolute bottom-4 left-0 right-0 flex justify-center gap-[5px]">
                    <button
                      v-for="(_, index) in post.image_urls"
                      :key="index"
                      type="button"
                      class="size-[6px] rounded-full transition-colors"
                      :class="index === (currentImageIndices[post.id] || 0) ? 'bg-white' : 'bg-white/40'"
                      @click.stop="setImage(post, index)"
                    />
                  </div>
                </div>
                <div v-else-if="post.image_status === 'processing'" class="h-[350px] w-full rounded bg-[#e0e0e0]"></div>
              </div>
              <PostActions />
            </div>
          </div>
        </template>
      </div>
    </div>
    <div class="fab"></div>

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
  </div>
</template>
