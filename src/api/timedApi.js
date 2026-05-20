export function requestByTime(data, delayMs = 1200) {
  return new Promise((resolve) => {
    window.setTimeout(() => {
      resolve({
        data,
        returnedAt: new Date(),
        delayMs,
      })
    }, delayMs)
  })
}

export function fetchHomeFeed() {
  return requestByTime(
    {
      posts: ['個人內容', '發佈內容'],
      shareText: '(打字框)',
    },
    1200,
  )
}

export function fetchExploreData() {
  return requestByTime(
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

export function fetchPersonalData() {
  return requestByTime(
    {
      profile: {
        id: '@userid',
        bio: '個人簡介',
      },
      posts: ['私人貼文', '公開貼文'],
      timer: '02:00:58',
    },
    900,
  )
}

export function fetchFreqData() {
  return requestByTime(
    {
      timeCards: ['時間', '時間', '時間'],
      stats: ['使用者資料', '互動數據', '收藏紀錄', '設定項目'],
    },
    800,
  )
}

export function fetchSocialData() {
  return requestByTime(
    {
      composerText: '輸入內容',
      posts: Array.from({ length: 8 }, (_, index) => ({
        id: index + 1,
        type: index % 2 === 0 ? 'text' : 'image',
        text: '社群貼文',
      })),
    },
    1000,
  )
}

export function fetchProfilePreview() {
  return requestByTime(
    {
      username: '@使用者',
      link: 'haha.tww.com',
    },
    700,
  )
}
