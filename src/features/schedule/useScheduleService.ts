import { computed, readonly, ref, shallowRef } from 'vue'
import { getLocalTodayKey } from '../../utils/date'
import {
  priorityMeta,
  type LegacyScheduleDecisionResult,
  type ReminderItem,
  type TaskItem,
} from './types'
import {
  getLocalStorage,
  hasStorageValue,
  legacyScheduleStorageKey,
  readStoredSchedule,
  removeStorageValue,
  scheduleStorageKeyFor,
  writeStoredSchedule,
} from './scheduleRepository'
import { cloneSeedSchedule } from './scheduleSeed'
import type { StoredSchedule } from './types'

function nextIdFrom<T extends { id: number }>(items: T[], fallback: number) {
  return items.reduce((maxId, item) => Math.max(maxId, item.id), fallback - 1) + 1
}

const todayKey = shallowRef(getLocalTodayKey())
const tasks = ref<TaskItem[]>([])
const reminders = ref<ReminderItem[]>([])
const scheduleErrorMessage = shallowRef('')
let nextTaskId = 1
let nextReminderId = 1
let todayRefreshTimer: number | undefined
const scheduleOwnerId = shallowRef<number | null>(null)
const pendingLegacyScheduleOwnerId = shallowRef<number | null>(null)
const pendingLegacySchedule = shallowRef<StoredSchedule | null>(null)
const isLegacyScheduleDecisionPending = computed(() => pendingLegacyScheduleOwnerId.value !== null)
const isScheduleReady = computed(() =>
  scheduleOwnerId.value !== null && pendingLegacyScheduleOwnerId.value === null,
)

function refreshTodayKey() {
  todayKey.value = getLocalTodayKey()
}

if (typeof window !== 'undefined') {
  todayRefreshTimer = window.setInterval(refreshTodayKey, 60_000)

  if (import.meta.hot) {
    import.meta.hot.dispose(() => {
      if (todayRefreshTimer !== undefined) {
        window.clearInterval(todayRefreshTimer)
      }
    })
  }
}

function persistSchedule() {
  const ownerId = scheduleOwnerId.value
  const storage = getLocalStorage()
  if (!storage || ownerId === null || !isScheduleReady.value) {
    scheduleErrorMessage.value = '排程無法儲存，請確認瀏覽器允許使用本機儲存空間。'
    return false
  }

  const saved = writeStoredSchedule(storage, scheduleStorageKeyFor(ownerId), {
    tasks: tasks.value,
    reminders: reminders.value,
  })
  scheduleErrorMessage.value = saved
    ? ''
    : '排程儲存失敗，可能是瀏覽器儲存空間已滿或目前受到限制。'
  return saved
}

function applySchedule(ownerId: number, schedule: StoredSchedule) {
  scheduleOwnerId.value = ownerId
  pendingLegacyScheduleOwnerId.value = null
  pendingLegacySchedule.value = null
  tasks.value = schedule.tasks
  reminders.value = schedule.reminders
  nextTaskId = nextIdFrom(tasks.value, 4)
  nextReminderId = nextIdFrom(reminders.value, 3)
}

function waitForLegacyScheduleDecision(ownerId: number, schedule: StoredSchedule) {
  scheduleOwnerId.value = ownerId
  pendingLegacyScheduleOwnerId.value = ownerId
  pendingLegacySchedule.value = schedule
  tasks.value = []
  reminders.value = []
  nextTaskId = 1
  nextReminderId = 1
}

function setScheduleOwner(userId: number) {
  if (!Number.isSafeInteger(userId) || userId <= 0) {
    clearScheduleOwner()
    return false
  }

  if (scheduleOwnerId.value === userId) return isScheduleReady.value

  const storage = getLocalStorage()
  const userStorageKey = scheduleStorageKeyFor(userId)
  const storedSchedule = readStoredSchedule(userStorageKey, storage)
  if (storedSchedule) {
    applySchedule(userId, storedSchedule)
    return true
  }

  const legacySchedule = readStoredSchedule(legacyScheduleStorageKey, storage)
  if (storage && legacySchedule && hasStorageValue(storage, legacyScheduleStorageKey)) {
    waitForLegacyScheduleDecision(userId, legacySchedule)
    return false
  }

  const schedule = cloneSeedSchedule()
  if (storage) writeStoredSchedule(storage, userStorageKey, schedule)
  applySchedule(userId, schedule)
  return true
}

