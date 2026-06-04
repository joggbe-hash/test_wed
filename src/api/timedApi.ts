export interface ApiResponse<T> {
  data: T
  returnedAt: string
  delayMs: number
}

export function requestByTime<T>(data: T, delayMs = 1200): Promise<ApiResponse<T>> {
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

export interface HomeFeedData {
  posts: string[]
  shareText: string
}

export function fetchHomeFeed(): Promise<ApiResponse<HomeFeedData>> {
  return requestByTime<HomeFeedData>(
    {
      posts: ['?犖?批捆', '?潔??批捆'],
      shareText: '(??獢?',
    },
    1200,
  )
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
  return requestByTime<ExploreData>(
    {
      categories: Array.from({ length: 20 }, (_, index) => ({
        id: index + 1,
        name: '分類',
      })),
      rows: Array.from({ length: 3 }, (_, rowIndex) => ({
        id: rowIndex + 1,
        cards: Array.from({ length: 20 }, (_, cardIndex) => ({
          id: `${rowIndex + 1}-${cardIndex + 1}`,
          title: '主題社群',
          tags: '#興趣標籤',
          description: '社群簡介',
          members: '555 人',
        })),
      })),
    },
    900,
  )
}

export interface PersonalProfile {
  id: string
  bio: string
}

export interface PersonalData {
  profile: PersonalProfile
  posts: string[]
  timer: string
}

export function fetchPersonalData(): Promise<ApiResponse<PersonalData>> {
  return requestByTime<PersonalData>(
    {
      profile: {
        id: '@userid',
        bio: '個人簡介',
      },
      posts: ['私人貼文', '公開貼文'],
      timer: new Date().toISOString(),
    },
    900,
  )
}

export interface FreqData {
  timeCards: string[]
  stats: string[]
}

export function fetchFreqData(): Promise<ApiResponse<FreqData>> {
  const now = new Date().toISOString()

  return requestByTime<FreqData>(
    {
      timeCards: [now, now, now],
      stats: ['使用者資料', '搜尋紀錄', '通知列表', '設定項目'],
    },
    800,
  )
}

export type SocialPostType = 'text' | 'image'

export interface SocialPost {
  id: number
  type: SocialPostType
  text: string
}

export interface SocialData {
  composerText: string
  posts: SocialPost[]
}

export function fetchSocialData(): Promise<ApiResponse<SocialData>> {
  return requestByTime<SocialData>(
    {
      composerText: '輸入貼文',
      posts: Array.from({ length: 8 }, (_, index) => ({
        id: index + 1,
        type: index % 2 === 0 ? 'text' : 'image',
        text: '社群貼文',
      })),
    },
    1000,
  )
}

export interface ProfilePreviewData {
  username: string
  link: string
}

export function fetchProfilePreview(): Promise<ApiResponse<ProfilePreviewData>> {
  return requestByTime<ProfilePreviewData>(
    {
      username: '@使用者',
      link: 'haha.tww.com',
    },
    700,
  )
}
