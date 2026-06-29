<script setup lang="ts">
import { computed, reactive, shallowRef } from 'vue'
import {
  priorityMeta,
  type Priority,
  type ReminderItem,
  type TaskItem,
  useScheduleMock,
} from '../composables/useScheduleMock'
import { formatLocalDateKey } from '../utils/date'

const props = defineProps<{
  dateKey: string
}>()

const emit = defineEmits<{
  close: []
  'update:dateKey': [value: string]
}>()

const {
  sortedTasks,
  sortedReminders,
  updateTask,
  deleteTask,
  updateReminder,
  deleteReminder,
} = useScheduleMock()

type EditingKind = 'task' | 'reminder'

const editingKind = shallowRef<EditingKind | null>(null)
const editingId = shallowRef<number | null>(null)
const editForm = reactive({
  title: '',
  note: '',
  priority: 'medium' as Priority,
})

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

const dateReminders = computed(() =>
  sortedReminders.value.filter((reminder) => reminder.date === props.dateKey),
)

const dateTasks = computed(() =>
  sortedTasks.value.filter((task) => task.date === props.dateKey),
)

const isEditing = computed(() => editingKind.value !== null && editingId.value !== null)
const editPanelTitle = computed(() => (editingKind.value === 'reminder' ? '編輯提醒' : '編輯任務'))

function shiftDate(offset: number) {
  const nextDate = parseDateKey(props.dateKey)
  nextDate.setDate(nextDate.getDate() + offset)
  cancelEdit()
  emit('update:dateKey', formatLocalDateKey(nextDate))
}

function priorityDots(priority: Priority) {
  const filled = priority === 'high' ? 5 : priority === 'medium' ? 3 : 1
  return Array.from({ length: 5 }, (_, index) => index < filled)
}

function startEditReminder(reminder: ReminderItem) {
  editingKind.value = 'reminder'
  editingId.value = reminder.id
  editForm.title = reminder.title
  editForm.note = reminder.note
  editForm.priority = 'medium'
}

function startEditTask(task: TaskItem) {
  editingKind.value = 'task'
  editingId.value = task.id
  editForm.title = task.title
  editForm.note = task.note ?? ''
  editForm.priority = task.priority
}

function cancelEdit() {
  editingKind.value = null
  editingId.value = null
}

function saveEdit() {
  const id = editingId.value
  const kind = editingKind.value
  const title = editForm.title.trim()
  if (!id || !kind || !title) return

  if (kind === 'reminder') {
    updateReminder(id, {
      title,
      note: editForm.note.trim(),
    })
  } else {
    updateTask(id, {
      title,
      note: editForm.note.trim(),
      priority: editForm.priority,
    })
  }

  cancelEdit()
}

function removeReminder(reminderId: number) {
  if (editingKind.value === 'reminder' && editingId.value === reminderId) cancelEdit()
  deleteReminder(reminderId)
}

function removeTask(taskId: number) {
  if (editingKind.value === 'task' && editingId.value === taskId) cancelEdit()
  deleteTask(taskId)
}
</script>

<template>
  <Teleport to="body">
    <div class="date-schedule-backdrop" role="presentation" @click.self="emit('close')">
      <section class="date-schedule-modal" role="dialog" aria-modal="true" aria-labelledby="date-schedule-title">
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

          <button type="button" class="date-schedule-close" aria-label="關閉" @click="emit('close')">
            <span aria-hidden="true">&times;</span>
          </button>
        </header>

        <form v-if="isEditing" class="date-edit-panel" @submit.prevent="saveEdit">
          <h3>{{ editPanelTitle }}</h3>

          <label>
            <span>名稱</span>
            <input v-model="editForm.title" type="text" required />
          </label>

          <label class="date-edit-note">
            <span>備註</span>
            <textarea v-model="editForm.note" rows="2"></textarea>
          </label>

          <label v-if="editingKind === 'task'">
            <span>重要度</span>
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

        <section class="date-schedule-reminders" aria-label="提醒">
          <article v-for="reminder in dateReminders" :key="reminder.id" class="date-reminder-card">
            <h3>{{ reminder.title }}</h3>
            <p>{{ reminder.note || '（無備註）' }}</p>

            <div class="date-card-actions">
              <button type="button" @click="startEditReminder(reminder)">編輯</button>
              <button type="button" @click="removeReminder(reminder.id)">刪除</button>
            </div>
          </article>

          <p v-if="dateReminders.length === 0" class="date-schedule-empty">
            這天沒有提醒
          </p>
        </section>

        <section class="date-schedule-tasks" aria-label="任務">
          <article v-for="task in dateTasks" :key="task.id" class="date-task-card">
            <div class="date-task-priority" :aria-label="priorityMeta[task.priority].label">
              <span
                v-for="(filled, index) in priorityDots(task.priority)"
                :key="index"
                :class="{ filled }"
                aria-hidden="true"
              ></span>
            </div>

            <h3>{{ task.title }}</h3>
            <p>{{ task.note || '（無備註）' }}</p>

            <div class="date-card-actions date-task-actions">
              <button type="button" @click="startEditTask(task)">編輯</button>
              <button type="button" @click="removeTask(task.id)">刪除</button>
            </div>
          </article>

          <p v-if="dateTasks.length === 0" class="date-schedule-empty date-schedule-empty-tasks">
            這天沒有任務
          </p>
        </section>
      </section>
    </div>
  </Teleport>
</template>
