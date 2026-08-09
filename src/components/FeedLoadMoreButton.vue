<script setup lang="ts">
defineProps<{
  hasMore: boolean
  isLoading: boolean
}>()

const emit = defineEmits<{
  loadMore: []
}>()
</script>

<template>
  <button
    v-if="hasMore"
    type="button"
    class="feed-load-more"
    :disabled="isLoading"
    @click="emit('loadMore')"
  >
    {{ isLoading ? '載入中…' : '載入更多' }}
  </button>
</template>
