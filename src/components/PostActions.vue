<script setup lang="ts">
import { ref } from 'vue'
import { ClosedEye, OpenEye } from './Icons'

const eyeClosed = ref(false)
const liked = ref(false)
const bookmarked = ref(false)
</script>

<template>
  <div class="flex items-center justify-between px-1.5 py-3 text-muted">
    <div class="flex gap-[15px]">
      <div class="flex cursor-pointer items-center" @click="eyeClosed = !eyeClosed">
        <ClosedEye v-if="eyeClosed" />
        <OpenEye v-else />
      </div>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg>
    </div>
    <div class="flex gap-[15px]">
      <svg
        class="cursor-pointer"
        width="20"
        height="20"
        viewBox="0 0 24 24"
        :fill="liked ? '#e74c3c' : 'none'"
        :stroke="liked ? '#e74c3c' : 'currentColor'"
        stroke-width="2"
        @click="liked = !liked"
      ><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path></svg>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="22" y1="2" x2="11" y2="13"></line><polygon points="22 2 15 22 11 13 2 9 22 2"></polygon></svg>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline></svg>
      <svg
        class="cursor-pointer"
        width="20"
        height="20"
        viewBox="0 0 24 24"
        :fill="bookmarked ? '#4A3320' : 'none'"
        :stroke="bookmarked ? '#4A3320' : 'currentColor'"
        stroke-width="2"
        @click="bookmarked = !bookmarked"
      ><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"></path></svg>
    </div>
  </div>
</template>