function importLegacySchedule(): LegacyScheduleDecisionResult {
  const ownerId = scheduleOwnerId.value
  const schedule = pendingLegacySchedule.value
  const storage = getLocalStorage()
  if (!storage || ownerId === null || pendingLegacyScheduleOwnerId.value !== ownerId || !schedule) {
    return { ok: false, message: '目前沒有可匯入的舊版排程資料。' }
  }

  const userStorageKey = scheduleStorageKeyFor(ownerId)
  if (!writeStoredSchedule(storage, userStorageKey, schedule)) {
    return { ok: false, message: '無法儲存匯入資料，請確認瀏覽器儲存空間後再試一次。' }
  }

  if (!removeStorageValue(storage, legacyScheduleStorageKey)) {
    removeStorageValue(storage, userStorageKey)
    return { ok: false, message: '無法完成舊資料清理，匯入尚未完成，請再試一次。' }
  }

  applySchedule(ownerId, schedule)
  return { ok: true }
}

function declineLegacySchedule(): LegacyScheduleDecisionResult {
  const ownerId = scheduleOwnerId.value
  const storage = getLocalStorage()
  if (ownerId === null || pendingLegacyScheduleOwnerId.value !== ownerId) {
    return { ok: false, message: '目前沒有等待處理的舊版排程資料。' }
  }
  if (!storage) {
    return { ok: false, message: '瀏覽器儲存空間無法使用，暫時不能建立新排程。' }
  }

  const schedule = cloneSeedSchedule()
  if (!writeStoredSchedule(storage, scheduleStorageKeyFor(ownerId), schedule)) {
    return { ok: false, message: '無法建立新排程，請確認瀏覽器儲存空間後再試一次。' }
  }

  applySchedule(ownerId, schedule)
  return { ok: true }
}

function clearScheduleOwner() {
  scheduleOwnerId.value = null
  pendingLegacyScheduleOwnerId.value = null
  pendingLegacySchedule.value = null
  tasks.value = []
  reminders.value = []
  nextTaskId = 1
  nextReminderId = 1
}

const sortedTasks = computed(() =>
  [...tasks.value].sort((a, b) => {
    const priorityDiff = priorityMeta[a.priority].rank - priorityMeta[b.priority].rank
    return priorityDiff || a.order - b.order
  }),
)

const todayTasks = computed(() =>
  sortedTasks.value.filter((task) => task.date === todayKey.value),
)

const sortedReminders = computed(() =>
  [...reminders.value].sort((a, b) => `${a.date} ${a.time}`.localeCompare(`${b.date} ${b.time}`)),
)

const todayReminders = computed(() =>
  sortedReminders.value.filter((reminder) => reminder.date === todayKey.value),
)

const readonlyTasks = readonly(tasks)
const readonlyReminders = readonly(reminders)
const readonlyTodayKey = readonly(todayKey)

