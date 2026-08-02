import { notifyUnauthorized } from './unauthorizedHandler'

export interface User {
  id: number
  username: string
  email: string
}

export interface BackendPost {
  id: number
  user_id: number
  username: string
  visibility: PostVisibility
  content?: string
  image_urls?: string[]
  image_status: string
  created_at: string
}

export type PostVisibility = 'public' | 'private'

export interface FeedResponse {
  posts: BackendPost[]
  next_cursor: string
}

export interface CurrentSessionResponse {
  user: User
}

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? ''

interface ApiFetchOptions {
  redirectOnUnauthorized?: boolean
}

// 前端所有後端請求都經過這裡，統一處理身分憑證、資料標頭、錯誤格式和未登入導回。
async function apiFetch<T>(
  path: string,
  init: RequestInit = {},
  options: ApiFetchOptions = {},
): Promise<T> {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    ...init,
    credentials: 'include',
    headers: {
      ...(init.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }),
      ...init.headers,
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
    )
  }

  if (!isJson) {
    throw new ApiError(
      response.status,
      `Expected JSON from ${path}, but received ${contentType || 'unknown content type'}`,
    )
  }

  return data as T
}

export function sendVerificationCode(email: string) {
  return apiFetch<{ message: string }>('/api/auth/send-code', {
    method: 'POST',
    body: JSON.stringify({ email }),
  })
}

export function registerAccount(payload: {
  username: string
  email: string
  password: string
  code: string
}) {
  return apiFetch<{ message: string; user_id: number }>('/api/auth/register', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function loginAccount(email: string, password: string, remember = false) {
  return apiFetch<{ user: User }>(
    '/api/auth/login',
    {
      method: 'POST',
      body: JSON.stringify({ email, password, remember }),
    },
    { redirectOnUnauthorized: false },
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
  return apiFetch<{ message: string }>('/api/auth/logout', {
    method: 'POST',
  })
}

export function fetchFeed(cursor?: string) {
  const params = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ''
  return apiFetch<FeedResponse>(`/api/feed${params}`)
}

export function createPost(
  content: string,
  images: File[] = [],
  visibility: PostVisibility = 'public',
) {
  if (images.length === 0) {
    return apiFetch<{ message: string; post_id: number }>('/api/posts', {
      method: 'POST',
      body: JSON.stringify({ content, visibility }),
    })
  }

  const formData = new FormData()
  formData.append('content', content)
  formData.append('visibility', visibility)
  images.forEach((image) => formData.append('images', image))

  return apiFetch<{ message: string; post_id: number }>('/api/posts', {
    method: 'POST',
    body: formData,
  })
}

export function deletePost(postId: number) {
  return apiFetch<{ message: string }>(`/api/posts/${postId}`, {
    method: 'DELETE',
  })
}
