<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import AddTaskModal from './AddTaskModal.vue'
import DateScheduleModal from './DateScheduleModal.vue'
import InspirationListModal from './InspirationListModal.vue'
import TaskDeleteConfirmDialog from './TaskDeleteConfirmDialog.vue'
import SidebarReminderPanel from './SidebarReminderPanel.vue'
import SidebarTaskPanel from './SidebarTaskPanel.vue'
import { type ReminderItem, type TaskItem, useScheduleMock } from '../composables/useScheduleMock'

const weekDays = ['SUN', 'MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT']
const currentDate = shallowRef(new Date())
const visibleMonth = shallowRef(new Date(currentDate.value.getFullYear(), currentDate.value.getMonth(), 1))
const selectedDateKey = ref<string | null>(null)
const isAddTaskModalOpen = shallowRef(false)
const isAddReminderModalOpen = shallowRef(false)
const isInspirationListOpen = shallowRef(false)
const isReminderListExpanded = shallowRef(false)
const isTaskListExpanded = shallowRef(false)
const openTaskMenuId = shallowRef<number | null>(null)
const openReminderMenuId = shallowRef<number | null>(null)
const taskMenuTop = shallowRef(0)
const taskMenuLeft = shallowRef(0)
const editingTaskId = shallowRef<number | null>(null)
const editingReminderId = shallowRef<number | null>(null)
const pendingDeleteTask = shallowRef<TaskItem | null>(null)
const pendingDeleteReminder = shallowRef<ReminderItem | null>(null)
const activeReminderNoteId = shallowRef<number | null>(null)
const activeReminderNoteTop = shallowRef(0)
const activeReminderNoteLeft = shallowRef(0)
const activeTaskNoteId = shallowRef<number | null>(null)
const activeTaskNoteTop = shallowRef(0)
const activeTaskNoteLeft = shallowRef(0)
const {
  todayKey,
  todayTasks,
  todayReminders,
  toggleTask,
  deleteTask,
  deleteReminder,
} = useScheduleMock()

const reminderPanelHeadingId = 'sidebar-reminder-heading'
const taskPanelHeadingId = 'sidebar-task-heading'
const sidebarReminders = computed(() => todayReminders.value)
const sidebarTasks = computed(() => todayTasks.value)
const activeReminderNote = computed(() => {
  if (openReminderMenuId.value !== null) return ''

  const reminder = sidebarReminders.value.find((item) => item.id === activeReminderNoteId.value)
  return reminder?.note?.trim() ?? ''
})
const activeTaskNote = computed(() => {
  if (openTaskMenuId.value !== null) return ''

  const task = sidebarTasks.value.find((item) => item.id === activeTaskNoteId.value)
  return task?.note?.trim() ?? ''
})
const activeReminderNoteStyle = computed(() => ({
  top: `${activeReminderNoteTop.value}px`,
  left: `${activeReminderNoteLeft.value}px`,
}))
const activeTaskNoteStyle = computed(() => ({
  top: `${activeTaskNoteTop.value}px`,
  left: `${activeTaskNoteLeft.value}px`,
}))
const taskMenuStyle = computed(() => ({
  top: `${taskMenuTop.value}px`,
  left: `${taskMenuLeft.value}px`,
}))
let calendarTimer: number | undefined

const monthLabel = computed(() =>
  new Intl.DateTimeFormat('en-US', {
    month: 'long',
    year: 'numeric',
  }).format(visibleMonth.value),
)

const calendarCells = computed(() => {
  const year = visibleMonth.value.getFullYear()
  const month = visibleMonth.value.getMonth()
  const todayDateKey = todayKey.value
  const firstDay = new Date(year, month, 1).getDay()
  const daysInMonth = new Date(year, month + 1, 0).getDate()
  const cells = Array.from({ length: firstDay }, (_, index) => ({
    key: `blank-${index}`,
    label: '',
    dateKey: '',
    isToday: false,
    isCurrentMonth: false,
  }))

  for (let day = 1; day <= daysInMonth; day += 1) {
    const dateKey = [
      year,
      String(month + 1).padStart(2, '0'),
      String(day).padStart(2, '0'),
    ].join('-')

    cells.push({
      key: dateKey,
      label: String(day),
      dateKey,
      isToday: dateKey === todayDateKey,
      isCurrentMonth: true,
    })
  }

  return cells
})

function shiftMonth(offset: number) {
  visibleMonth.value = new Date(
    visibleMonth.value.getFullYear(),
    visibleMonth.value.getMonth() + offset,
    1,
  )
}

