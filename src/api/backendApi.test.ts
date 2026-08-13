import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  createPost,
  fetchFeed,
  fetchMyPosts,
  loginAccount,
  logoutAccount,
  registerAccount,
  sendVerificationCode,
  verifyLoginOwnership,
  verifyLoginAccount,
} from './backendApi'
import { setUnauthorizedHandler } from './unauthorizedHandler'

describe('backendApi', () => {
  const canonicalGrant = 'A'.repeat(43)

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
	if (!('challenge_id' in challenge)) throw new Error('Expected login verification challenge')

	const controller = new AbortController()
	await verifyLoginAccount({
	  email: 'user@example.com',
	  code: '123456',
	  challenge_id: challenge.challenge_id,
	  remember: true,
	}, controller.signal)

	expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/auth/login')
	expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/auth/login/verify')
	const [, verifyInit] = fetchMock.mock.calls[1] as [string, RequestInit]
	expect(JSON.parse(verifyInit.body as string)).toEqual({
	  email: 'user@example.com',
	  code: '123456',
	  challenge_id: 'login-challenge-1',
	  remember: true,
	})
	expect(verifyInit.signal).toBe(controller.signal)
  })

  it.each([
    ['missing user', {}, 200],
    ['null user', { user: null }, 200],
    ['array user', { user: [] }, 200],
    ['zero id', { user: { id: 0, username: 'tester' } }, 200],
    ['fractional id', { user: { id: 1.5, username: 'tester' } }, 200],
    ['empty username', { user: { id: 1, username: '' } }, 200],
    ['oversized username', { user: { id: 1, username: 'a'.repeat(21) } }, 200],
    ['unexpected success status', { user: { id: 1, username: 'tester' } }, 201],
  ])('rejects a malformed login verification response: %s', async (_case, body, status) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify(body),
      { status, headers: { 'Content-Type': 'application/json' } },
    )))

    await expect(verifyLoginAccount({
      email: 'user@example.com',
      code: '123456',
      challenge_id: 'login-challenge-1',
      remember: false,
    })).rejects.toMatchObject({ status })
  })

  it('preserves a validated ownership challenge and submits its grant on retry', async () => {
    const ownershipChallenge = {
      challenge_id: 'ownership-challenge-1',
      code_format: 'base32-16-v1',
      expires_in_seconds: 86400,
    }
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(
        JSON.stringify({
          error: 'email ownership required',
          code: 'LOGIN_EMAIL_OWNERSHIP_REQUIRED',
          ownership_challenge: ownershipChallenge,
        }),
        { status: 429, headers: { 'Content-Type': 'application/json' } },
      ))
      .mockResolvedValueOnce(new Response(
        JSON.stringify({
          password_verification_grant: canonicalGrant,
          expires_in_seconds: 300,
          max_attempts: 3,
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ))
      .mockResolvedValueOnce(new Response(
        JSON.stringify({
          message: 'login verification code sent',
          challenge_id: 'login-challenge-2',
          requires_verification: true,
          expires_in_seconds: 300,
        }),
        { status: 202, headers: { 'Content-Type': 'application/json' } },
      ))
    vi.stubGlobal('fetch', fetchMock)

    await expect(loginAccount('user@example.com', 'Password1')).rejects.toMatchObject({
      status: 429,
      code: 'LOGIN_EMAIL_OWNERSHIP_REQUIRED',
      ownershipChallenge,
    })

    const grant = await verifyLoginOwnership({
      email: 'user@example.com',
      challenge_id: ownershipChallenge.challenge_id,
      code: 'ABCDEFGHJKLMNPQ2',
    })
    expect(grant).toEqual({
      password_verification_grant: canonicalGrant,
      expires_in_seconds: 300,
      max_attempts: 3,
    })
    expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/auth/login/ownership/verify')
    const [, ownershipInit] = fetchMock.mock.calls[1] as [string, RequestInit]
    expect(JSON.parse(ownershipInit.body as string)).toEqual({
      email: 'user@example.com',
      challenge_id: 'ownership-challenge-1',
      code: 'ABCDEFGHJKLMNPQ2',
    })

    await loginAccount('user@example.com', 'Password1', true, {
      passwordVerificationGrant: grant.password_verification_grant,
    })
    const [, retryInit] = fetchMock.mock.calls[2] as [string, RequestInit]
    expect(JSON.parse(retryInit.body as string)).toEqual({
      email: 'user@example.com',
      password: 'Password1',
      remember: true,
      password_verification_grant: canonicalGrant,
    })
  })

  it('does not expose malformed ownership metadata through ApiError', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({
        error: 'email ownership required',
        code: 'LOGIN_EMAIL_OWNERSHIP_REQUIRED',
        ownership_challenge: {
          challenge_id: '',
          code_format: 'base32',
          expires_in_seconds: 0,
        },
      }),
      { status: 429, headers: { 'Content-Type': 'application/json' } },
    )))

    await expect(loginAccount('user@example.com', 'Password1')).rejects.toMatchObject({
      status: 429,
      code: undefined,
      ownershipChallenge: undefined,
    })
  })

  it('rejects a malformed ownership grant response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({
        password_verification_grant: 'not-a-canonical-token',
        expires_in_seconds: '300',
        max_attempts: 0,
      }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    )))

    await expect(verifyLoginOwnership({
      email: 'user@example.com',
      challenge_id: 'ownership-challenge-1',
      code: 'ABCDEFGHJKLMNPQ2',
    })).rejects.toMatchObject({ status: 200 })
  })

  it('rejects a direct user response from the challenge-only login endpoint', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ user: { id: 7, username: 'tester' } }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    )))

    await expect(loginAccount('user@example.com', 'Password1', false, {
      passwordVerificationGrant: canonicalGrant,
    })).rejects.toMatchObject({ status: 200 })
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
