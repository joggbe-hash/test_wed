import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const api = vi.hoisted(() => ({
  fetchFeed: vi.fn(),
}))

vi.mock('../api/backendApi', () => ({
  fetchFeed: api.fetchFeed,
}))

import { useFeedStore } from './useFeedStore'

describe('useFeedStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    api.fetchFeed.mockReset()
  })

  it('loads once, supports forced refresh, and resets state', async () => {
    api.fetchFeed
      .mockResolvedValueOnce({
        posts: [{ id: 1, username: 'first', user_id: 1, image_status: 'none', created_at: '' }],
        next_cursor: '',
      })
      .mockResolvedValueOnce({
        posts: [{ id: 2, username: 'second', user_id: 2, image_status: 'none', created_at: '' }],
        next_cursor: '',
      })
    const store = useFeedStore()

    await store.loadPosts()
    await store.loadPosts()
    expect(api.fetchFeed).toHaveBeenCalledTimes(1)
    expect(store.posts.map((post) => post.id)).toEqual([1])

    await store.loadPosts(true)
    expect(store.posts.map((post) => post.id)).toEqual([2])

    store.reset()
    expect(store.posts).toEqual([])
    expect(store.isLoaded).toBe(false)
    expect(store.nextCursor).toBe('')
    expect(store.hasMore).toBe(false)
  })

  it('loads the next cursor and removes duplicate posts', async () => {
    api.fetchFeed
      .mockResolvedValueOnce({
        posts: [{ id: 1, username: 'first', user_id: 1, visibility: 'public', image_status: 'none', created_at: '' }],
        next_cursor: 'page-2',
      })
      .mockResolvedValueOnce({
        posts: [
          { id: 1, username: 'first', user_id: 1, visibility: 'public', image_status: 'none', created_at: '' },
          { id: 2, username: 'second', user_id: 2, visibility: 'public', image_status: 'none', created_at: '' },
        ],
        next_cursor: '',
      })
    const store = useFeedStore()

    await store.loadPosts()
    expect(store.hasMore).toBe(true)

    await store.loadMore()

    expect(api.fetchFeed).toHaveBeenLastCalledWith('page-2')
    expect(store.posts.map((post) => post.id)).toEqual([1, 2])
    expect(store.hasMore).toBe(false)
  })

  it('does not apply a response that finishes after account-scoped state is reset', async () => {
    let resolveFeed!: (value: {
      posts: Array<{
        id: number
        username: string
        user_id: number
        visibility: 'private'
        image_status: 'none'
        created_at: string
      }>
      next_cursor: string
    }) => void
    api.fetchFeed.mockReturnValue(new Promise((resolve) => {
      resolveFeed = resolve
    }))
    const store = useFeedStore()

    const pendingLoad = store.loadPosts()
    store.reset()
    resolveFeed({
      posts: [{
        id: 99,
        username: 'previous-account',
        user_id: 1,
        visibility: 'private',
        image_status: 'none',
        created_at: '',
      }],
      next_cursor: 'stale-page',
    })
    await pendingLoad

    expect(store.posts).toEqual([])
    expect(store.isLoaded).toBe(false)
    expect(store.nextCursor).toBe('')
  })
})
