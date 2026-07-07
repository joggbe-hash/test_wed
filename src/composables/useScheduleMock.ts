import { computed, ref, shallowRef } from 'vue'
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

const scheduleStorageKey = 'type-wsp-schedule-mock'
const today = getLocalTodayKey()

const seedTasks: TaskItem[] = [
  { id: 1, title: '完成專題流程圖', date: today, time: '09:30', priority: 'high', importance: 5, completed: false, order: 1 },
  { id: 2, title: '整理貼文互動版面', date: today, time: '11:00', priority: 'high', importance: 5, completed: false, order: 2 },
  { id: 3, title: '確認左側邊欄內容', date: today, time: '15:20', priority: 'medium', importance: 3, completed: true, order: 3 },
]

const seedReminders: ReminderItem[] = [
  { id: 1, title: '提醒', date: today, time: '08:50', note: '檢查今天任務' },
  { id: 2, title: '回顧', date: today, time: '21:00', note: '記下今天想做的事' },
]

function hasLocalStorage() {
  return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined'
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

function readStoredSchedule(): StoredSchedule | null {
  if (!hasLocalStorage()) return null

  try {
    const rawValue = window.localStorage.getItem(scheduleStorageKey)
    if (!rawValue) return null

    const parsedValue: unknown = JSON.parse(rawValue)
    if (!isRecord(parsedValue) || !Array.isArray(parsedValue.tasks) || !Array.isArray(parsedValue.reminders)) {
      return null
    }

    const storedTasks = parsedValue.tasks.filter(isTaskItem).map(normalizeTaskItem)
    const storedReminders = parsedValue.reminders.filter(isReminderItem)
    return {
      tasks: storedTasks,
      reminders: storedReminders,
    }
  } catch {
    return null
  }
}

function nextIdFrom<T extends { id: number }>(items: T[], fallback: number) {
  return items.reduce((maxId, item) => Math.max(maxId, item.id), fallback - 1) + 1
}

const storedSchedule = readStoredSchedule()
const todayKey = shallowRef(getLocalTodayKey())
const tasks = ref<TaskItem[]>(storedSchedule?.tasks ?? seedTasks)
const reminders = ref<ReminderItem[]>(storedSchedule?.reminders ?? seedReminders)
let nextTaskId = nextIdFrom(tasks.value, 4)
let nextReminderId = nextIdFrom(reminders.value, 3)
let todayRefreshTimer: number | undefined

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
  if (!hasLocalStorage()) return

  try {
    window.localStorage.setItem(scheduleStorageKey, JSON.stringify({
      tasks: tasks.value,
      reminders: reminders.value,
    }))
  } catch {
    // localStorage may be unavailable in private mode or when quota is exceeded.
  }
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
    tasks.value.push({
      ...payload,
      id: nextTaskId,
      completed: false,
      order: nextTaskId,
    })
    nextTaskId += 1
    persistSchedule()
  }

  function addReminder(payload: Omit<ReminderItem, 'id'>) {
    reminders.value.push({
      ...payload,
      id: nextReminderId,
    })
    nextReminderId += 1
    persistSchedule()
  }

  function updateTask(taskId: number, payload: Partial<Omit<TaskItem, 'id' | 'order'>>) {
    const task = tasks.value.find((item) => item.id === taskId)
    if (!task) return false

    Object.assign(task, payload)
    persistSchedule()
    return true
  }

  function deleteTask(taskId: number) {
    const nextTasks = tasks.value.filter((item) => item.id !== taskId)
    if (nextTasks.length === tasks.value.length) return false

    tasks.value = nextTasks
    persistSchedule()
    return true
  }

  function updateReminder(reminderId: number, payload: Partial<Omit<ReminderItem, 'id'>>) {
    const reminder = reminders.value.find((item) => item.id === reminderId)
    if (!reminder) return false

    Object.assign(reminder, payload)
    persistSchedule()
    return true
  }

  function deleteReminder(reminderId: number) {
    const nextReminders = reminders.value.filter((item) => item.id !== reminderId)
    if (nextReminders.length === reminders.value.length) return false

    reminders.value = nextReminders
    persistSchedule()
    return true
  }

  function toggleTask(taskId: number) {
    const task = tasks.value.find((item) => item.id === taskId)
    if (!task) return

    task.completed = !task.completed
    persistSchedule()
  }

  function reorderTaskWithinPriority(sourceId: number, targetId: number) {
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
    group.forEach((item, index) => {
      item.order = priorityMeta[item.priority].rank * 100 + index
    })
    persistSchedule()
    return true
  }

  return {
    tasks,
    reminders,
    todayKey,
    sortedTasks,
    todayTasks,
    sortedReminders,
    todayReminders,
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
