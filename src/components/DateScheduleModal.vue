<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, onUnmounted, shallowRef, useTemplateRef, watch } from 'vue'
import {
  type ReminderItem,
  type TaskItem,
  useSchedule,
} from '../composables/useSchedule'
import {
  focusWithoutScroll,
  isUsableFocusTarget,
  trapDialogFocus,
} from '../composables/useDialogFocusTrap'
import { useDateScheduleEditor } from '../features/schedule/useDateScheduleEditor'
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
const { deleteTask, deleteReminder } = useSchedule()

type PendingDelete =
  | { kind: 'task'; item: TaskItem }
  | { kind: 'reminder'; item: ReminderItem }
const pendingDelete = shallowRef<PendingDelete | null>(null)
const dateScheduleModal = useTemplateRef<HTMLElement>('dateScheduleModal')
const dateScheduleClose = useTemplateRef<HTMLButtonElement>('dateScheduleClose')
const reminderEditor = useTemplateRef<InstanceType<typeof DateScheduleReminderEditor>>('reminderEditor')
let outerReturnFocus = typeof document !== 'undefined' && document.activeElement instanceof HTMLElement
  ? document.activeElement
  : null
let reminderEditorReturnFocus: HTMLElement | null = null
let reminderEditorReturnId: number | null = null
let pendingDeleteReturnFocus: HTMLElement | null = null

const {
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
} = useDateScheduleEditor({
  props,
  close: () => emit('close'),
  updateDateKey: (value) => emit('update:dateKey', value),
  rememberReminderEditorTrigger,
  focusReminderEditor,
})
const isOuterDialogInert = computed(() =>
  isReminderEditorOpen.value || isDailyTaskPromptOpen.value,
)
const isOuterDialogInactive = computed(() =>
  isOuterDialogInert.value || pendingDelete.value !== null,
)
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
