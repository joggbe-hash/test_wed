import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchFeed } from '../api/backendApi'
import type { BackendPost } from '../api/backendApi'

export const useFeedStore = defineStore('feed', () => {
  const posts = ref<BackendPost[]>([])
  const isLoaded = ref(false)
  const isLoading = ref(false)

  async function loadPosts(forceRefresh = false) {
    if (isLoaded.value && !forceRefresh) {
      return
    }

    isLoading.value = true
    try {
      const response = await fetchFeed()
      posts.value = response.posts
      isLoaded.value = true
    } finally {
      isLoading.value = false
    }
  }

  function prependPost(post: BackendPost) {
    posts.value.unshift(post)
  }

  function removePost(postId: number) {
    posts.value = posts.value.filter((post) => post.id !== postId)
  }

  return {
    posts,
    isLoaded,
    isLoading,
    loadPosts,
    prependPost,
    removePost,
  }
})
