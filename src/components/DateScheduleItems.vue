<script setup lang="ts">
import { priorityMeta, taskImportanceCount, type ReminderItem, type TaskItem } from '../composables/useSchedule'

defineProps<{ reminders: ReminderItem[], tasks: TaskItem[] }>()
const emit = defineEmits<{
  addReminder: []
  addTask: []
  editReminder: [reminder: ReminderItem]
  editTask: [task: TaskItem]
  deleteReminder: [reminder: ReminderItem]
  deleteTask: [task: TaskItem]
}>()

function reminderTimeRange(reminder: ReminderItem) {
  return reminder.endTime ? `${reminder.time} - ${reminder.endTime}` : reminder.time
}

function handleHorizontalWheel(event: WheelEvent) {
  const row = event.currentTarget
  if (!(row instanceof HTMLElement)) return
  event.preventDefault()
  row.scrollLeft += event.deltaY
}
</script>

<template>
  <section class="date-schedule-reminders" aria-label="提醒" @wheel="handleHorizontalWheel">
    <article v-for="reminder in reminders" :key="reminder.id" class="date-reminder-card">
      <h3>{{ reminder.title }}</h3>
      <p class="date-reminder-note">{{ reminder.note || '（無備註）' }}</p>
      <p class="date-reminder-time">{{ reminderTimeRange(reminder) }}</p>
      <div class="date-card-actions">
        <button type="button" :data-reminder-edit-id="reminder.id" @click="emit('editReminder', reminder)">編輯</button>
        <button type="button" @click="emit('deleteReminder', reminder)">刪除</button>
      </div>
    </article>
    <button type="button" data-add-reminder class="date-add-reminder-card" :class="{ 'date-add-reminder-card-empty': reminders.length === 0 }" @click="emit('addReminder')">
      <span class="date-add-reminder-empty-label">這天沒有提醒</span>
      <span class="date-add-reminder-add-label">新增提醒</span>
      <strong aria-hidden="true">+</strong>
    </button>
  </section>

  <section class="date-schedule-tasks" aria-label="任務" @wheel="handleHorizontalWheel">
    <article v-for="task in tasks" :key="task.id" class="date-task-card">
      <div class="date-task-priority" :aria-label="priorityMeta[task.priority].label">
        <span v-for="dot in taskImportanceCount(task)" :key="dot" class="filled" aria-hidden="true"></span>
      </div>
      <h3>{{ task.title }}</h3>
      <p>{{ task.note || '（無備註）' }}</p>
      <div class="date-card-actions date-task-actions">
        <button type="button" @click="emit('editTask', task)">編輯</button>
        <button type="button" @click="emit('deleteTask', task)">刪除</button>
      </div>
    </article>
    <button type="button" class="date-add-task-card" :class="{ 'date-add-task-card-empty': tasks.length === 0 }" @click="emit('addTask')">
      <span class="date-add-task-empty-label">這天還沒有任務</span>
      <span class="date-add-task-add-label">新增任務</span>
      <strong aria-hidden="true">+</strong>
    </button>
  </section>
</template>
