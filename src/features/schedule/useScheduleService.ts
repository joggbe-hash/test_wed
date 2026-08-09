import { computed, readonly, ref, shallowRef } from 'vue'
import { fetchSchedule, saveSchedule } from '../../api/backendApi'
import { getLocalTodayKey } from '../../utils/date'
import {
  priorityMeta,
  type ReminderItem,
  type StoredSchedule,
  type TaskItem,
} from './types'

function nextIdFrom<T extends { id: number }>(items: T[]) {
  return items.reduce((maxId, item) => Math.max(maxId, item.id), 0) + 1
}

const todayKey = shallowRef(getLocalTodayKey())
const tasks = ref<TaskItem[]>([])
const reminders = ref<ReminderItem[]>([])
const scheduleErrorMessage = shallowRef('')
const scheduleOwnerId = shallowRef<number | null>(null)
const isScheduleReady = shallowRef(false)
const isLegacyScheduleDecisionPending = shallowRef(false)
let todayRefreshTimer: number | undefined
let loadRevision = 0
let saveQueue: Promise<void> = Promise.resolve()

function refreshTodayKey() {
  todayKey.value = getLocalTodayKey()
}

if (typeof window !== 'undefined') {
  todayRefreshTimer = window.setInterval(refreshTodayKey, 60_000)
  if (import.meta.hot) {
    import.meta.hot.dispose(() => {
      if (todayRefreshTimer !== undefined) window.clearInterval(todayRefreshTimer)
    })
  }
}

function snapshotSchedule(): StoredSchedule {
  return {
    tasks: tasks.value.map((task) => ({ ...task })),
    reminders: reminders.value.map((reminder) => ({ ...reminder })),
  }
}

function queueScheduleSave(ownerId: number) {
  const snapshot = snapshotSchedule()
  saveQueue = saveQueue.then(async () => {
    if (scheduleOwnerId.value !== ownerId) return
    try {
      await saveSchedule(snapshot)
      if (scheduleOwnerId.value === ownerId) scheduleErrorMessage.value = ''
    } catch {
      if (scheduleOwnerId.value === ownerId) {
        scheduleErrorMessage.value = '行程無法儲存到伺服器，請確認連線後再試一次。'
      }
    }
  })
}

async function setScheduleOwner(userId: number) {
  if (!Number.isSafeInteger(userId) || userId <= 0) {
    clearScheduleOwner()
    return false
  }
  if (scheduleOwnerId.value === userId && isScheduleReady.value) return true

  const revision = ++loadRevision
  scheduleOwnerId.value = userId
  isScheduleReady.value = false
  tasks.value = []
  reminders.value = []
  scheduleErrorMessage.value = ''

  try {
    const schedule = await fetchSchedule()
    if (revision !== loadRevision || scheduleOwnerId.value !== userId) return false
    tasks.value = schedule.tasks
    reminders.value = schedule.reminders
    isScheduleReady.value = true
    return true
  } catch {
    if (revision === loadRevision && scheduleOwnerId.value === userId) {
      scheduleErrorMessage.value = '無法從伺服器載入行程資料。'
    }
    return false
  }
}

function clearScheduleOwner() {
  loadRevision += 1
  scheduleOwnerId.value = null
  isScheduleReady.value = false
  tasks.value = []
  reminders.value = []
  scheduleErrorMessage.value = ''
}

function mutateSchedule(mutation: () => boolean) {
  const ownerId = scheduleOwnerId.value
  if (ownerId === null || !isScheduleReady.value || !mutation()) return false
  queueScheduleSave(ownerId)
  return true
}

const sortedTasks = computed(() =>
  [...tasks.value].sort((a, b) => {
    const priorityDiff = priorityMeta[a.priority].rank - priorityMeta[b.priority].rank
    return priorityDiff || a.order - b.order
  }),
)

const todayTasks = computed(() => sortedTasks.value.filter((task) => task.date === todayKey.value))
const sortedReminders = computed(() =>
  [...reminders.value].sort((a, b) => `${a.date} ${a.time}`.localeCompare(`${b.date} ${b.time}`)),
)
const todayReminders = computed(() =>
  sortedReminders.value.filter((reminder) => reminder.date === todayKey.value),
)

export function useSchedule() {
  function addTask(payload: Omit<TaskItem, 'id' | 'completed' | 'order'>) {
    return mutateSchedule(() => {
      const id = nextIdFrom(tasks.value)
      tasks.value.push({ ...payload, id, completed: false, order: id })
      return true
    })
  }

  function addReminder(payload: Omit<ReminderItem, 'id'>) {
    return mutateSchedule(() => {
      reminders.value.push({ ...payload, id: nextIdFrom(reminders.value) })
      return true
    })
  }

  function updateTask(taskId: number, payload: Partial<Omit<TaskItem, 'id' | 'order'>>) {
    return mutateSchedule(() => {
      const task = tasks.value.find((item) => item.id === taskId)
      if (!task) return false
      Object.assign(task, payload)
      return true
    })
  }

  function deleteTask(taskId: number) {
    return mutateSchedule(() => {
      const nextTasks = tasks.value.filter((item) => item.id !== taskId)
      if (nextTasks.length === tasks.value.length) return false
      tasks.value = nextTasks
      return true
    })
  }

  function updateReminder(reminderId: number, payload: Partial<Omit<ReminderItem, 'id'>>) {
    return mutateSchedule(() => {
      const reminder = reminders.value.find((item) => item.id === reminderId)
      if (!reminder) return false
      Object.assign(reminder, payload)
      return true
    })
  }

  function deleteReminder(reminderId: number) {
    return mutateSchedule(() => {
      const nextReminders = reminders.value.filter((item) => item.id !== reminderId)
      if (nextReminders.length === reminders.value.length) return false
      reminders.value = nextReminders
      return true
    })
  }

  function toggleTask(taskId: number) {
    return mutateSchedule(() => {
      const task = tasks.value.find((item) => item.id === taskId)
      if (!task) return false
      task.completed = !task.completed
      return true
    })
  }

  function reorderTaskWithinPriority(sourceId: number, targetId: number) {
    return mutateSchedule(() => {
      const source = tasks.value.find((item) => item.id === sourceId)
      const target = tasks.value.find((item) => item.id === targetId)
      if (!source || !target || source.priority !== target.priority || source.id === target.id) return false
      const group = tasks.value
        .filter((item) => item.priority === source.priority)
        .sort((a, b) => a.order - b.order)
      const from = group.findIndex((item) => item.id === sourceId)
      const to = group.findIndex((item) => item.id === targetId)
      const [moved] = group.splice(from, 1)
      group.splice(to, 0, moved)
      group.forEach((item, index) => {
        item.order = priorityMeta[item.priority].rank * 100 + index
      })
      return true
    })
  }

  return {
    tasks: readonly(tasks),
    reminders: readonly(reminders),
    todayKey: readonly(todayKey),
    scheduleErrorMessage: readonly(scheduleErrorMessage),
    isScheduleReady: readonly(isScheduleReady),
    isLegacyScheduleDecisionPending: readonly(isLegacyScheduleDecisionPending),
    sortedTasks,
    todayTasks,
    sortedReminders,
    todayReminders,
    setScheduleOwner,
    clearScheduleOwner,
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
  isLegacyScheduleDecisionPending,
  isScheduleReady,
  setScheduleOwner,
}