function setSelectedDateKey(dateKey: string | null) {
  selectedDateKey.value = dateKey
  if (!dateKey) return

  const [year, month] = dateKey.split('-').map(Number)
  if (!year || !month) return

  visibleMonth.value = new Date(year, month - 1, 1)
}

function openDateSchedule(dateKey: string) {
  if (!dateKey) return
  isAddReminderModalOpen.value = false
  editingReminderId.value = null
  setSelectedDateKey(dateKey)
}

function openTodayReminderForm() {
  isAddReminderModalOpen.value = true
  editingReminderId.value = null
  setSelectedDateKey(todayKey.value)
}

function closeDateSchedule() {
  isAddReminderModalOpen.value = false
  editingReminderId.value = null
  setSelectedDateKey(null)
}

function taskMenuTriggerId(taskId: number) {
  return `sidebar-task-menu-trigger-${taskId}`
}

function reminderMenuTriggerId(reminderId: number) {
  return `sidebar-reminder-menu-trigger-${reminderId}`
}

function previewPosition(row: HTMLElement, text: string) {
  const rect = row.getBoundingClientRect()
  const tooltipWidth = Math.min(260, Math.max(96, text.length * 8 + 28))
  const rightLeft = rect.right + 10
  const leftLeft = rect.left - tooltipWidth - 10
  const hasRightSpace = rightLeft + tooltipWidth <= window.innerWidth - 8

  return {
    top: Math.max(12, Math.min(rect.top + rect.height / 2, window.innerHeight - 12)),
    left: hasRightSpace ? rightLeft : Math.max(8, leftLeft),
  }
}

function showReminderNote(reminder: ReminderItem, event: MouseEvent | FocusEvent) {
  const note = reminder.note?.trim()
  if (!note) {
    activeReminderNoteId.value = null
    return
  }

  const row = event.currentTarget
  if (row instanceof HTMLElement) {
    const position = previewPosition(row, note)
    activeReminderNoteTop.value = position.top
    activeReminderNoteLeft.value = position.left
  }

  activeTaskNoteId.value = null
  activeReminderNoteId.value = reminder.id
}

function hideReminderNote(reminderId: number) {
  if (activeReminderNoteId.value === reminderId) {
    activeReminderNoteId.value = null
  }
}

function handleReminderFocusOut(event: FocusEvent, reminderId: number) {
  const nextTarget = event.relatedTarget
  const currentTarget = event.currentTarget
  if (
    nextTarget instanceof Node &&
    currentTarget instanceof HTMLElement &&
    currentTarget.contains(nextTarget)
  ) {
    return
  }

  hideReminderNote(reminderId)
}

function showTaskNote(task: TaskItem, event: MouseEvent | FocusEvent) {
  const note = task.note?.trim()
  if (!note) {
    activeTaskNoteId.value = null
    return
  }

  const row = event.currentTarget
  if (row instanceof HTMLElement) {
    const position = previewPosition(row, note)
    activeTaskNoteTop.value = position.top
    activeTaskNoteLeft.value = position.left
  }

  activeTaskNoteId.value = task.id
  activeReminderNoteId.value = null
}

function hideTaskNote(taskId: number) {
  if (activeTaskNoteId.value === taskId) {
    activeTaskNoteId.value = null
  }
}

function handleTaskFocusOut(event: FocusEvent, taskId: number) {
  const nextTarget = event.relatedTarget
  const currentTarget = event.currentTarget
  if (
    nextTarget instanceof Node &&
    currentTarget instanceof HTMLElement &&
    currentTarget.contains(nextTarget)
  ) {
    return
  }

  hideTaskNote(taskId)
}

function closeTaskMenu() {
  openTaskMenuId.value = null
  openReminderMenuId.value = null
}

function closeTaskMenuAndRestoreFocus() {
  const triggerId = openTaskMenuId.value !== null
    ? taskMenuTriggerId(openTaskMenuId.value)
    : openReminderMenuId.value !== null
      ? reminderMenuTriggerId(openReminderMenuId.value)
      : null

  closeTaskMenu()
  if (!triggerId) return

  void nextTick(() => document.getElementById(triggerId)?.focus())
}

