import { shallowRef } from 'vue'
import { getLocalTodayKey } from '../utils/date'

const IGNORED_PATHS = new Set(['/', '/login', '/forgot-password'])

export const showDailyTaskPrompt = shallowRef(false)

let hasPromptedForCurrentLogin = false

export function getTodayTaskDate() {
  return getLocalTodayKey()
}

export function resetDailyTaskPrompt() {
  hasPromptedForCurrentLogin = false
  showDailyTaskPrompt.value = false
}

export function maybeOpenDailyTaskPrompt(path: string, scheduleReady: boolean) {
  if (!scheduleReady || IGNORED_PATHS.has(path)) {
    resetDailyTaskPrompt()
    return
  }

  if (hasPromptedForCurrentLogin) return

  hasPromptedForCurrentLogin = true
  showDailyTaskPrompt.value = true
}

export function markDailyTaskPromptHandled() {
  showDailyTaskPrompt.value = false
}
