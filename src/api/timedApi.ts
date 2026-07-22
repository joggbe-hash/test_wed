import {
  createExploreDemoData,
  createFreqDemoData,
  createProfilePreviewDemoData,
} from '../fixtures/demoData'

export interface ApiResponse<T> {
  data: T
  returnedAt: string
  delayMs: number
}

const isDemoApiEnabled = import.meta.env.VITE_USE_MOCK_API === 'true'

function requireDemoApi() {
  if (!isDemoApiEnabled) {
    throw new Error('此頁面的資料 API 尚未啟用；請連接正式 API 或在 demo build 啟用 mock。')
  }
}

export function requestByTime<T>(data: T, delayMs = 1200): Promise<ApiResponse<T>> {
  requireDemoApi()

  return new Promise((resolve) => {
    window.setTimeout(() => {
      resolve({
        data,
        returnedAt: new Date().toISOString(),
        delayMs,
      })
    }, delayMs)
  })
}

export interface ExploreCategory {
  id: number
  name: string
}

export interface ExploreCard {
  id: string
  title: string
  tags: string
  description: string
  members: string
}

export interface ExploreRow {
  id: number
  cards: ExploreCard[]
}

export interface ExploreData {
  categories: ExploreCategory[]
  rows: ExploreRow[]
}

export function fetchExploreData(): Promise<ApiResponse<ExploreData>> {
  return requestByTime(createExploreDemoData(), 900)
}

export type FreqActionId = 'profile' | 'explore' | 'notifications' | 'settings'

export interface FreqActionSummary {
  id: FreqActionId
  title: string
}

export interface FreqData {
  actions: FreqActionSummary[]
}

export function fetchFreqData(): Promise<ApiResponse<FreqData>> {
  return requestByTime(createFreqDemoData(), 800)
}

export interface ProfilePreviewData {
  username: string
  link: string
}

export function fetchProfilePreview(): Promise<ApiResponse<ProfilePreviewData>> {
  return requestByTime(createProfilePreviewDemoData(), 700)
}