function updateTaskMenuPosition(trigger: HTMLElement) {
  const rect = trigger.getBoundingClientRect()
  const menuWidth = 96
  const menuHeight = 76
  const gap = 6
  const viewportPadding = 8
  const container = trigger.closest<HTMLElement>('.sidebar-panel')
  const containerRect = container?.getBoundingClientRect()
  const boundaryLeft = Math.max(
    viewportPadding,
    (containerRect?.left ?? viewportPadding) + 4,
  )
  const boundaryRight = Math.min(
    window.innerWidth - viewportPadding,
    (containerRect?.right ?? window.innerWidth - viewportPadding) - 4,
  )
  const preferredLeft = rect.right + gap
  const fallbackLeft = rect.left - gap - menuWidth
  const hasRightSpace = preferredLeft + menuWidth <= boundaryRight
  const hasLeftSpace = fallbackLeft >= boundaryLeft

  taskMenuTop.value = Math.max(
    viewportPadding,
    Math.min(
      rect.top + rect.height / 2 - menuHeight / 2,
      window.innerHeight - viewportPadding - menuHeight,
    ),
  )
  taskMenuLeft.value = hasRightSpace
    ? preferredLeft
    : hasLeftSpace
      ? fallbackLeft
      : Math.max(boundaryLeft, boundaryRight - menuWidth)
}

function toggleTaskMenu(taskId: number, event: MouseEvent) {
  if (openTaskMenuId.value === taskId) {
    closeTaskMenu()
    return
  }

  const trigger = event.currentTarget
  if (trigger instanceof HTMLElement) {
    updateTaskMenuPosition(trigger)
  }

  activeTaskNoteId.value = null
  openReminderMenuId.value = null
  openTaskMenuId.value = taskId
}

function toggleReminderMenu(reminderId: number, event: MouseEvent) {
  if (openReminderMenuId.value === reminderId) {
    closeTaskMenu()
    return
  }

  const trigger = event.currentTarget
  if (trigger instanceof HTMLElement) {
    updateTaskMenuPosition(trigger)
  }

  activeReminderNoteId.value = null
  openTaskMenuId.value = null
  openReminderMenuId.value = reminderId
}

function toggleTaskListExpansion() {
  isTaskListExpanded.value = !isTaskListExpanded.value
  closeTaskMenu()
  activeTaskNoteId.value = null
  activeReminderNoteId.value = null
}

function toggleReminderListExpansion() {
  isReminderListExpanded.value = !isReminderListExpanded.value
  closeTaskMenu()
  activeTaskNoteId.value = null
  activeReminderNoteId.value = null
}

function editSidebarTask(task: TaskItem) {
  closeTaskMenu()
  editingTaskId.value = task.id
}

function removeSidebarTask(task: TaskItem) {
  closeTaskMenu()
  activeTaskNoteId.value = null
  activeReminderNoteId.value = null
  pendingDeleteTask.value = task
}

function closeDeleteDialog() {
  pendingDeleteTask.value = null
}

function confirmDeleteTask() {
  const task = pendingDeleteTask.value
  if (!task) return

  deleteTask(task.id)
  pendingDeleteTask.value = null
}

function removeSidebarReminder(reminder: ReminderItem) {
  closeTaskMenu()
  activeReminderNoteId.value = null
  pendingDeleteReminder.value = reminder
}

function editSidebarReminder(reminder: ReminderItem) {
  closeTaskMenu()
  isAddReminderModalOpen.value = false
  editingReminderId.value = reminder.id
  setSelectedDateKey(reminder.date)
}

function closeReminderDeleteDialog() {
  pendingDeleteReminder.value = null
}

function confirmDeleteReminder() {
  const reminder = pendingDeleteReminder.value
  if (!reminder) return

  deleteReminder(reminder.id)
  pendingDeleteReminder.value = null
}

function refreshCurrentDate() {
  const previousDate = currentDate.value
  const nextDate = new Date()
  const wasViewingCurrentMonth =
    visibleMonth.value.getFullYear() === previousDate.getFullYear() &&
    visibleMonth.value.getMonth() === previousDate.getMonth()

  currentDate.value = nextDate

  if (wasViewingCurrentMonth) {
    visibleMonth.value = new Date(nextDate.getFullYear(), nextDate.getMonth(), 1)
  }
}

onMounted(() => {
  calendarTimer = window.setInterval(refreshCurrentDate, 60_000)
  document.addEventListener('click', closeTaskMenu)
  document.addEventListener('scroll', closeTaskMenu, true)
  window.addEventListener('resize', closeTaskMenu)
})

onBeforeUnmount(() => {
  if (calendarTimer !== undefined) {
    window.clearInterval(calendarTimer)
  }

  document.removeEventListener('click', closeTaskMenu)
  document.removeEventListener('scroll', closeTaskMenu, true)
  window.removeEventListener('resize', closeTaskMenu)
})
</script>

