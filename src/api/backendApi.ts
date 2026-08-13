import { notifyUnauthorized } from './unauthorizedHandler'
import type {
  ApiMessageResponse,
  CreatePostResponse,
  CurrentSessionResponse,
  FeedResponse,
  LoginChallengeResponse,
  LoginOwnershipChallenge,
  LoginOwnershipGrantResponse,
  LoginOwnershipVerificationRequest,
  LoginResponse,
  LoginVerificationRequest,
  PostVisibility,
  RegisterAccountRequest,
  SendVerificationCodeResponse,
} from './contracts'
import type { StoredSchedule } from '../features/schedule/types'

export type {
  BackendPost,
  FeedResponse,
  ImageStatus,
  PostVisibility,
  User,
} from './contracts'

export class ApiError extends Error {
  status: number
  code: string | undefined
  ownershipChallenge: LoginOwnershipChallenge | undefined

  constructor(
    status: number,
    message: string,
    details: {
      code?: string
      ownershipChallenge?: LoginOwnershipChallenge
    } = {},
  ) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = details.code
    this.ownershipChallenge = details.ownershipChallenge
  }
}

const loginOwnershipRequiredCode = 'LOGIN_EMAIL_OWNERSHIP_REQUIRED'
const passwordVerificationGrantPattern = /^[A-Za-z0-9_-]{43}$/
const maximumUsernameCodePoints = 20

function isIntegerBetween(value: unknown, minimum: number, maximum: number): value is number {
  return typeof value === 'number' && Number.isInteger(value) && value >= minimum && value <= maximum
}

function isBoundedString(value: unknown, maximumLength: number): value is string {
  return typeof value === 'string' && value.length > 0 && value.length <= maximumLength
}

function isBoundedUsername(value: unknown): value is string {
  return typeof value === 'string' && value.length > 0 && [...value].length <= maximumUsernameCodePoints
}

function loginOwnershipErrorDetails(status: number, data: unknown) {
  if (status !== 429 || !isRecord(data) || data.code !== loginOwnershipRequiredCode) return {}
  const challenge = data.ownership_challenge
  if (!isRecord(challenge)) return {}
  if (
    !isBoundedString(challenge.challenge_id, 128) ||
    challenge.code_format !== 'base32-16-v1' ||
    !isIntegerBetween(challenge.expires_in_seconds, 1, 86400)
  ) return {}

  return {
    code: loginOwnershipRequiredCode,
    ownershipChallenge: {
      challenge_id: challenge.challenge_id,
      code_format: challenge.code_format,
      expires_in_seconds: challenge.expires_in_seconds,
    } satisfies LoginOwnershipChallenge,
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? ''

interface ApiFetchOptions<T> {
  redirectOnUnauthorized?: boolean
  parse?: (data: unknown, status: number) => T
}

// 前端所有後端請求都經過這裡，統一處理身分憑證、資料標頭、錯誤格式和未登入導回。
async function apiFetch<T>(
  path: string,
  init: RequestInit = {},
  options: ApiFetchOptions<T> = {},
): Promise<T> {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    ...init,
    credentials: 'include',
    headers: {
      ...(init.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }),
      ...init.headers,
      'X-Type-WSP-Request': '1',
    },
  })

  const text = await response.text()
  const contentType = response.headers.get('Content-Type') ?? ''
  const isJson = contentType.includes('application/json')
  const data = text && isJson ? JSON.parse(text) : null

  if (!response.ok) {
    if (response.status === 401 && options.redirectOnUnauthorized !== false) {
      void notifyUnauthorized()
    }

    throw new ApiError(
      response.status,
      data?.error ?? data?.message ?? text.slice(0, 120) ?? 'API request failed',
      loginOwnershipErrorDetails(response.status, data),
    )
  }

  if (!isJson) {
    throw new ApiError(
      response.status,
      `Expected JSON from ${path}, but received ${contentType || 'unknown content type'}`,
    )
  }

  return options.parse ? options.parse(data, response.status) : data as T
}

function parseLoginChallenge(data: unknown, status: number): LoginChallengeResponse {
  if (!isRecord(data)) throw new ApiError(status, 'Invalid login response')
  if (
    status === 202 &&
    isBoundedString(data.message, 500) &&
    isBoundedString(data.challenge_id, 128) &&
    data.requires_verification === true &&
    isIntegerBetween(data.expires_in_seconds, 1, 3600)
  ) {
    return {
      message: data.message,
      challenge_id: data.challenge_id,
      requires_verification: true,
      expires_in_seconds: data.expires_in_seconds,
    } satisfies LoginChallengeResponse
  }
  throw new ApiError(status, 'Invalid login response')
}

