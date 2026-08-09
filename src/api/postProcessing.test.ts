import { describe, expect, it, vi } from 'vitest'
import { waitForPostProcessing } from './postProcessing'

describe('waitForPostProcessing', () => {
  it('returns the terminal image status for the target post', async () => {
    const fetcher = vi.fn().mockResolvedValue({
      posts: [{
        id: 7,
        user_id: 1,
        username: 'owner',
        visibility: 'public',
        image_status: 'ready',
        created_at: '2026-08-04T00:00:00Z',
      }],
      next_cursor: '',
    })

    await expect(waitForPostProcessing(7, {
      fetcher,
      pollDelaysMs: [0],
    })).resolves.toBe('ready')
  })

  it('returns null when the post remains unavailable or processing', async () => {
    const fetcher = vi.fn().mockResolvedValue({ posts: [], next_cursor: '' })

    await expect(waitForPostProcessing(7, {
      fetcher,
      pollDelaysMs: [0, 0],
    })).resolves.toBeNull()
  })
})
