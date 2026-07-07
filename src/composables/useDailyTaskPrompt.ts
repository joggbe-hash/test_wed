import { shallowRef } from 'vue'
import { getLocalTodayKey } from '../utils/date'

const IGNORED_PATHS = new Set(['/login', '/forgot-password'])

export const showDailyTaskPrompt = shallowRef(false)

let hasPromptedForCurrentLogin = false

export function getTodayTaskDate() {
  return getLocalTodayKey()
}

export function maybeOpenDailyTaskPrompt(path: string) {
  if (IGNORED_PATHS.has(path)) {
    hasPromptedForCurrentLogin = false
    showDailyTaskPrompt.value = false
    return
  }

  if (hasPromptedForCurrentLogin) return

  hasPromptedForCurrentLogin = true
  showDailyTaskPrompt.value = true
}

export function markDailyTaskPromptHandled() {
  showDailyTaskPrompt.value = false
}
