import { defineStore } from 'pinia'
import { computed, ref, shallowRef } from 'vue'
import { fetchFeed } from '../api/backendApi'
import { waitForPostProcessing } from '../api/postProcessing'
import type { BackendPost } from '../api/backendApi'

export const useFeedStore = defineStore('feed', () => {
  const posts = ref<BackendPost[]>([])
  const isLoaded = shallowRef(false)
  const isLoading = shallowRef(false)
  const isLoadingMore = shallowRef(false)
  const nextCursor = shallowRef('')
  const hasMore = computed(() => nextCursor.value.length > 0)
  const processingPolls = new Map<number, AbortController>()
  let requestGeneration = 0

  function applyFirstPage(response: Awaited<ReturnType<typeof fetchFeed>>) {
    posts.value = response.posts
    nextCursor.value = response.next_cursor
    isLoaded.value = true
  }

  async function loadPosts(forceRefresh = false) {
    if (isLoaded.value && !forceRefresh) {
      return
    }

    const generation = requestGeneration
    isLoading.value = true
    try {
      const response = await fetchFeed()
      if (generation !== requestGeneration) return
      applyFirstPage(response)
    } finally {
      if (generation === requestGeneration) isLoading.value = false
    }
  }

  async function loadMore() {
    if (!isLoaded.value || !hasMore.value || isLoading.value || isLoadingMore.value) return

    const generation = requestGeneration
    isLoadingMore.value = true
    try {
      const response = await fetchFeed(nextCursor.value)
      if (generation !== requestGeneration) return
      const knownPostIds = new Set(posts.value.map((post) => post.id))
      posts.value = [
        ...posts.value,
        ...response.posts.filter((post) => !knownPostIds.has(post.id)),
      ]
      nextCursor.value = response.next_cursor
    } finally {
      if (generation === requestGeneration) isLoadingMore.value = false
    }
  }

  async function trackPostProcessing(postId: number) {
    processingPolls.get(postId)?.abort()
    const controller = new AbortController()
    processingPolls.set(postId, controller)
    const generation = requestGeneration

    try {
      const status = await waitForPostProcessing(postId, { signal: controller.signal })
      if (!status || generation !== requestGeneration) return
      const response = await fetchFeed()
      if (generation === requestGeneration) applyFirstPage(response)
    } catch (error) {
      if (!(error instanceof DOMException && error.name === 'AbortError')) {
        console.error('Unable to refresh image processing status', error)
      }
    } finally {
      if (processingPolls.get(postId) === controller) processingPolls.delete(postId)
    }
  }

  function prependPost(post: BackendPost) {
    posts.value.unshift(post)
  }

  function removePost(postId: number) {
    posts.value = posts.value.filter((post) => post.id !== postId)
  }

  function reset() {
    requestGeneration += 1
    processingPolls.forEach((controller) => controller.abort())
    processingPolls.clear()
    posts.value = []
    isLoaded.value = false
    isLoading.value = false
    isLoadingMore.value = false
    nextCursor.value = ''
  }

  return {
    posts,
    isLoaded,
    isLoading,
    isLoadingMore,
    nextCursor,
    hasMore,
    loadPosts,
    loadMore,
    trackPostProcessing,
    prependPost,
    removePost,
    reset,
  }
})
