import { computed, ref, shallowReadonly, shallowRef } from 'vue'
import { fetchMyPosts, type BackendPost } from '../../api/backendApi'

export function usePersonalPosts() {
  const posts = ref<BackendPost[]>([])
  const isLoading = shallowRef(false)
  const isLoadingMore = shallowRef(false)
  const nextCursor = shallowRef('')
  const hasMore = computed(() => nextCursor.value.length > 0)

  async function loadPosts() {
    isLoading.value = true
    try {
      const response = await fetchMyPosts()
      posts.value = response.posts
      nextCursor.value = response.next_cursor
    } finally {
      isLoading.value = false
    }
  }

  async function loadMore() {
    if (!hasMore.value || isLoading.value || isLoadingMore.value) return

    isLoadingMore.value = true
    try {
      const response = await fetchMyPosts(nextCursor.value)
      const knownPostIds = new Set(posts.value.map((post) => post.id))
      posts.value = [
        ...posts.value,
        ...response.posts.filter((post) => !knownPostIds.has(post.id)),
      ]
      nextCursor.value = response.next_cursor
    } finally {
      isLoadingMore.value = false
    }
  }

  function removePost(postId: number) {
    posts.value = posts.value.filter((post) => post.id !== postId)
  }

  return {
    posts: shallowReadonly(posts),
    isLoading: shallowReadonly(isLoading),
    isLoadingMore: shallowReadonly(isLoadingMore),
    hasMore,
    loadPosts,
    loadMore,
    removePost,
  }
}
