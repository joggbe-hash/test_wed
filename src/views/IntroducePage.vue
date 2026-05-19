<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { usePageCss } from '../composables/usePageCss'

usePageCss('introduce_page.css')

const router = useRouter()
const backgroundPath = computed(() => window.history.length > 1 ? '/home' : '/home')
</script>

<template>
  <iframe id="bg-iframe" :src="backgroundPath" style="position: absolute; top: 0; left: 0; width: 100%; height: 100%; z-index: 0; border: none; pointer-events: none;"></iframe>

  <div class="modal-backdrop" @click="router.back()"></div>

  <div class="profile-card">
    <div class="profile-cover"></div>

    <div class="profile-avatar-container">
      <div class="profile-avatar"></div>
    </div>

    <div class="profile-info">
      <div class="profile-username">@使用者</div>
      <div class="profile-link">
        <svg viewBox="0 0 24 24" style="width:16px; height:16px; fill:currentColor;"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/></svg>
        haha.tww.com
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

      <div class="expand-arrow" @click="router.back()">
        <svg viewBox="0 0 24 24"><path d="M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6 1.41-1.41z"/></svg>
      </div>
    </div>
  </div>
</template>
