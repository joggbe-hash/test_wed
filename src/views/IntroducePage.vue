<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import LoadingPanel from '../components/LoadingPanel.vue'
import { fetchProfilePreview } from '../api/timedApi'
import { showIntroduceModal } from '../composables/useModal'

const props = defineProps({
  overlay: {
    type: Boolean,
    default: false,
  },
})

const router = useRouter()
const isLoading = ref(true)
const profile = ref({ username: '', link: '' })

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

onMounted(async () => {
  if (props.overlay) {
    document.body.style.overflow = 'hidden'
  }
  isLoading.value = true
  const response = await fetchProfilePreview()
  profile.value = response.data
  isLoading.value = false
})

onBeforeUnmount(() => {
  if (props.overlay) {
    document.body.style.overflow = ''
  }
})
</script>

<template>
  <div class="introduce-container">

    <div class="modal-backdrop" @click="closePreview"></div>

    <LoadingPanel v-if="isLoading" class="relative z-10 scale-150" />

    <div v-else class="profile-card">
      <div class="profile-cover"></div>

      <div class="profile-avatar-container">
        <div class="profile-avatar">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg>
        </div>
      </div>

      <div class="profile-info">
        <div class="profile-username">{{ profile.username }}</div>
        <div class="profile-link">
          <svg class="size-4 fill-current" viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/></svg>
          {{ profile.link }}
        </div>

        <div class="profile-actions">
          <button class="action-btn">
            <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"></path><polyline points="22,6 12,13 2,6"></polyline></svg>
          </button>
          <button class="action-btn">
            <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><line x1="19" y1="8" x2="19" y2="14"></line><line x1="16" y1="11" x2="22" y2="11"></line></svg>
          </button>
          <button class="action-btn">
            <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="1"></circle><circle cx="19" cy="12" r="1"></circle><circle cx="5" cy="12" r="1"></circle></svg>
          </button>
        </div>

        <div class="expand-arrow" @click="closePreview">
          <svg viewBox="0 0 24 24"><path d="M7.41 8.59 12 13.17l4.59-4.58L18 10l-6 6-6-6z"/></svg>
        </div>
      </div>
    </div>
  </div>
</template>
