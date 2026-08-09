import { readonly, ref, shallowRef } from 'vue'
import {
  createInspiration,
  deleteInspiration,
  fetchInspirations,
  updateInspiration,
} from '../api/backendApi'
import { useSession } from './useSession'

export interface InspirationItem {
  id: number
  date: string
  text: string
  imageLabel?: string
}

export function useInspirationStore() {
  const { currentUser } = useSession()
  const items = ref<InspirationItem[]>([])
  const errorMessage = shallowRef('')
  const isLoading = shallowRef(false)
  let requestRevision = 0

  async function loadItems() {
    if (!currentUser.value) {
      items.value = []
      errorMessage.value = '請先登入後再使用靈感筆記。'
      return false
    }

    const revision = ++requestRevision
    isLoading.value = true
    try {
      const response = await fetchInspirations()
      if (revision !== requestRevision) return false
      items.value = response.items
      errorMessage.value = ''
      return true
    } catch {
      if (revision === requestRevision) {
        errorMessage.value = '無法從伺服器載入靈感筆記。'
      }
      return false
    } finally {
      if (revision === requestRevision) isLoading.value = false
    }
  }

  async function addItem(payload: Omit<InspirationItem, 'id'>) {
    try {
      const created = await createInspiration(payload)
      items.value = [created, ...items.value]
      errorMessage.value = ''
      return true
    } catch {
      errorMessage.value = '靈感筆記無法儲存到伺服器。'
      return false
    }
  }

  async function updateItem(id: number, text: string) {
    if (!items.value.some((item) => item.id === id)) return false
    try {
      await updateInspiration(id, text)
      items.value = items.value.map((item) => item.id === id ? { ...item, text } : item)
      errorMessage.value = ''
      return true
    } catch {
      errorMessage.value = '靈感筆記無法更新到伺服器。'
      return false
    }
  }

  async function deleteItem(id: number) {
    if (!items.value.some((item) => item.id === id)) return false
    try {
      await deleteInspiration(id)
      items.value = items.value.filter((item) => item.id !== id)
      errorMessage.value = ''
      return true
    } catch {
      errorMessage.value = '靈感筆記無法從伺服器刪除。'
      return false
    }
  }

  return {
    items: readonly(items),
    errorMessage: readonly(errorMessage),
    isLoading: readonly(isLoading),
    loadItems,
    addItem,
    updateItem,
    deleteItem,
  }
}
