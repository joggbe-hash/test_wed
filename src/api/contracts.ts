export interface User {
  id: number
  username: string
}

export type PostVisibility = 'public' | 'private'

export type ImageStatus = 'none' | 'processing' | 'ready' | 'failed'

export interface BackendPost {
  id: number
  user_id: number
  username: string
  visibility: PostVisibility
  content?: string
  image_urls?: string[]
  image_status: ImageStatus
  created_at: string
}

export interface FeedResponse {
  posts: BackendPost[]
  next_cursor: string
}

export interface CurrentSessionResponse {
  user: User
}

export interface ApiMessageResponse {
  message: string
}

export interface SendVerificationCodeResponse extends ApiMessageResponse {
  challenge_id: string
}

export interface LoginChallengeResponse extends ApiMessageResponse {
  challenge_id: string
  requires_verification: true
  expires_in_seconds: number
}

export interface LoginOwnershipChallenge {
  challenge_id: string
  code_format: 'base32-16-v1'
  expires_in_seconds: number
}

export interface LoginOwnershipVerificationRequest {
  email: string
  challenge_id: string
  code: string
}

export interface LoginOwnershipGrantResponse {
  password_verification_grant: string
  expires_in_seconds: number
  max_attempts: number
}

export interface LoginVerificationRequest {
  email: string
  code: string
  challenge_id: string
  remember: boolean
}

export interface LoginResponse {
  user: User
}

export interface RegisterAccountRequest {
  username: string
  email: string
  password: string
  code: string
  challenge_id: string
}

export interface CreatePostResponse extends ApiMessageResponse {
  post_id: number
}
