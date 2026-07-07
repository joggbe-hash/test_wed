<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import AddTaskModal from './AddTaskModal.vue'
import DateScheduleModal from './DateScheduleModal.vue'
import InspirationListModal from './InspirationListModal.vue'
import TaskDeleteConfirmDialog from './TaskDeleteConfirmDialog.vue'
import { type ReminderItem, type TaskItem, useScheduleMock } from '../composables/useScheduleMock'

const weekDays = ['SUN', 'MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT']
const currentDate = shallowRef(new Date())
const visibleMonth = shallowRef(new Date(currentDate.value.getFullYear(), currentDate.value.getMonth(), 1))
const selectedDateKey = ref<string | null>(null)
const isAddTaskModalOpen = shallowRef(false)
const isInspirationListOpen = shallowRef(false)
const isReminderListExpanded = shallowRef(false)
const isTaskListExpanded = shallowRef(false)
const openTaskMenuId = shallowRef<number | null>(null)
const taskMenuTop = shallowRef(0)
const taskMenuLeft = shallowRef(0)
const editingTaskId = shallowRef<number | null>(null)
const pendingDeleteTask = shallowRef<TaskItem | null>(null)
const activeReminderNoteId = shallowRef<number | null>(null)
const activeReminderNoteTop = shallowRef(0)
const activeReminderNoteLeft = shallowRef(0)
const activeTaskNoteId = shallowRef<number | null>(null)
const activeTaskNoteTop = shallowRef(0)
const activeTaskNoteLeft = shallowRef(0)
const { todayKey, todayTasks, todayReminders, toggleTask, deleteTask } = useScheduleMock()

