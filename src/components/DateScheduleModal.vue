<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, onUnmounted, reactive, shallowRef, useTemplateRef, watch } from 'vue'
import {
  type Priority,
  type ReminderItem,
  type TaskItem,
  useScheduleMock,
} from '../composables/useScheduleMock'
import { formatLocalDateKey } from '../utils/date'
import DailyTaskPrompt from './DailyTaskPrompt.vue'
import DateScheduleItems from './DateScheduleItems.vue'
import DateScheduleReminderEditor from './DateScheduleReminderEditor.vue'
import TaskDeleteConfirmDialog from './TaskDeleteConfirmDialog.vue'

const props = defineProps<{
  dateKey: string
  startWithNewReminder?: boolean
  editReminderId?: number | null
}>()

const emit = defineEmits<{
  close: []
  'update:dateKey': [value: string]
}>()
const {
  sortedTasks,
  sortedReminders,
  addReminder,
  updateTask,
  deleteTask,
  updateReminder,
  deleteReminder,
} = useScheduleMock()

type EditingKind = 'task' | 'reminder' | 'new-reminder'
type PendingDelete =
  | { kind: 'task'; item: TaskItem }
  | { kind: 'reminder'; item: ReminderItem }
const editingKind = shallowRef<EditingKind | null>(null)
const editingId = shallowRef<number | null>(null)
const isDailyTaskPromptOpen = shallowRef(false)
const taskPromptEditId = shallowRef<number | null>(null)
const pendingDelete = shallowRef<PendingDelete | null>(null)
const editForm = reactive({
  title: '',
  note: '',
  priority: 'medium' as Priority,
  date: props.dateKey,
  startTime: '08:00',
  endTime: '09:00',
})
const dateScheduleModal = useTemplateRef<HTMLElement>('dateScheduleModal')
const dateScheduleClose = useTemplateRef<HTMLButtonElement>('dateScheduleClose')
const reminderEditor = useTemplateRef<InstanceType<typeof DateScheduleReminderEditor>>('reminderEditor')
let outerReturnFocus = typeof document !== 'undefined' && document.activeElement instanceof HTMLElement
  ? document.activeElement
  : null
let reminderEditorReturnFocus: HTMLElement | null = null
let reminderEditorReturnId: number | null = null
let pendingDeleteReturnFocus: HTMLElement | null = null
function parseDateKey(dateKey: string) {
  const [year, month, day] = dateKey.split('-').map(Number)
  return new Date(year, month - 1, day)
}

const selectedDate = computed(() => parseDateKey(props.dateKey))
const selectedDay = computed(() => selectedDate.value.getDate())
const selectedLabel = computed(() =>
  new Intl.DateTimeFormat('zh-TW', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    weekday: 'short',
  }).format(selectedDate.value),
)
const reminderEditorDateKey = computed(() => editForm.date || props.dateKey)
const reminderEditorDate = computed(() => parseDateKey(reminderEditorDateKey.value))
const reminderEditorDay = computed(() => reminderEditorDate.value.getDate())
const reminderEditorDateLabel = computed(() => formatReminderDateLabel(reminderEditorDateKey.value))

const dateReminders = computed(() =>
  sortedReminders.value.filter((reminder) => reminder.date === props.dateKey),
)
const dateTasks = computed(() =>
  sortedTasks.value.filter((task) => task.date === props.dateKey),
)

const isReminderForm = computed(() => editingKind.value === 'reminder' || editingKind.value === 'new-reminder')
const isTaskEditPanelOpen = computed(() => editingKind.value === 'task')
const isReminderEditorOpen = computed(() => isReminderForm.value)
const isOuterDialogInert = computed(() =>
  isReminderEditorOpen.value || isDailyTaskPromptOpen.value,
)
const isOuterDialogInactive = computed(() =>
  isOuterDialogInert.value || pendingDelete.value !== null,
)
const editPanelTitle = computed(() => {
  if (editingKind.value === 'new-reminder') return '新增提醒'
  if (editingKind.value === 'reminder') return '編輯提醒'
  return '編輯任務'
})