<template>
  <aside class="sidebar-widgets" aria-label="側邊工具">
    <section class="sidebar-panel calendar-panel" aria-label="月曆">
      <header class="calendar-header">
        <button type="button" class="calendar-nav-btn" aria-label="上一個月" @click="shiftMonth(-1)">
          <span aria-hidden="true">&lsaquo;</span>
        </button>
        <h2>{{ monthLabel }}</h2>
        <button type="button" class="calendar-nav-btn" aria-label="下一個月" @click="shiftMonth(1)">
          <span aria-hidden="true">&rsaquo;</span>
        </button>
      </header>

      <div class="calendar-grid calendar-weekdays" aria-hidden="true">
        <span v-for="day in weekDays" :key="day">{{ day }}</span>
      </div>

      <div class="calendar-grid calendar-days">
        <button
          v-for="cell in calendarCells"
          :key="cell.key"
          type="button"
          class="calendar-day"
          :class="{ active: cell.isToday, muted: !cell.isCurrentMonth }"
          :disabled="!cell.isCurrentMonth"
          :aria-label="cell.isCurrentMonth ? `查看 ${cell.dateKey} 的行程` : undefined"
          @click="openDateSchedule(cell.dateKey)"
        >
          {{ cell.label }}
        </button>
      </div>
    </section>

    <SidebarReminderPanel
      :reminders="sidebarReminders"
      :expanded="isReminderListExpanded"
      :open-menu-id="openReminderMenuId"
      :menu-style="taskMenuStyle"
      :active-note="activeReminderNote"
      :active-note-style="activeReminderNoteStyle"
      @toggle-expansion="toggleReminderListExpansion"
      @add="openTodayReminderForm"
      @show-note="showReminderNote"
      @hide-note="hideReminderNote"
      @focus-out="handleReminderFocusOut"
      @toggle-menu="toggleReminderMenu"
      @close-menu="closeTaskMenuAndRestoreFocus"
      @edit="editSidebarReminder"
      @remove="removeSidebarReminder"
    />

    <SidebarTaskPanel
      :tasks="sidebarTasks"
      :expanded="isTaskListExpanded"
      :open-menu-id="openTaskMenuId"
      :menu-style="taskMenuStyle"
      :active-note="activeTaskNote"
      :active-note-style="activeTaskNoteStyle"
      @toggle-expansion="toggleTaskListExpansion"
      @add="isAddTaskModalOpen = true"
      @toggle-task="toggleTask"
      @show-note="showTaskNote"
      @hide-note="hideTaskNote"
      @focus-out="handleTaskFocusOut"
      @toggle-menu="toggleTaskMenu"
      @close-menu="closeTaskMenuAndRestoreFocus"
      @edit="editSidebarTask"
      @remove="removeSidebarTask"
    />

    <button type="button" class="sidebar-panel note-panel" @click="isInspirationListOpen = true">
      記下想做的事
    </button>

    <DateScheduleModal
      v-if="selectedDateKey"
      :date-key="selectedDateKey"
      :start-with-new-reminder="isAddReminderModalOpen"
      :edit-reminder-id="editingReminderId"
      @update:date-key="setSelectedDateKey"
      @close="closeDateSchedule"
    />

    <AddTaskModal v-if="isAddTaskModalOpen" @close="isAddTaskModalOpen = false" />
    <AddTaskModal
      v-if="editingTaskId !== null"
      :edit-task-id="editingTaskId"
      @close="editingTaskId = null"
    />
    <InspirationListModal v-if="isInspirationListOpen" @close="isInspirationListOpen = false" />
    <TaskDeleteConfirmDialog
      v-if="pendingDeleteTask"
      :item="pendingDeleteTask"
      :return-focus-id="taskMenuTriggerId(pendingDeleteTask.id)"
      :confirm-focus-id="taskPanelHeadingId"
      @cancel="closeDeleteDialog"
      @confirm="confirmDeleteTask"
    />
    <TaskDeleteConfirmDialog
      v-if="pendingDeleteReminder"
      :item="pendingDeleteReminder"
      kind="reminder"
      :return-focus-id="reminderMenuTriggerId(pendingDeleteReminder.id)"
      :confirm-focus-id="reminderPanelHeadingId"
      @cancel="closeReminderDeleteDialog"
      @confirm="confirmDeleteReminder"
    />
  </aside>
</template>
