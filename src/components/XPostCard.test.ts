import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { BackendPost } from '../api/backendApi'
import XPostCard from './XPostCard.vue'

const post: BackendPost = {
  id: 9,
  user_id: 2,
  username: 'other-user',
  visibility: 'public',
  content: '別人的貼文',
  image_status: 'none',
  created_at: '2026-08-04T12:00:00Z',
}

describe('XPostCard delete permissions', () => {
  it('does not offer post settings or deletion to non-owners', () => {
    const wrapper = mount(XPostCard, {
      props: { post, isMenuOpen: false, canDelete: false },
      global: {
        stubs: {
          PostActions: true,
          TaskDeleteConfirmDialog: true,
        },
      },
    })

    expect(wrapper.find('[aria-label="貼文設定"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('刪除貼文')
  })

  it('offers deletion to the post owner', () => {
    const wrapper = mount(XPostCard, {
      props: { post, isMenuOpen: true, canDelete: true },
      global: {
        stubs: {
          PostActions: true,
          TaskDeleteConfirmDialog: true,
        },
      },
    })

    expect(wrapper.get('[aria-label="貼文設定"]').attributes('aria-label')).toBe('貼文設定')
    expect(wrapper.get('.x-post-delete').text()).toBe('刪除貼文')
  })
})
