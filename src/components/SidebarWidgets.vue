<script setup lang="ts">
import { computed, shallowRef } from 'vue'
import AddTaskModal from './AddTaskModal.vue'
import DateScheduleModal from './DateScheduleModal.vue'
import InspirationListModal from './InspirationListModal.vue'
import TaskDeleteConfirmDialog from './TaskDeleteConfirmDialog.vue'
import SidebarReminderPanel from './SidebarReminderPanel.vue'
import SidebarTaskPanel from './SidebarTaskPanel.vue'
import { type ReminderItem, type TaskItem, useSchedule } from '../composables/useSchedule'
import { useSidebarCalendar } from '../features/schedule/useSidebarCalendar'
import { useSidebarInteractions } from '../features/schedule/useSidebarInteractions'

const isAddTaskModalOpen = shallowRef(false)
const isAddReminderModalOpen = shallowRef(false)
const isInspirationListOpen = shallowRef(false)
const editingTaskId = shallowRef<number | null>(null)
const editingReminderId = shallowRef<number | null>(null)
const pendingDeleteTask = shallowRef<TaskItem | null>(null)
const pendingDeleteReminder = shallowRef<ReminderItem | null>(null)
const {
  todayKey,
  todayTasks,
  todayReminders,
  toggleTask,
  deleteTask,
  deleteReminder,
} = useSchedule()

const sidebarReminders = computed(() => todayReminders.value)
const sidebarTasks = computed(() => todayTasks.value)
const {
  weekDays,
  selectedDateKey,
  monthLabel,
  calendarCells,
  shiftMonth,
  setSelectedDateKey,
} = useSidebarCalendar(todayKey)
const {
  isReminderListExpanded,
  isTaskListExpanded,
  openTaskMenuId,
  openReminderMenuId,
  activeReminderNoteId,
  activeTaskNoteId,
  reminderPanelHeadingId,
  taskPanelHeadingId,
  activeReminderNote,
  activeTaskNote,
  activeReminderNoteStyle,
  activeTaskNoteStyle,
  taskMenuStyle,
  taskMenuTriggerId,
  reminderMenuTriggerId,
  showReminderNote,
  hideReminderNote,
  handleReminderFocusOut,
  showTaskNote,
  hideTaskNote,
  handleTaskFocusOut,
  closeTaskMenu,
  closeTaskMenuAndRestoreFocus,
  toggleTaskMenu,
  toggleReminderMenu,
  toggleTaskListExpansion,
  toggleReminderListExpansion,
} = useSidebarInteractions(sidebarTasks, sidebarReminders)

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

function confirmDeleteTask() {
  if (!pendingDeleteTask.value) return
  deleteTask(pendingDeleteTask.value.id)
  pendingDeleteTask.value = null
}

function editSidebarReminder(reminder: ReminderItem) {
  closeTaskMenu()
  isAddReminderModalOpen.value = false
  editingReminderId.value = reminder.id
  setSelectedDateKey(reminder.date)
}

function removeSidebarReminder(reminder: ReminderItem) {
  closeTaskMenu()
  activeReminderNoteId.value = null
  pendingDeleteReminder.value = reminder
}

function confirmDeleteReminder() {
  if (!pendingDeleteReminder.value) return
  deleteReminder(pendingDeleteReminder.value.id)
  pendingDeleteReminder.value = null
}
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
      @cancel="pendingDeleteTask = null"
      @confirm="confirmDeleteTask"
    />
    <TaskDeleteConfirmDialog
      v-if="pendingDeleteReminder"
      :item="pendingDeleteReminder"
      kind="reminder"
      :return-focus-id="reminderMenuTriggerId(pendingDeleteReminder.id)"
      :confirm-focus-id="reminderPanelHeadingId"
      @cancel="pendingDeleteReminder = null"
      @confirm="confirmDeleteReminder"
    />
  </aside>
</template>
