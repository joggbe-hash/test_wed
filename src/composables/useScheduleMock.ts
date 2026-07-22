import { computed, readonly, ref, shallowRef } from 'vue'
import { getLocalTodayKey } from '../utils/date'

export type Priority = 'high' | 'medium' | 'low'
export type TaskImportance = 1 | 2 | 3 | 4 | 5

export interface TaskItem {
  id: number
  title: string
  note?: string
  date: string
  time: string
  endTime?: string
  priority: Priority
  importance?: TaskImportance
  completed: boolean
  order: number
}

export interface ReminderItem {
  id: number
  title: string
  date: string
  time: string
  endTime?: string
  note: string
}

export type LegacyScheduleDecisionResult =
  | { ok: true }
  | { ok: false; message: string }

export const priorityMeta: Record<Priority, { label: string, rank: number }> = {
  high: { label: '高', rank: 1 },
  medium: { label: '中', rank: 2 },
  low: { label: '低', rank: 3 },
}

export function priorityToImportance(priority: Priority): TaskImportance {
  if (priority === 'high') return 5
  if (priority === 'medium') return 3
  return 1
}

export function taskImportanceCount(task: Pick<TaskItem, 'importance' | 'priority'>): TaskImportance {
  return task.importance ?? priorityToImportance(task.priority)
}

interface StoredSchedule {
  tasks: TaskItem[]
  reminders: ReminderItem[]
}

type StoredTaskItem = Omit<TaskItem, 'importance'> & { importance?: TaskImportance | 0 }

const scheduleStorageKeyPrefix = 'type-wsp-schedule-mock'
const legacyScheduleStorageKey = scheduleStorageKeyPrefix

const seedTasks: Array<Omit<TaskItem, 'date'>> = [
  { id: 1, title: '完成專題流程圖', time: '09:30', priority: 'high', importance: 5, completed: false, order: 1 },
  { id: 2, title: '整理貼文互動版面', time: '11:00', priority: 'high', importance: 5, completed: false, order: 2 },
  { id: 3, title: '確認左側邊欄內容', time: '15:20', priority: 'medium', importance: 3, completed: true, order: 3 },
]

const seedReminders: Array<Omit<ReminderItem, 'date'>> = [
  { id: 1, title: '提醒', time: '08:50', note: '檢查今天任務' },
  { id: 2, title: '回顧', time: '21:00', note: '記下今天想做的事' },
]

function getLocalStorage(): Storage | null {
  if (typeof window === 'undefined') return null

  try {
    return window.localStorage
  } catch {
    return null
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object'
}

function isPriority(value: unknown): value is Priority {
  return value === 'high' || value === 'medium' || value === 'low'
}

function isStoredTaskImportance(value: unknown): value is TaskImportance | 0 {
  return typeof value === 'number' && Number.isInteger(value) && value >= 0 && value <= 5
}

function normalizeTaskImportance(importance: TaskImportance | 0 | undefined, priority: Priority): TaskImportance {
  if (importance === undefined) return priorityToImportance(priority)
  if (importance === 0) return 1
  return importance
}

function isTaskItem(value: unknown): value is StoredTaskItem {
  return isRecord(value) &&
    typeof value.id === 'number' &&
    typeof value.title === 'string' &&
    (value.note === undefined || typeof value.note === 'string') &&
    typeof value.date === 'string' &&
    typeof value.time === 'string' &&
    (value.endTime === undefined || typeof value.endTime === 'string') &&
    isPriority(value.priority) &&
    (value.importance === undefined || isStoredTaskImportance(value.importance)) &&
    typeof value.completed === 'boolean' &&
    typeof value.order === 'number'
}

function normalizeTaskItem(task: StoredTaskItem): TaskItem {
  return {
    ...task,
    importance: normalizeTaskImportance(task.importance, task.priority),
  }
}

function isReminderItem(value: unknown): value is ReminderItem {
  return isRecord(value) &&
    typeof value.id === 'number' &&
    typeof value.title === 'string' &&
    typeof value.date === 'string' &&
    typeof value.time === 'string' &&
    (value.endTime === undefined || typeof value.endTime === 'string') &&
    typeof value.note === 'string'
}

function readStoredSchedule(storageKey: string, storage = getLocalStorage()): StoredSchedule | null {
  if (!storage) return null

  try {
    const rawValue = storage.getItem(storageKey)
    if (!rawValue) return null

    const parsedValue: unknown = JSON.parse(rawValue)
    if (!isRecord(parsedValue) || !Array.isArray(parsedValue.tasks) || !Array.isArray(parsedValue.reminders)) {
      return null
    }

    if (!parsedValue.tasks.every(isTaskItem) || !parsedValue.reminders.every(isReminderItem)) {
      return null
    }

    return {
      tasks: parsedValue.tasks.map(normalizeTaskItem),
      reminders: parsedValue.reminders,
    }
  } catch {
    return null
  }
}

function hasStorageValue(storage: Storage, storageKey: string) {
  try {
    return storage.getItem(storageKey) !== null
  } catch {
    return false
  }
}

function writeStoredSchedule(storage: Storage, storageKey: string, schedule: StoredSchedule) {
  try {
    storage.setItem(storageKey, JSON.stringify(schedule))
    return true
  } catch {
    return false
  }
}

function removeStorageValue(storage: Storage, storageKey: string) {
  try {
    storage.removeItem(storageKey)
    return true
  } catch {
    return false
  }
}

function nextIdFrom<T extends { id: number }>(items: T[], fallback: number) {
  return items.reduce((maxId, item) => Math.max(maxId, item.id), fallback - 1) + 1
}

function cloneSeedSchedule(): StoredSchedule {
  const date = getLocalTodayKey()

  return {
    tasks: seedTasks.map((task) => ({ ...task, date })),
    reminders: seedReminders.map((reminder) => ({ ...reminder, date })),
  }
}

function scheduleStorageKeyFor(userId: number) {
  return `${scheduleStorageKeyPrefix}:${userId}`
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
    tasks,
    reminders,
    todayKey,
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
