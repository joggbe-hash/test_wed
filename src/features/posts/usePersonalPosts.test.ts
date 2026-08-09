import { beforeEach, describe, expect, it, vi } from 'vitest'

const dependencies = vi.hoisted(() => ({
  fetchMyPosts: vi.fn(),
}))

vi.mock('../../api/backendApi', () => ({
  fetchMyPosts: dependencies.fetchMyPosts,
}))

import { usePersonalPosts } from './usePersonalPosts'

describe('usePersonalPosts', () => {
  beforeEach(() => {
    dependencies.fetchMyPosts.mockReset()
  })

  it('loads and paginates only responses from the personal posts endpoint', async () => {
    dependencies.fetchMyPosts
      .mockResolvedValueOnce({
        posts: [{ id: 1, user_id: 7, username: 'owner', visibility: 'public', image_status: 'none', created_at: '' }],
        next_cursor: 'page-2',
      })
      .mockResolvedValueOnce({
        posts: [
          { id: 1, user_id: 7, username: 'owner', visibility: 'public', image_status: 'none', created_at: '' },
          { id: 2, user_id: 7, username: 'owner', visibility: 'private', image_status: 'none', created_at: '' },
        ],
        next_cursor: '',
      })

    const personalPosts = usePersonalPosts()
    await personalPosts.loadPosts()
    await personalPosts.loadMore()

    expect(dependencies.fetchMyPosts).toHaveBeenNthCalledWith(1)
    expect(dependencies.fetchMyPosts).toHaveBeenNthCalledWith(2, 'page-2')
    expect(personalPosts.posts.value.map((post) => post.id)).toEqual([1, 2])
    expect(personalPosts.hasMore.value).toBe(false)
  })
})
