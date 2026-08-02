import { afterEach, describe, expect, it, vi } from 'vitest'
import { createPost, fetchFeed } from './backendApi'
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

  it('sends the selected visibility in JSON post requests', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ message: 'post created', post_id: 1 }),
      { status: 201, headers: { 'Content-Type': 'application/json' } },
    ))
    vi.stubGlobal('fetch', fetchMock)

    await createPost('private note', [], 'private')

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(JSON.parse(init.body as string)).toEqual({ content: 'private note', visibility: 'private' })
  })

  it('sends the selected visibility in multipart post requests', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ message: 'post created', post_id: 1 }),
      { status: 201, headers: { 'Content-Type': 'application/json' } },
    ))
    vi.stubGlobal('fetch', fetchMock)

    await createPost('private image', [new File(['image'], 'image.png', { type: 'image/png' })], 'private')

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    const body = init.body as FormData
    expect(body.get('content')).toBe('private image')
    expect(body.get('visibility')).toBe('private')
  })
})
