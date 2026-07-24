<script setup lang="ts">
import { computed, useTemplateRef } from 'vue'
import { useAccessibleDialog } from '../composables/useAccessibleDialog'

const props = defineProps<{
  urls: string[]
}>()

const emit = defineEmits<{
  close: []
}>()

const index = defineModel<number>('index', { required: true })
const dialog = useTemplateRef<HTMLElement>('dialog')
const closeButton = useTemplateRef<HTMLButtonElement>('closeButton')
const currentUrl = computed(() => props.urls[index.value] ?? '')
const imageLabel = computed(() => `貼文圖片 ${index.value + 1}／${props.urls.length}`)

function close() {
  emit('close')
}

function previous() {
  if (props.urls.length <= 1) return
  index.value = index.value === 0 ? props.urls.length - 1 : index.value - 1
}

function next() {
  if (props.urls.length <= 1) return
  index.value = (index.value + 1) % props.urls.length
}

function handleArrowKeys(event: KeyboardEvent) {
  if (event.key === 'ArrowLeft') {
    event.preventDefault()
    previous()
  } else if (event.key === 'ArrowRight') {
    event.preventDefault()
    next()
  }
}

useAccessibleDialog({
  dialog,
  initialFocus: closeButton,
  onClose: close,
  backgroundSelector: '[data-app-route-content]',
})
</script>

<template>
  <Teleport to="body">
    <div class="image-viewer-backdrop" @click.self="close">
      <section
        ref="dialog"
        class="image-viewer-dialog"
        role="dialog"
        aria-modal="true"
        aria-label="圖片檢視器"
        tabindex="-1"
        @click.self="close"
        @keydown="handleArrowKeys"
      >
        <button
          ref="closeButton"
          type="button"
          class="image-viewer-close"
          aria-label="關閉圖片檢視器"
          @click="close"
        >
          <span aria-hidden="true">×</span>
        </button>

        <button
          v-if="urls.length > 1"
          type="button"
          class="image-viewer-previous"
          aria-label="上一張圖片"
          @click="previous"
        >
          <span aria-hidden="true">‹</span>
        </button>

        <figure class="image-viewer-figure">
          <img
            :src="currentUrl"
            :alt="imageLabel"
            width="1600"
            height="900"
            class="image-viewer-image"
          >
          <figcaption class="sr-only">{{ imageLabel }}</figcaption>
        </figure>

        <button
          v-if="urls.length > 1"
          type="button"
          class="image-viewer-next"
          aria-label="下一張圖片"
          @click="next"
        >
          <span aria-hidden="true">›</span>
        </button>

        <p class="image-viewer-counter tabular-nums" aria-live="polite">
          {{ index + 1 }}／{{ urls.length }}
        </p>
      </section>
    </div>
  </Teleport>
</template>

<style scoped>
@reference "../style.css";

.image-viewer-backdrop {
  @apply fixed inset-0 z-[300] flex items-center justify-center bg-black/90 p-4;
  padding:
    max(1rem, env(safe-area-inset-top))
    max(1rem, env(safe-area-inset-right))
    max(1rem, env(safe-area-inset-bottom))
    max(1rem, env(safe-area-inset-left));
}

.image-viewer-dialog {
  @apply relative flex size-full items-center justify-center overscroll-contain focus-visible:outline focus-visible:outline-2 focus-visible:outline-white;
}

.image-viewer-close,
.image-viewer-previous,
.image-viewer-next {
  @apply absolute z-10 flex size-12 items-center justify-center rounded-full bg-black/70 text-3xl text-white transition-colors hover:bg-white/20 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white;
}

.image-viewer-close { @apply left-1 top-1; }
.image-viewer-previous { @apply left-1 top-1/2 -translate-y-1/2; }
.image-viewer-next { @apply right-1 top-1/2 -translate-y-1/2; }

.image-viewer-figure {
  @apply flex max-h-full max-w-full items-center justify-center;
}

.image-viewer-image {
  @apply max-h-[82dvh] max-w-[92vw] object-contain;
  width: auto;
  height: auto;
}

.image-viewer-counter {
  @apply absolute bottom-1 left-1/2 -translate-x-1/2 rounded-full bg-black/70 px-4 py-2 text-sm font-bold text-white;
}
</style>