function parseLoginOwnershipGrant(data: unknown, status: number): LoginOwnershipGrantResponse {
  if (
    status !== 200 ||
    !isRecord(data) ||
    typeof data.password_verification_grant !== 'string' ||
    !passwordVerificationGrantPattern.test(data.password_verification_grant) ||
    !isIntegerBetween(data.expires_in_seconds, 1, 300) ||
    !isIntegerBetween(data.max_attempts, 1, 3)
  ) throw new ApiError(status, 'Invalid login ownership response')

  return {
    password_verification_grant: data.password_verification_grant,
    expires_in_seconds: data.expires_in_seconds,
    max_attempts: data.max_attempts,
  }
}

function parseLoginResponse(data: unknown, status: number): LoginResponse {
  const user = isRecord(data) ? data.user : undefined
  if (
    status !== 200 ||
    !isRecord(user) ||
    !isIntegerBetween(user.id, 1, Number.MAX_SAFE_INTEGER) ||
    !isBoundedUsername(user.username)
  ) throw new ApiError(status, 'Invalid login response')

  return {
    user: {
      id: user.id,
      username: user.username,
    },
  }
}

export function sendVerificationCode(email: string) {
  return apiFetch<SendVerificationCodeResponse>('/api/auth/send-code', {
    method: 'POST',
    body: JSON.stringify({ email }),
  })
}

export function registerAccount(payload: RegisterAccountRequest) {
  return apiFetch<{ message: string; user_id: number }>('/api/auth/register', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

interface LoginAccountOptions {
  passwordVerificationGrant?: string
  signal?: AbortSignal
}

export function loginAccount(
  email: string,
  password: string,
  remember = false,
  options: LoginAccountOptions = {},
) {
  const { passwordVerificationGrant, signal } = options
  return apiFetch<LoginChallengeResponse>(
    '/api/auth/login',
    {
      method: 'POST',
      body: JSON.stringify({
        email,
        password,
        remember,
        ...(passwordVerificationGrant ? { password_verification_grant: passwordVerificationGrant } : {}),
      }),
      signal,
    },
    { redirectOnUnauthorized: false, parse: parseLoginChallenge },
  )
}

export function verifyLoginOwnership(
  payload: LoginOwnershipVerificationRequest,
  signal?: AbortSignal,
) {
  return apiFetch<LoginOwnershipGrantResponse>(
    '/api/auth/login/ownership/verify',
    {
      method: 'POST',
      body: JSON.stringify(payload),
      signal,
    },
    { redirectOnUnauthorized: false, parse: parseLoginOwnershipGrant },
  )
}

export function verifyLoginAccount(
  payload: LoginVerificationRequest,
  signal?: AbortSignal,
) {
  return apiFetch<LoginResponse>(
    '/api/auth/login/verify',
    {
      method: 'POST',
      body: JSON.stringify(payload),
      signal,
    },
    { redirectOnUnauthorized: false, parse: parseLoginResponse },
  )
}

export function checkCurrentSession() {
  return apiFetch<CurrentSessionResponse>(
    '/api/auth/session',
    {},
    { redirectOnUnauthorized: false },
  )
}

export function logoutAccount() {
  return apiFetch<ApiMessageResponse>('/api/auth/logout', {
    method: 'POST',
  })
}

export function fetchFeed(cursor?: string) {
  const params = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ''
  return apiFetch<FeedResponse>(`/api/feed${params}`)
}

export function fetchMyPosts(cursor?: string) {
  const params = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ''
  return apiFetch<FeedResponse>(`/api/posts/me${params}`)
}

export function createPost(
  content: string,
  images: File[] = [],
  visibility: PostVisibility = 'public',
) {
  if (images.length === 0) {
    return apiFetch<CreatePostResponse>('/api/posts', {
      method: 'POST',
      body: JSON.stringify({ content, visibility }),
    })
  }

  const formData = new FormData()
  formData.append('content', content)
  formData.append('visibility', visibility)
  images.forEach((image) => formData.append('images', image))

  return apiFetch<CreatePostResponse>('/api/posts', {
    method: 'POST',
    body: formData,
  })
}

export function deletePost(postId: number) {
  return apiFetch<ApiMessageResponse>(`/api/posts/${postId}`, {
    method: 'DELETE',
  })
}

export interface InspirationItemResponse {
  id: number
  date: string
  text: string
  imageLabel?: string
}

export function fetchSchedule() {
  return apiFetch<StoredSchedule>('/api/schedule')
}

export function saveSchedule(schedule: StoredSchedule) {
  return apiFetch<StoredSchedule>('/api/schedule', {
    method: 'PUT',
    body: JSON.stringify(schedule),
  })
}

export function fetchInspirations() {
  return apiFetch<{ items: InspirationItemResponse[] }>('/api/inspirations')
}

export function createInspiration(payload: Omit<InspirationItemResponse, 'id'>) {
  return apiFetch<InspirationItemResponse>('/api/inspirations', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function updateInspiration(id: number, text: string) {
  return apiFetch<ApiMessageResponse>(`/api/inspirations/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ text }),
  })
}

export function deleteInspiration(id: number) {
  return apiFetch<ApiMessageResponse>(`/api/inspirations/${id}`, {
    method: 'DELETE',
  })
}
