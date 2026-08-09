<script setup lang="ts">
import { onBeforeUnmount, ref, shallowRef, useTemplateRef, watch } from 'vue'
import { createPost } from '../api/backendApi'
import { apiErrorMessage } from '../api/errors'
import { useAccessibleDialog } from '../composables/useAccessibleDialog'
import { closeComposeModal } from '../composables/useComposeModal'
import { useFeedStore } from '../stores/useFeedStore'
import {
  acceptedPostImageInput,
  maxPostImages,
  validatePostImages,
} from '../utils/postImages'

interface ImagePreview {
  name: string
  url: string
}

const feedStore = useFeedStore()
const content = shallowRef('')
const imageInput = useTemplateRef<HTMLInputElement>('imageInput')
const selectedImages = ref<File[]>([])
const imagePreviews = ref<ImagePreview[]>([])
const isSubmitting = shallowRef(false)
const errorMessage = shallowRef('')
const visibility = shallowRef<'public' | 'private'>('public')
const dialog = useTemplateRef<HTMLElement>('dialog')
const cancelButton = useTemplateRef<HTMLButtonElement>('cancelButton')

function revokeImagePreviews() {
  imagePreviews.value.forEach((preview) => URL.revokeObjectURL(preview.url))
  imagePreviews.value = []
}

watch(selectedImages, (images) => {
  revokeImagePreviews()
  imagePreviews.value = images.slice(0, maxPostImages).map((image) => ({
    name: image.name,
    url: URL.createObjectURL(image),
  }))
})

function openImagePicker() {
  imageInput.value?.click()
}

function handleImageChange(event: Event) {
  const input = event.target as HTMLInputElement
  const validation = validatePostImages(input.files ?? [])
  errorMessage.value = validation.errorMessage
  selectedImages.value = validation.accepted
  if (validation.errorMessage) input.value = ''
}

function removeSelectedImage(imageIndex: number) {
  selectedImages.value = selectedImages.value.filter((_, index) => index !== imageIndex)
  if (selectedImages.value.length === 0 && imageInput.value) {
    imageInput.value.value = ''
  }
}

function closeModal() {
  if (isSubmitting.value) return
  closeComposeModal()
}

useAccessibleDialog({
  dialog,
  initialFocus: cancelButton,
  onClose: closeModal,
  backgroundSelector: '[data-app-route-content]',
})

async function submitPost() {
  const trimmedContent = content.value.trim()
  if ((!trimmedContent && selectedImages.value.length === 0) || isSubmitting.value) {
    return
  }

  isSubmitting.value = true
  errorMessage.value = ''

  try {
    const validation = validatePostImages(selectedImages.value)
    if (validation.errorMessage) {
      errorMessage.value = validation.errorMessage
      return
    }

    const imagesToUpload = validation.accepted
    const result = await createPost(trimmedContent, imagesToUpload, visibility.value)

    content.value = ''
    selectedImages.value = []
    visibility.value = 'public'
    if (imageInput.value) {
      imageInput.value.value = ''
    }
    await feedStore.loadPosts(true)
    if (imagesToUpload.length > 0) void feedStore.trackPostProcessing(result.post_id)
    closeComposeModal()
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '無法建立貼文，請稍後再試。')
  } finally {
    isSubmitting.value = false
  }
}

onBeforeUnmount(() => {
  revokeImagePreviews()
})
</script>

<template>
  <div class="compose-modal-backdrop" @click.self="closeModal">
    <section
      ref="dialog"
      class="compose-modal"
      role="dialog"
      aria-modal="true"
      aria-labelledby="compose-title"
      tabindex="-1"
    >
      <header class="compose-modal-header">
        <button ref="cancelButton" type="button" class="compose-modal-cancel" @click="closeModal">取消</button>
        <div class="compose-modal-title-group">
          <h2 id="compose-title">建立貼文</h2>
        </div>
        <div class="compose-modal-header-actions">
          <button type="button" class="compose-modal-icon" aria-label="貼文草稿">
            <span class="material-symbols-outlined">article</span>
          </button>
          <button type="button" class="compose-modal-icon" aria-label="更多選項">
            <span>•••</span>
          </button>
        </div>
      </header>

      <div class="compose-modal-body">
        <div class="compose-modal-avatar" aria-hidden="true">
          <span></span>
        </div>

        <div class="compose-modal-content">
          <div class="compose-modal-userline">
            <strong>ynkai0102</strong>
            <div class="compose-visibility-toggle" role="group" aria-label="貼文可見範圍">
              <button
                type="button"
                :class="{ active: visibility === 'public' }"
                :aria-pressed="visibility === 'public'"
                @click="visibility = 'public'"
              >
                公開貼文
              </button>
              <button
                type="button"
                :class="{ active: visibility === 'private' }"
                :aria-pressed="visibility === 'private'"
                @click="visibility = 'private'"
              >
                私人貼文
              </button>
            </div>
          </div>

          <label class="sr-only" for="compose-content">貼文內容</label>
          <textarea
            id="compose-content"
            v-model="content"
            class="compose-modal-textarea"
            placeholder="寫點今天想留下的事..."
            rows="5"
            maxlength="5000"
          ></textarea>

          <div
            v-if="imagePreviews.length"
            class="compose-media-grid compose-modal-media"
            :class="`compose-media-grid-${Math.min(imagePreviews.length, 4)}`"
          >
            <div
              v-for="(image, index) in imagePreviews"
              :key="`${image.name}-${index}`"
              class="compose-media-tile"
            >
              <img :src="image.url" :alt="image.name" width="1200" height="675">
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

          <div class="compose-modal-tools" aria-label="發文工具列">
            <button type="button" aria-label="新增圖片" @click="openImagePicker">
              <span class="material-symbols-outlined">image</span>
            </button>
            <input
              ref="imageInput"
              type="file"
              class="sr-only"
              :accept="acceptedPostImageInput"
              multiple
              aria-label="選擇最多四張貼文圖片"
              @change="handleImageChange"
            >
            <button type="button" aria-label="GIF"><span>GIF</span></button>
            <button type="button" aria-label="表情符號"><span class="material-symbols-outlined">mood</span></button>
            <button type="button" aria-label="清單"><span class="material-symbols-outlined">format_list_bulleted</span></button>
            <button type="button" aria-label="地點"><span class="material-symbols-outlined">location_on</span></button>
          </div>
        </div>
      </div>

      <p v-if="errorMessage" class="compose-modal-error" role="alert" aria-live="assertive">{{ errorMessage }}</p>

      <footer class="compose-modal-footer">
        <button type="button" class="compose-modal-options">
          {{ visibility === 'public' ? '公開貼文' : '私人貼文' }}
        </button>
        <button
          type="button"
          class="compose-modal-submit"
          :disabled="isSubmitting || (!content.trim() && selectedImages.length === 0)"
          @click="submitPost"
        >
          {{ isSubmitting ? '發佈中' : '發佈' }}
        </button>
      </footer>
    </section>
  </div>
</template>
