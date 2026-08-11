import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  createPost,
  fetchFeed,
  fetchMyPosts,
  loginAccount,
  logoutAccount,
  registerAccount,
  sendVerificationCode,
  verifyLoginAccount,
} from './backendApi'
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

  it('adds the browser-request header to every API call', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ posts: [], next_cursor: '' }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    ))
    vi.stubGlobal('fetch', fetchMock)

    await fetchFeed()

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(new Headers(init.headers).get('X-Type-WSP-Request')).toBe('1')
  })

  it('returns the verification challenge and sends it during registration', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(
        JSON.stringify({ message: 'verification code sent', challenge_id: 'challenge-1' }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ))
      .mockResolvedValueOnce(new Response(
        JSON.stringify({ message: 'registered', user_id: 1 }),
        { status: 201, headers: { 'Content-Type': 'application/json' } },
      ))
    vi.stubGlobal('fetch', fetchMock)

    const response = await sendVerificationCode('user@example.com')
    expect(response.challenge_id).toBe('challenge-1')

    await registerAccount({
      username: 'user',
      email: 'user@example.com',
      password: 'password1',
      code: '123456',
      challenge_id: response.challenge_id,
    })

    const [, init] = fetchMock.mock.calls[1] as [string, RequestInit]
    expect(JSON.parse(init.body as string)).toMatchObject({ challenge_id: 'challenge-1' })
  })

  it('creates a login challenge before completing the session', async () => {
	const fetchMock = vi.fn()
	  .mockResolvedValueOnce(new Response(
		JSON.stringify({
		  message: 'login verification code sent',
		  challenge_id: 'login-challenge-1',
		  requires_verification: true,
		  expires_in_seconds: 300,
		}),
		{ status: 202, headers: { 'Content-Type': 'application/json' } },
	  ))
	  .mockResolvedValueOnce(new Response(
		JSON.stringify({ user: { id: 1, username: 'tester' } }),
		{ status: 200, headers: { 'Content-Type': 'application/json' } },
	  ))
	vi.stubGlobal('fetch', fetchMock)

	const challenge = await loginAccount('user@example.com', 'Password1')
	expect(challenge).toMatchObject({
	  challenge_id: 'login-challenge-1',
	  requires_verification: true,
	})

	await verifyLoginAccount({
	  email: 'user@example.com',
	  code: '123456',
	  challenge_id: challenge.challenge_id,
	  remember: true,
	})

	expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/auth/login')
	expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/auth/login/verify')
	const [, verifyInit] = fetchMock.mock.calls[1] as [string, RequestInit]
	expect(JSON.parse(verifyInit.body as string)).toEqual({
	  email: 'user@example.com',
	  code: '123456',
	  challenge_id: 'login-challenge-1',
	  remember: true,
	})
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

  it('loads only the current user posts from the personal endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ posts: [], next_cursor: '' }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    ))
    vi.stubGlobal('fetch', fetchMock)

    await fetchMyPosts('next page')

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/posts/me?cursor=next%20page',
      expect.objectContaining({ credentials: 'include' }),
    )
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

  it('requests logout of only the current browser session', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ message: 'logged out' }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    ))
    vi.stubGlobal('fetch', fetchMock)

    await logoutAccount()

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/logout',
      expect.objectContaining({ method: 'POST', credentials: 'include' }),
    )
  })
})
