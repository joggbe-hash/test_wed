import { computed, reactive, shallowRef, type DeepReadonly } from 'vue'
import { useSchedule } from '../../composables/useSchedule'
import { formatLocalDateKey } from '../../utils/date'
import type { Priority, ReminderItem, TaskItem } from './types'

export type DateScheduleEditingKind = 'task' | 'reminder' | 'new-reminder'

interface DateScheduleEditorProps {
  dateKey: string
  editReminderId?: number | null
}

interface DateScheduleEditorOptions {
  props: DeepReadonly<DateScheduleEditorProps>
  close: () => void
  updateDateKey: (value: string) => void
  rememberReminderEditorTrigger: (reminderId: number | null) => void
  focusReminderEditor: () => void
}

function parseDateKey(dateKey: string) {
  const [year, month, day] = dateKey.split('-').map(Number)
  return new Date(year, month - 1, day)
}

function addMinutesToTime(time: string, minutesToAdd: number) {
  const [rawHours = 0, rawMinutes = 0] = time.split(':').map(Number)
  const totalMinutes = ((rawHours * 60 + rawMinutes + minutesToAdd) % 1440 + 1440) % 1440
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}`
}

function defaultReminderEndTime(startTime: string) {
  return addMinutesToTime(startTime || '08:00', 60)
}

export function useDateScheduleEditor(options: DateScheduleEditorOptions) {
  const {
    sortedTasks,
    sortedReminders,
    addReminder,
    updateTask,
    updateReminder,
  } = useSchedule()
  const editingKind = shallowRef<DateScheduleEditingKind | null>(null)
  const editingId = shallowRef<number | null>(null)
  const isDailyTaskPromptOpen = shallowRef(false)
  const taskPromptEditId = shallowRef<number | null>(null)
  const editForm = reactive({
    title: '',
    note: '',
    priority: 'medium' as Priority,
    date: options.props.dateKey,
    startTime: '08:00',
    endTime: '09:00',
  })

  const selectedDate = computed(() => parseDateKey(options.props.dateKey))
  const selectedDay = computed(() => selectedDate.value.getDate())
  const selectedLabel = computed(() =>
    new Intl.DateTimeFormat('zh-TW', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      weekday: 'short',
    }).format(selectedDate.value),
  )
  const reminderEditorDateKey = computed(() => editForm.date || options.props.dateKey)
  const reminderEditorDate = computed(() => parseDateKey(reminderEditorDateKey.value))
  const reminderEditorDay = computed(() => reminderEditorDate.value.getDate())
  const reminderEditorDateLabel = computed(() => {
    const editorDate = reminderEditorDate.value
    const weekday = new Intl.DateTimeFormat('zh-TW', { weekday: 'short' }).format(editorDate)
    return `${editorDate.getMonth() + 1}月${editorDate.getDate()}日 ${weekday}`
  })
  const dateReminders = computed(() =>
    sortedReminders.value.filter((reminder) => reminder.date === options.props.dateKey),
  )
  const dateTasks = computed(() =>
    sortedTasks.value.filter((task) => task.date === options.props.dateKey),
  )
  const isReminderForm = computed(() =>
    editingKind.value === 'reminder' || editingKind.value === 'new-reminder',
  )
  const isTaskEditPanelOpen = computed(() => editingKind.value === 'task')
  const isReminderEditorOpen = computed(() => isReminderForm.value)
  const editPanelTitle = computed(() => {
    if (editingKind.value === 'new-reminder') return '新增提醒'
    if (editingKind.value === 'reminder') return '編輯提醒'
    return '編輯任務'
  })

  function cancelEdit() {
    editingKind.value = null
    editingId.value = null
  }

  function shiftDate(offset: number) {
    const nextDate = parseDateKey(options.props.dateKey)
    nextDate.setDate(nextDate.getDate() + offset)
    cancelEdit()
    options.updateDateKey(formatLocalDateKey(nextDate))
  }

  function startAddReminder() {
    options.rememberReminderEditorTrigger(null)
    editingKind.value = 'new-reminder'
    editingId.value = null
    editForm.title = ''
    editForm.note = ''
    editForm.priority = 'medium'
    editForm.date = options.props.dateKey
    editForm.startTime = '08:00'
    editForm.endTime = defaultReminderEndTime(editForm.startTime)
  }

  function startEditReminder(reminder: ReminderItem) {
    options.rememberReminderEditorTrigger(reminder.id)
    editingKind.value = 'reminder'
    editingId.value = reminder.id
    editForm.title = reminder.title
    editForm.note = reminder.note
    editForm.priority = 'medium'
    editForm.date = reminder.date
    editForm.startTime = reminder.time
    editForm.endTime = reminder.endTime ?? defaultReminderEndTime(reminder.time)
  }

  function openRequestedReminderEditor() {
    const reminderId = options.props.editReminderId
    if (reminderId === undefined || reminderId === null) return false

    const reminder = sortedReminders.value.find((item) => item.id === reminderId)
    if (!reminder) {
      options.close()
      return true
    }

    startEditReminder(reminder)
    return true
  }

  function startEditTask(task: TaskItem) {
    cancelEdit()
    taskPromptEditId.value = task.id
    isDailyTaskPromptOpen.value = true
  }

  function openAddTaskPrompt() {
    cancelEdit()
    taskPromptEditId.value = null
    isDailyTaskPromptOpen.value = true
  }

  function closeTaskPrompt() {
    isDailyTaskPromptOpen.value = false
    taskPromptEditId.value = null
  }

  function saveReminder(closeAfterSave = true) {
    const kind = editingKind.value
    const title = editForm.title.trim()
    if (!kind || !isReminderForm.value || !title) return false

    const date = editForm.date || options.props.dateKey
    const startTime = editForm.startTime || '08:00'
    const endTime = !editForm.endTime || editForm.endTime <= startTime
      ? defaultReminderEndTime(startTime)
      : editForm.endTime
    const saved = kind === 'new-reminder'
      ? addReminder({
        title,
        date,
        time: startTime,
        endTime,
        note: editForm.note.trim(),
      })
      : editingId.value
        ? updateReminder(editingId.value, {
          title,
          date,
          time: startTime,
          endTime,
          note: editForm.note.trim(),
        })
        : false

    if (!saved) return false

    options.updateDateKey(date)
    if (closeAfterSave) cancelEdit()
    return true
  }

  function saveReminderAndCreateNext() {
    if (!saveReminder(false)) return

    editingKind.value = 'new-reminder'
    editingId.value = null
    editForm.title = ''
    editForm.note = ''
    options.focusReminderEditor()
  }

  function saveEdit() {
    if (isReminderForm.value) {
      saveReminder(true)
      return
    }

    const id = editingId.value
    const title = editForm.title.trim()
    if (!editingKind.value || !id || !title) return

    if (updateTask(id, {
      title,
      note: editForm.note.trim(),
      priority: editForm.priority,
    })) {
      cancelEdit()
    }
  }

  return {
    editingKind,
    isDailyTaskPromptOpen,
    taskPromptEditId,
    editForm,
    selectedDay,
    selectedLabel,
    reminderEditorDay,
    reminderEditorDateLabel,
    dateReminders,
    dateTasks,
    isTaskEditPanelOpen,
    isReminderEditorOpen,
    editPanelTitle,
    shiftDate,
    startAddReminder,
    startEditReminder,
    openRequestedReminderEditor,
    startEditTask,
    openAddTaskPrompt,
    closeTaskPrompt,
    cancelEdit,
    saveReminderAndCreateNext,
    saveEdit,
  }
}
