export interface User {
  id: number
  username: string
  email: string
}

export interface BackendPost {
  id: number
  user_id: number
  username: string
  content?: string
  image_urls?: string[]
  image_status: string
  created_at: string
}

export interface FeedResponse {
  posts: BackendPost[]
  next_cursor: string
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

async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    ...init,
    credentials: 'include',
    headers: {
      ...(init.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }),
      ...init.headers,
    },
  })

  const text = await response.text()
  const data = text ? JSON.parse(text) : null

  if (!response.ok) {
    throw new ApiError(response.status, data?.error ?? data?.message ?? 'API request failed')
  }

  return data as T
}

export function sendVerificationCode(email: string) {
  return apiFetch<{ message: string; debug_code?: string }>('/api/auth/send-code', {
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

export function loginAccount(email: string, password: string) {
  return apiFetch<{ user: User }>('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
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

export function createPost(content: string, images: File[] = []) {
  if (images.length === 0) {
    return apiFetch<{ message: string; post_id: number }>('/api/posts', {
      method: 'POST',
      body: JSON.stringify({ content }),
    })
  }

  const formData = new FormData()
  formData.append('content', content)
  images.forEach((image) => formData.append('images', image))

  return apiFetch<{ message: string; post_id: number }>('/api/posts', {
    method: 'POST',
    body: formData,
  })
}
