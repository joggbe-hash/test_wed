<script setup lang="ts">
import { ref } from 'vue'

const liked = ref(false)
const bookmarked = ref(false)
</script>

<template>
  <div class="post-actions" @mousedown.prevent>
    <div class="post-actions-right">
      <button
        type="button"
        class="post-action-icon"
        :class="{ 'post-action-icon-liked': liked }"
        :aria-pressed="liked"
        aria-label="喜歡"
        @click="liked = !liked"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path>
        </svg>
      </button>

      <button type="button" class="post-action-icon" aria-label="分享">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <line x1="22" y1="2" x2="11" y2="13"></line>
          <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
        </svg>
      </button>

      <button type="button" class="post-action-icon" aria-label="活動">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline>
        </svg>
      </button>

      <button
        type="button"
        class="post-action-icon"
        :class="{ 'post-action-icon-bookmarked': bookmarked }"
        :aria-pressed="bookmarked"
        aria-label="收藏"
        @click="bookmarked = !bookmarked"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"></path>
        </svg>
      </button>
    </div>
  </div>
</template>
