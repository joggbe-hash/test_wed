import {
  priorityToImportance,
  type Priority,
  type ReminderItem,
  type StoredSchedule,
  type TaskImportance,
  type TaskItem,
} from './types'

type StoredTaskItem = Omit<TaskItem, 'importance'> & {
  importance?: TaskImportance | 0
}

const scheduleStorageKeyPrefix = 'type-wsp-schedule-mock'
export const legacyScheduleStorageKey = scheduleStorageKeyPrefix

export function getLocalStorage(): Storage | null {
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
  return typeof value === 'number'
    && Number.isInteger(value)
    && value >= 0
    && value <= 5
}

function normalizeTaskImportance(
  importance: TaskImportance | 0 | undefined,
  priority: Priority,
): TaskImportance {
  if (importance === undefined) return priorityToImportance(priority)
  if (importance === 0) return 1
  return importance
}

function isTaskItem(value: unknown): value is StoredTaskItem {
  return isRecord(value)
    && typeof value.id === 'number'
    && typeof value.title === 'string'
    && (value.note === undefined || typeof value.note === 'string')
    && typeof value.date === 'string'
    && typeof value.time === 'string'
    && (value.endTime === undefined || typeof value.endTime === 'string')
    && isPriority(value.priority)
    && (value.importance === undefined || isStoredTaskImportance(value.importance))
    && typeof value.completed === 'boolean'
    && typeof value.order === 'number'
}

function normalizeTaskItem(task: StoredTaskItem): TaskItem {
  return {
    ...task,
    importance: normalizeTaskImportance(task.importance, task.priority),
  }
}

function isReminderItem(value: unknown): value is ReminderItem {
  return isRecord(value)
    && typeof value.id === 'number'
    && typeof value.title === 'string'
    && typeof value.date === 'string'
    && typeof value.time === 'string'
    && (value.endTime === undefined || typeof value.endTime === 'string')
    && typeof value.note === 'string'
}

export function readStoredSchedule(
  storageKey: string,
  storage = getLocalStorage(),
): StoredSchedule | null {
  if (!storage) return null

  try {
    const rawValue = storage.getItem(storageKey)
    if (!rawValue) return null

    const parsedValue: unknown = JSON.parse(rawValue)
    if (
      !isRecord(parsedValue)
      || !Array.isArray(parsedValue.tasks)
      || !Array.isArray(parsedValue.reminders)
    ) {
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

export function hasStorageValue(storage: Storage, storageKey: string) {
  try {
    return storage.getItem(storageKey) !== null
  } catch {
    return false
  }
}

export function writeStoredSchedule(
  storage: Storage,
  storageKey: string,
  schedule: StoredSchedule,
) {
  try {
    storage.setItem(storageKey, JSON.stringify(schedule))
    return true
  } catch {
    return false
  }
}

export function removeStorageValue(storage: Storage, storageKey: string) {
  try {
    storage.removeItem(storageKey)
    return true
  } catch {
    return false
  }
}

export function scheduleStorageKeyFor(userId: number) {
  return `${scheduleStorageKeyPrefix}:${userId}`
}
