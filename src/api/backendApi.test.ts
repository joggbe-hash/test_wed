import { afterEach, describe, expect, it, vi } from 'vitest'
import { fetchFeed } from './backendApi'
import { setUnauthorizedHandler } from './unauthorizedHandler'

describe('backendApi', () => {
  afterEach(() => {
    setUnauthorizedHandler(null)
    vi.unstubAllGlobals()
  })

  it('returns typed JSON responses', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ posts: [], next_cursor: '' }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    )))

    await expect(fetchFeed()).resolves.toEqual({ posts: [], next_cursor: '' })
  })

  it('notifies the application boundary for unauthorized requests', async () => {
    const unauthorized = vi.fn()
    setUnauthorizedHandler(unauthorized)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ error: 'unauthorized' }),
      { status: 401, headers: { 'Content-Type': 'application/json' } },
    )))

    await expect(fetchFeed()).rejects.toMatchObject({ status: 401 })
    expect(unauthorized).toHaveBeenCalledOnce()
  })
})
