import { onMounted, onUnmounted, readonly, shallowRef } from 'vue'

const minuteMilliseconds = 60 * 1000
const now = shallowRef(Date.now())
let activeConsumers = 0
let refreshTimer: number | undefined

function startSharedClock() {
  activeConsumers += 1
  now.value = Date.now()

  if (refreshTimer !== undefined) return

  refreshTimer = window.setInterval(() => {
    now.value = Date.now()
  }, minuteMilliseconds)
}

function stopSharedClock() {
  activeConsumers = Math.max(0, activeConsumers - 1)
  if (activeConsumers > 0 || refreshTimer === undefined) return

  window.clearInterval(refreshTimer)
  refreshTimer = undefined
}

export function useSharedMinuteNow() {
  onMounted(startSharedClock)
  onUnmounted(stopSharedClock)

  return readonly(now)
}

if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    if (refreshTimer !== undefined) {
      window.clearInterval(refreshTimer)
      refreshTimer = undefined
    }
    activeConsumers = 0
  })
}
