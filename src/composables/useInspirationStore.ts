import { readonly, ref, shallowRef } from 'vue'
import { useSession } from './useSession'

export interface InspirationItem {
  id: number
  date: string
  text: string
  imageLabel?: string
}

const storageKeyPrefix = 'type-wsp-inspirations'
const maximumTextLength = 700
const legacyDemoItems: InspirationItem[] = [
  { id: 1, date: '2026-06-25', text: '去租書店把柯南漫畫裡的壞人都圈出來 *要帶紅筆' },
  { id: 2, date: '2026-06-24', text: '舉辦吃麻糬大賽可以解決人口老化問題' },
  {
    id: 3,
    date: '2026-06-24',
    text: '蛇的尿道跟肛門是同一個，尿液結晶有時會導致出口堵塞',
    imageLabel: '某貼文縮圖連結',
  },
  { id: 4, date: '2026-06-18', text: '買電擊棒偷電路人' },
]

function isLegacyDemoItem(item: InspirationItem) {
  return legacyDemoItems.some((demo) =>
    item.id === demo.id
      && item.date === demo.date
      && item.text === demo.text
      && item.imageLabel === demo.imageLabel,
  )
}

function getStorage() {
  try {
    return typeof window === 'undefined' ? null : window.localStorage
  } catch {
    return null
  }
}

function readStorageItem(storage: Storage | null, key: string | null) {
  if (!storage || !key) return { value: null, failed: false }
  try {
    return { value: storage.getItem(key), failed: false }
  } catch {
    return { value: null, failed: true }
  }
}

function isValidDateKey(value: unknown): value is string {
  return typeof value === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(value)
}

function parseStoredItems(value: string | null): InspirationItem[] | null {
  if (value === null) return null

  try {
    const parsed: unknown = JSON.parse(value)
    if (!Array.isArray(parsed)) return null

    const items = parsed.filter((item): item is InspirationItem => {
      if (!item || typeof item !== 'object') return false
      const candidate = item as Partial<InspirationItem>
      return Number.isSafeInteger(candidate.id)
        && Number(candidate.id) > 0
        && isValidDateKey(candidate.date)
        && typeof candidate.text === 'string'
        && candidate.text.trim().length > 0
        && candidate.text.length <= maximumTextLength
        && (candidate.imageLabel === undefined || typeof candidate.imageLabel === 'string')
    })
    return items.length === parsed.length ? items : null
  } catch {
    return null
  }
}

export function useInspirationStore() {
  const { currentUser } = useSession()
  const errorMessage = shallowRef('')
  const ownerId = currentUser.value?.id ?? null
  const storageKey = ownerId === null ? null : `${storageKeyPrefix}:${ownerId}`
  const storage = getStorage()
  const storedValue = readStorageItem(storage, storageKey)
  const storedItems = parseStoredItems(storedValue.value)
  const cleanedItems = storedItems?.filter((item) => !isLegacyDemoItem(item)) ?? []
  const needsDemoCleanup = storedItems !== null && cleanedItems.length !== storedItems.length
  const items = ref<InspirationItem[]>(cleanedItems)

  if (!storage || storedValue.failed) {
    errorMessage.value = '靈感清單無法使用瀏覽器儲存空間，新增內容可能無法保存。'
  } else if (ownerId === null) {
    errorMessage.value = '尚未取得登入使用者，暫時無法儲存靈感清單。'
  } else if (storedValue.value !== null && storedItems === null) {
    errorMessage.value = '既有靈感清單資料格式異常，目前無法載入。'
  } else if (needsDemoCleanup && storageKey) {
    try {
      storage.setItem(storageKey, JSON.stringify(cleanedItems))
    } catch {
      errorMessage.value = '無法清除舊版示範資料，請確認瀏覽器允許使用本機儲存空間。'
    }
  }

  function persist(nextItems: InspirationItem[]) {
    if (!storage || !storageKey) {
      errorMessage.value = '靈感清單儲存失敗，請重新登入並確認瀏覽器允許本機儲存。'
      return false
    }

    try {
      storage.setItem(storageKey, JSON.stringify(nextItems))
      items.value = nextItems
      errorMessage.value = ''
      return true
    } catch {
      errorMessage.value = '靈感清單儲存失敗，可能是瀏覽器儲存空間已滿或受到限制。'
      return false
    }
  }

  function addItem(payload: Omit<InspirationItem, 'id'>) {
    const nextId = items.value.reduce((largest, item) => Math.max(largest, item.id), 0) + 1
    return persist([{ ...payload, id: nextId }, ...items.value])
  }

  function updateItem(id: number, text: string) {
    if (!items.value.some((item) => item.id === id)) return false
    return persist(items.value.map((item) => item.id === id ? { ...item, text } : item))
  }

  function deleteItem(id: number) {
    const nextItems = items.value.filter((item) => item.id !== id)
    if (nextItems.length === items.value.length) return false
    return persist(nextItems)
  }

  return {
    items: readonly(items),
    errorMessage: readonly(errorMessage),
    addItem,
    updateItem,
    deleteItem,
  }
}