export function useScheduleMock() {
  function addTask(payload: Omit<TaskItem, 'id' | 'completed' | 'order'>) {
    if (!isScheduleReady.value) return false

    const task: TaskItem = {
      ...payload,
      id: nextTaskId,
      completed: false,
      order: nextTaskId,
    }
    tasks.value.push(task)
    nextTaskId += 1
    if (!persistSchedule()) {
      tasks.value = tasks.value.filter((item) => item !== task)
      nextTaskId -= 1
      return false
    }
    return true
  }

  function addReminder(payload: Omit<ReminderItem, 'id'>) {
    if (!isScheduleReady.value) return false

    const reminder: ReminderItem = {
      ...payload,
      id: nextReminderId,
    }
    reminders.value.push(reminder)
    nextReminderId += 1
    if (!persistSchedule()) {
      reminders.value = reminders.value.filter((item) => item !== reminder)
      nextReminderId -= 1
      return false
    }
    return true
  }

  function updateTask(taskId: number, payload: Partial<Omit<TaskItem, 'id' | 'order'>>) {
    if (!isScheduleReady.value) return false

    const task = tasks.value.find((item) => item.id === taskId)
    if (!task) return false

    const previousTask = { ...task }
    Object.assign(task, payload)
    if (!persistSchedule()) {
      Object.assign(task, previousTask)
      return false
    }
    return true
  }

  function deleteTask(taskId: number) {
    if (!isScheduleReady.value) return false

    const nextTasks = tasks.value.filter((item) => item.id !== taskId)
    if (nextTasks.length === tasks.value.length) return false

    const previousTasks = tasks.value
    tasks.value = nextTasks
    if (!persistSchedule()) {
      tasks.value = previousTasks
      return false
    }
    return true
  }

  function updateReminder(reminderId: number, payload: Partial<Omit<ReminderItem, 'id'>>) {
    if (!isScheduleReady.value) return false

    const reminder = reminders.value.find((item) => item.id === reminderId)
    if (!reminder) return false

    const previousReminder = { ...reminder }
    Object.assign(reminder, payload)
    if (!persistSchedule()) {
      Object.assign(reminder, previousReminder)
      return false
    }
    return true
  }

  function deleteReminder(reminderId: number) {
    if (!isScheduleReady.value) return false

    const nextReminders = reminders.value.filter((item) => item.id !== reminderId)
    if (nextReminders.length === reminders.value.length) return false

    const previousReminders = reminders.value
    reminders.value = nextReminders
    if (!persistSchedule()) {
      reminders.value = previousReminders
      return false
    }
    return true
  }

  function toggleTask(taskId: number) {
    if (!isScheduleReady.value) return false

    const task = tasks.value.find((item) => item.id === taskId)
    if (!task) return false

    task.completed = !task.completed
    if (!persistSchedule()) {
      task.completed = !task.completed
      return false
    }
    return true
  }

  function reorderTaskWithinPriority(sourceId: number, targetId: number) {
    if (!isScheduleReady.value) return false

    const source = tasks.value.find((item) => item.id === sourceId)
    const target = tasks.value.find((item) => item.id === targetId)
    if (!source || !target || source.priority !== target.priority || source.id === target.id) return false

    const group = [...tasks.value]
      .filter((item) => item.priority === source.priority)
      .sort((a, b) => a.order - b.order)
    const from = group.findIndex((item) => item.id === sourceId)
    const to = group.findIndex((item) => item.id === targetId)
    if (from < 0 || to < 0) return false

    const [moved] = group.splice(from, 1)
    group.splice(to, 0, moved)
    const previousOrders = new Map(group.map((item) => [item.id, item.order]))
    group.forEach((item, index) => {
      item.order = priorityMeta[item.priority].rank * 100 + index
    })
    if (!persistSchedule()) {
      group.forEach((item) => {
        item.order = previousOrders.get(item.id) ?? item.order
      })
      return false
    }
    return true
  }

  return {
    tasks: readonlyTasks,
    reminders: readonlyReminders,
    todayKey: readonlyTodayKey,
    scheduleErrorMessage: readonly(scheduleErrorMessage),
    isScheduleReady,
    isLegacyScheduleDecisionPending,
    sortedTasks,
    todayTasks,
    sortedReminders,
    todayReminders,
    setScheduleOwner,
    clearScheduleOwner,
    importLegacySchedule,
    declineLegacySchedule,
    addTask,
    addReminder,
    updateTask,
    deleteTask,
    updateReminder,
    deleteReminder,
    toggleTask,
    reorderTaskWithinPriority,
  }
}

export {
  clearScheduleOwner,
  declineLegacySchedule,
  importLegacySchedule,
  isLegacyScheduleDecisionPending,
  isScheduleReady,
  setScheduleOwner,
}
