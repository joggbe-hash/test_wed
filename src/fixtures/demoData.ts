import type {
  ExploreData,
  FreqData,
  ProfilePreviewData,
} from '../api/timedApi'

export function createExploreDemoData(): ExploreData {
  return {
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
  }
}

export function createFreqDemoData(): FreqData {
  return {
    actions: [
      { id: 'profile', title: '使用者資料' },
      { id: 'explore', title: '搜尋紀錄' },
      { id: 'notifications', title: '通知列表' },
      { id: 'settings', title: '設定項目' },
    ],
  }
}

export function createProfilePreviewDemoData(): ProfilePreviewData {
  return {
    username: '@使用者',
    link: 'demo.type-wsp.local',
  }
}
