<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, useTemplateRef } from 'vue'
import { useRouter } from 'vue-router'
import { showIntroduceModal } from '../composables/useModal'
import { useAccessibleDialog } from '../composables/useAccessibleDialog'
import { useSession } from '../composables/useSession'

const props = defineProps({
  overlay: {
    type: Boolean,
    default: false,
  },
})

const router = useRouter()
const { currentUser } = useSession()
const profileUsername = computed(() => currentUser.value ? `@${currentUser.value.username}` : '')
const hasProfile = computed(() => currentUser.value !== null)
const dialog = useTemplateRef<HTMLElement>('dialog')
const closeButton = useTemplateRef<HTMLButtonElement>('closeButton')

function closePreview() {
  if (props.overlay) {
    showIntroduceModal.value = false
    return
  }

  if (window.history.length > 1) {
    router.back()
    return
  }

  router.push('/home')
}

useAccessibleDialog({
  dialog,
  initialFocus: closeButton,
  onClose: closePreview,
  backgroundSelector: props.overlay ? '[data-app-route-content]' : undefined,
})

onMounted(() => {
  if (props.overlay) {
    document.body.style.overflow = 'hidden'
  }
})

onBeforeUnmount(() => {
  if (props.overlay) {
    document.body.style.overflow = ''
  }
})
</script>

<template>
  <div class="introduce-container">

    <div class="modal-backdrop" aria-hidden="true" @click="closePreview"></div>

    <section
      ref="dialog"
      class="profile-card"
      role="dialog"
      aria-modal="true"
      aria-labelledby="profile-preview-title"
      tabindex="-1"
    >
      <div v-if="!hasProfile" class="m-6 rounded-xl border border-red-300 bg-red-50 p-5 text-red-900" role="alert">
        <p>目前沒有可顯示的登入資料。</p>
        <button ref="closeButton" type="button" class="mt-4 rounded-lg border border-red-500 px-4 py-2 font-bold" @click="closePreview">
          關閉
        </button>
      </div>

      <template v-else>
      <div class="profile-cover"></div>

      <div class="profile-avatar-container">
        <div class="profile-avatar">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg>
        </div>
      </div>

      <div class="profile-info">
        <h2 id="profile-preview-title" class="profile-username">{{ profileUsername }}</h2>

        <div class="profile-actions">
          <button type="button" class="action-btn" aria-label="傳送訊息">
            <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"></path><polyline points="22,6 12,13 2,6"></polyline></svg>
          </button>
          <button type="button" class="action-btn" aria-label="追蹤使用者">
            <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><line x1="19" y1="8" x2="19" y2="14"></line><line x1="16" y1="11" x2="22" y2="11"></line></svg>
          </button>
          <button type="button" class="action-btn" aria-label="更多選項">
            <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="1"></circle><circle cx="19" cy="12" r="1"></circle><circle cx="5" cy="12" r="1"></circle></svg>
          </button>
        </div>

        <button ref="closeButton" type="button" class="expand-arrow" aria-label="關閉個人資料預覽" @click="closePreview">
          <svg viewBox="0 0 24 24"><path d="M7.41 8.59 12 13.17l4.59-4.58L18 10l-6 6-6-6z"/></svg>
        </button>
      </div>
      </template>
    </section>
  </div>
</template>