const reminderListId = 'sidebar-reminder-list'
const taskListId = 'sidebar-task-list'
const sidebarReminders = computed(() => todayReminders.value)
const sidebarTasks = computed(() => todayTasks.value)
const activeReminderNote = computed(() => {
  const reminder = sidebarReminders.value.find((item) => item.id === activeReminderNoteId.value)
  return reminder?.note?.trim() ?? ''
})
const activeTaskNote = computed(() => {
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
  setSelectedDateKey(dateKey)
}

function taskCheckboxId(taskId: number) {
  return `sidebar-task-${taskId}`
}

function taskMenuId(taskId: number) {
  return `sidebar-task-menu-${taskId}`
}

function reminderTimeRange(reminder: ReminderItem) {
  return reminder.endTime ? `${reminder.time} - ${reminder.endTime}` : reminder.time
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
}

function updateTaskMenuPosition(trigger: HTMLElement) {
  const rect = trigger.getBoundingClientRect()
  const menuWidth = 96
  const menuHeight = 76
  const gap = 6
  const viewportPadding = 8
  const shouldOpenAbove = rect.bottom + gap + menuHeight > window.innerHeight - viewportPadding

  taskMenuTop.value = shouldOpenAbove
    ? Math.max(viewportPadding, rect.top - gap - menuHeight)
    : Math.min(rect.bottom + gap, window.innerHeight - viewportPadding - menuHeight)
  taskMenuLeft.value = Math.max(
    viewportPadding,
    Math.min(rect.right - menuWidth, window.innerWidth - viewportPadding - menuWidth),
  )
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
  openTaskMenuId.value = taskId
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

    <section class="sidebar-panel reminder-panel">
      <button
        type="button"
        class="sidebar-panel-heading"
        :aria-expanded="isReminderListExpanded"
        :aria-controls="reminderListId"
        :aria-label="isReminderListExpanded ? '收合今日提醒清單' : '展開今日提醒清單'"
        @click="toggleReminderListExpansion"
      >
        今日提醒
      </button>
      <div
        :id="reminderListId"
        class="task-list sidebar-reminder-list"
        :class="{ 'task-list-expanded': isReminderListExpanded }"
      >
        <div
          v-if="sidebarReminders.length === 0"
          class="task-row sidebar-reminder-row sidebar-reminder-empty"
        >
          <span class="sidebar-reminder-title">:p</span>
        </div>
        <template v-else>
          <div
            v-for="reminder in sidebarReminders"
            :key="reminder.id"
            class="task-row sidebar-reminder-row"
            :tabindex="reminder.note.trim() ? 0 : undefined"
            @mouseenter="showReminderNote(reminder, $event)"
            @mouseleave="hideReminderNote(reminder.id)"
            @focusin="showReminderNote(reminder, $event)"
            @focusout="handleReminderFocusOut($event, reminder.id)"
          >
            <span class="sidebar-reminder-time" :title="reminderTimeRange(reminder)">
              {{ reminderTimeRange(reminder) }}
            </span>
            <span class="sidebar-reminder-title" :title="reminder.title">
              {{ reminder.title }}
            </span>
          </div>
        </template>
      </div>
      <div
        v-if="activeReminderNote"
        class="task-note-preview"
        :style="activeReminderNoteStyle"
        role="tooltip"
        aria-live="polite"
      >
        {{ activeReminderNote }}
      </div>
    </section>

    <section class="sidebar-panel task-panel">
      <button
        type="button"
        class="sidebar-panel-heading"
        :aria-expanded="isTaskListExpanded"
        :aria-controls="taskListId"
        :aria-label="isTaskListExpanded ? '收合今日任務清單' : '展開今日任務清單'"
        @click="toggleTaskListExpansion"
      >
        今日任務
      </button>
      <div
        :id="taskListId"
        class="task-list"
        :class="{ 'task-list-expanded': isTaskListExpanded }"
      >
        <div
          v-for="task in sidebarTasks"
          :key="task.id"
          class="task-row"
          @mouseenter="showTaskNote(task, $event)"
          @mouseleave="hideTaskNote(task.id)"
          @focusin="showTaskNote(task, $event)"
          @focusout="handleTaskFocusOut($event, task.id)"
        >
          <input
            :id="taskCheckboxId(task.id)"
            type="checkbox"
            :checked="task.completed"
            :aria-label="`${task.title} 完成狀態`"
            @change="toggleTask(task.id)"
          >
          <label class="task-row-title" :for="taskCheckboxId(task.id)" :title="task.title">
            {{ task.title }}
          </label>
          <div class="task-row-actions" @click.stop>
            <button
              type="button"
              class="task-menu-trigger"
              :aria-label="`${task.title} 更多操作`"
              aria-haspopup="menu"
              :aria-expanded="openTaskMenuId === task.id"
              :aria-controls="taskMenuId(task.id)"
              @click="toggleTaskMenu(task.id, $event)"
              @keydown.escape.stop.prevent="closeTaskMenu"
            >
              <span aria-hidden="true">⋮</span>
            </button>
            <div
              v-if="openTaskMenuId === task.id"
              :id="taskMenuId(task.id)"
              class="task-menu-panel"
              :style="taskMenuStyle"
              role="menu"
              @keydown.escape.stop.prevent="closeTaskMenu"
            >
              <button type="button" role="menuitem" @click="editSidebarTask(task)">編輯</button>
              <button type="button" role="menuitem" @click="removeSidebarTask(task)">刪除</button>
            </div>
          </div>
        </div>
      </div>
      <div
        v-if="activeTaskNote"
        class="task-note-preview"
        :style="activeTaskNoteStyle"
        role="tooltip"
        aria-live="polite"
      >
        {{ activeTaskNote }}
      </div>
      <button type="button" class="add-task-btn" @click="isAddTaskModalOpen = true">+新增任務</button>
    </section>

    <button type="button" class="sidebar-panel note-panel" @click="isInspirationListOpen = true">
      記下想做的事
    </button>

    <DateScheduleModal
      v-if="selectedDateKey"
      :date-key="selectedDateKey"
      @update:date-key="setSelectedDateKey"
      @close="setSelectedDateKey(null)"
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
      :task="pendingDeleteTask"
      @cancel="closeDeleteDialog"
      @confirm="confirmDeleteTask"
    />
  </aside>
</template>