const focusableSelector = [
  'button:not([disabled])',
  '[href]',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(', ')

function isUsableFocusTarget(element: HTMLElement | null): element is HTMLElement {
  return Boolean(element && element.isConnected && element !== document.body)
}

function focusWithoutScroll(element: HTMLElement | null) {
  if (!isUsableFocusTarget(element)) return
  element.focus({ preventScroll: true })
}

function getFocusableElements(panel: HTMLElement | null) {
  if (!panel) return []

  return Array.from(panel.querySelectorAll<HTMLElement>(focusableSelector))
    .filter((element) => element.getClientRects().length > 0)
}

function trapDialogFocus(event: KeyboardEvent, panel: HTMLElement | null) {
  const focusableElements = getFocusableElements(panel)
  if (!panel || focusableElements.length === 0) {
    event.preventDefault()
    focusWithoutScroll(panel)
    return
  }

  const firstElement = focusableElements[0]
  const lastElement = focusableElements[focusableElements.length - 1]
  const activeElement = document.activeElement

  if (!(activeElement instanceof Node) || !panel.contains(activeElement)) {
    event.preventDefault()
    focusWithoutScroll(event.shiftKey ? lastElement : firstElement)
    return
  }

  if (event.shiftKey && activeElement === firstElement) {
    event.preventDefault()
    focusWithoutScroll(lastElement)
    return
  }

  if (!event.shiftKey && activeElement === lastElement) {
    event.preventDefault()
    focusWithoutScroll(firstElement)
  }
}

function handleOuterDialogTab(event: KeyboardEvent) {
  trapDialogFocus(event, dateScheduleModal.value)
}

function handleReminderDialogTab(event: KeyboardEvent) {
  trapDialogFocus(event, reminderEditor.value?.dialog ?? null)
}

function rememberReminderEditorTrigger(reminderId: number | null) {
  const activeElement = document.activeElement
  reminderEditorReturnFocus = activeElement instanceof HTMLElement ? activeElement : null
  reminderEditorReturnId = reminderId
}

function focusOuterDialog() {
  void nextTick(() => focusWithoutScroll(dateScheduleClose.value))
}

function focusReminderEditor() {
  void nextTick(() => focusWithoutScroll(reminderEditor.value?.titleInput ?? null))
}

function restoreReminderEditorFocus() {
  const returnFocus = reminderEditorReturnFocus
  const reminderId = reminderEditorReturnId
  reminderEditorReturnFocus = null
  reminderEditorReturnId = null

  void nextTick(() => {
    const fallback = reminderId === null
      ? dateScheduleModal.value?.querySelector<HTMLElement>('[data-add-reminder]') ?? null
      : dateScheduleModal.value?.querySelector<HTMLButtonElement>(`[data-reminder-edit-id="${reminderId}"]`) ?? null
    const returnFocusIsInsideOuter = Boolean(
      isUsableFocusTarget(returnFocus) && dateScheduleModal.value?.contains(returnFocus),
    )
    focusWithoutScroll(returnFocusIsInsideOuter ? returnFocus : fallback ?? dateScheduleClose.value)
  })
}

function resolveOuterReturnFocus() {
  if (isUsableFocusTarget(outerReturnFocus)) return outerReturnFocus

  if (props.editReminderId !== undefined && props.editReminderId !== null) {
    return document.querySelector<HTMLElement>(
      `[aria-controls="sidebar-reminder-menu-${props.editReminderId}"]`,
    )
  }

  if (props.startWithNewReminder) {
    return document.querySelector<HTMLElement>('.reminder-panel .add-task-btn')
  }

  return document.querySelector<HTMLElement>(
    `[aria-label="查看 ${props.dateKey} 的行程"]`,
  )
}

function formatReminderDateLabel(dateKey: string) {
  return new Intl.DateTimeFormat('zh-TW', {
    month: 'numeric',
    day: 'numeric',
    weekday: 'short',
  }).format(parseDateKey(dateKey))
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

function shiftDate(offset: number) {
  const nextDate = parseDateKey(props.dateKey)
  nextDate.setDate(nextDate.getDate() + offset)
  cancelEdit()
  emit('update:dateKey', formatLocalDateKey(nextDate))
}

function startAddReminder() {
  rememberReminderEditorTrigger(null)
  editingKind.value = 'new-reminder'
  editingId.value = null
  editForm.title = ''
  editForm.note = ''
  editForm.priority = 'medium'
  editForm.date = props.dateKey
  editForm.startTime = '08:00'
  editForm.endTime = defaultReminderEndTime(editForm.startTime)
}

function startEditReminder(reminder: ReminderItem) {
  rememberReminderEditorTrigger(reminder.id)
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
  const reminderId = props.editReminderId
  if (reminderId === undefined || reminderId === null) return false

  const reminder = sortedReminders.value.find((item) => item.id === reminderId)
  if (!reminder) {
    emit('close')
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

function cancelEdit() {
  editingKind.value = null
  editingId.value = null
}

function saveReminder(closeAfterSave = true) {
  const kind = editingKind.value
  const title = editForm.title.trim()
  if (!kind || !isReminderForm.value || !title) return false

  const date = editForm.date || props.dateKey
  const startTime = editForm.startTime || '08:00'
  const endTime = !editForm.endTime || editForm.endTime <= startTime
    ? defaultReminderEndTime(startTime)
    : editForm.endTime

  let saved = false
  if (kind === 'new-reminder') {
    saved = addReminder({
      title,
      date,
      time: startTime,
      endTime,
      note: editForm.note.trim(),
    })
  } else {
    const id = editingId.value
    if (!id) return false

    saved = updateReminder(id, {
      title,
      date,
      time: startTime,
      endTime,
      note: editForm.note.trim(),
    })
  }

  if (!saved) return false

  emit('update:dateKey', date)
  if (closeAfterSave) {
    cancelEdit()
  }

  return true
}

function saveReminderAndCreateNext() {
  if (!saveReminder(false)) return

  editingKind.value = 'new-reminder'
  editingId.value = null
  editForm.title = ''
  editForm.note = ''
  focusReminderEditor()
}

function saveEdit() {
  if (isReminderForm.value) {
    saveReminder(true)
    return
  }

  const kind = editingKind.value
  const title = editForm.title.trim()
  if (!kind || !title) return

  const id = editingId.value
  if (!id) return

  const saved = updateTask(id, {
    title,
    note: editForm.note.trim(),
    priority: editForm.priority,
  })

  if (saved) cancelEdit()
}

function requestDeleteReminder(reminder: ReminderItem) {
  pendingDeleteReturnFocus = document.activeElement instanceof HTMLElement
    ? document.activeElement
    : null
  pendingDelete.value = { kind: 'reminder', item: reminder }
}

function requestDeleteTask(task: TaskItem) {
  pendingDeleteReturnFocus = document.activeElement instanceof HTMLElement
    ? document.activeElement
    : null
  pendingDelete.value = { kind: 'task', item: task }
}

function closeDeleteConfirmation() {
  const returnFocus = pendingDeleteReturnFocus
  pendingDeleteReturnFocus = null
  pendingDelete.value = null
  void nextTick(() => {
    const returnFocusIsInsideOuter = Boolean(
      isUsableFocusTarget(returnFocus) && dateScheduleModal.value?.contains(returnFocus),
    )
    focusWithoutScroll(returnFocusIsInsideOuter ? returnFocus : dateScheduleClose.value)
  })
}

function confirmPendingDelete() {
  const pending = pendingDelete.value
  if (!pending) return

  const deleted = pending.kind === 'reminder'
    ? deleteReminder(pending.item.id)
    : deleteTask(pending.item.id)
  pendingDeleteReturnFocus = null
  pendingDelete.value = null

  if (deleted) {
    void nextTick(() => focusWithoutScroll(dateScheduleClose.value))
  }
}

const handleKeyDown = (e: KeyboardEvent) => {
  if (e.defaultPrevented || pendingDelete.value || isDailyTaskPromptOpen.value) return

  if (e.key === 'Escape') {
    e.preventDefault()
    if (editingKind.value) {
      cancelEdit()
      return
    }

    emit('close')
  }
}

function handleBackdropClick() {
  if (pendingDelete.value || isDailyTaskPromptOpen.value) return

  if (isReminderEditorOpen.value) {
    cancelEdit()
    return
  }

  emit('close')
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
  if (isReminderEditorOpen.value) {
    focusReminderEditor()
    return
  }

  focusOuterDialog()
})

onBeforeUnmount(() => {
  focusWithoutScroll(resolveOuterReturnFocus())
  outerReturnFocus = null
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
})

watch(
  () => [props.startWithNewReminder, props.editReminderId] as const,
  () => {
    if (openRequestedReminderEditor()) return
    if (props.startWithNewReminder) startAddReminder()
  },
  { immediate: true },
)

watch(
  isReminderEditorOpen,
  (isOpen, wasOpen) => {
    if (isOpen) {
      focusReminderEditor()
      return
    }

    if (wasOpen) restoreReminderEditorFocus()
  },
  { flush: 'post' },
)
</script>

<template>
  <Teleport to="body">
    <div class="date-schedule-backdrop" role="presentation" @click.self="handleBackdropClick">
      <section
        ref="dateScheduleModal"
        class="date-schedule-modal"
        :class="{
          'date-schedule-modal-reminder-editing': isReminderEditorOpen,
          'date-schedule-modal-task-prompt-open': isDailyTaskPromptOpen,
        }"
        role="dialog"
        :inert="isOuterDialogInactive"
        :aria-hidden="isOuterDialogInactive ? 'true' : undefined"
        :aria-modal="isOuterDialogInactive ? undefined : 'true'"
        aria-labelledby="date-schedule-title"
        tabindex="-1"
        @keydown.tab="handleOuterDialogTab"
      >
        <header class="date-schedule-header">
          <button type="button" class="date-schedule-nav" aria-label="前一天" @click="shiftDate(-1)">
            &lt;
          </button>

          <div class="date-schedule-calendar-mark" aria-hidden="true">
            <span>{{ selectedDay }}</span>
          </div>

          <button type="button" class="date-schedule-nav" aria-label="後一天" @click="shiftDate(1)">
            &gt;
          </button>

          <p id="date-schedule-title" class="date-schedule-title">{{ selectedLabel }}</p>

          <button ref="dateScheduleClose" type="button" class="date-schedule-close" aria-label="關閉" @click="emit('close')">
            <span aria-hidden="true">&times;</span>
          </button>
        </header>

        <form v-if="isTaskEditPanelOpen" class="date-edit-panel" @submit.prevent="saveEdit">
          <h3>{{ editPanelTitle }}</h3>

          <label>
            <span>標題</span>
            <input v-model="editForm.title" type="text" required>
          </label>

          <label class="date-edit-note">
            <span>備註</span>
            <textarea v-model="editForm.note" rows="2"></textarea>
          </label>

          <label>
            <span>重要程度</span>
            <select v-model="editForm.priority">
              <option value="high">高</option>
              <option value="medium">中</option>
              <option value="low">低</option>
            </select>
          </label>

          <div class="date-edit-actions">
            <button type="submit">儲存</button>
            <button type="button" @click="cancelEdit">取消</button>
          </div>
        </form>

        <DateScheduleItems
          :reminders="dateReminders"
          :tasks="dateTasks"
          @add-reminder="startAddReminder"
          @add-task="openAddTaskPrompt"
          @edit-reminder="startEditReminder"
          @edit-task="startEditTask"
          @delete-reminder="requestDeleteReminder"
          @delete-task="requestDeleteTask"
        />
      </section>

      <DateScheduleReminderEditor
        v-if="isReminderEditorOpen"
        ref="reminderEditor"
        v-model:title="editForm.title"
        v-model:note="editForm.note"
        v-model:date="editForm.date"
        v-model:start-time="editForm.startTime"
        v-model:end-time="editForm.endTime"
        :editor-title="editPanelTitle"
        :day="reminderEditorDay"
        :date-label="reminderEditorDateLabel"
        :is-new="editingKind === 'new-reminder'"
        :inactive="pendingDelete !== null"
        @submit="saveEdit"
        @cancel="cancelEdit"
        @save-next="saveReminderAndCreateNext"
        @keydown.tab="handleReminderDialogTab"
      />

      <DailyTaskPrompt
        v-if="isDailyTaskPromptOpen"
        :task-date="dateKey"
        :edit-task-id="taskPromptEditId"
        @close="closeTaskPrompt"
      />

      <TaskDeleteConfirmDialog
        v-if="pendingDelete"
        :item="pendingDelete.item"
        :kind="pendingDelete.kind"
        @cancel="closeDeleteConfirmation"
        @confirm="confirmPendingDelete"
      />
    </div>
  </Teleport>
</template>
