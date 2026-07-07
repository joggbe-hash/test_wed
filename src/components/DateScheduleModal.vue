<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, shallowRef, useTemplateRef } from 'vue'
import {
  priorityMeta,
  taskImportanceCount,
  type Priority,
  type ReminderItem,
  type TaskItem,
  useScheduleMock,
} from '../composables/useScheduleMock'
import { formatLocalDateKey } from '../utils/date'
import DailyTaskPrompt from './DailyTaskPrompt.vue'

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
  addReminder,
  updateTask,
  deleteTask,
  updateReminder,
  deleteReminder,
} = useScheduleMock()

type EditingKind = 'task' | 'reminder' | 'new-reminder'

const editingKind = shallowRef<EditingKind | null>(null)
const editingId = shallowRef<number | null>(null)
const isDailyTaskPromptOpen = shallowRef(false)
const taskPromptEditId = shallowRef<number | null>(null)
const editForm = reactive({
  title: '',
  note: '',
  priority: 'medium' as Priority,
  date: props.dateKey,
  startTime: '08:00',
  endTime: '09:00',
})
const startTimeInput = useTemplateRef<HTMLInputElement>('startTimeInput')
const endTimeInput = useTemplateRef<HTMLInputElement>('endTimeInput')

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
const editPanelTitle = computed(() => {
  if (editingKind.value === 'new-reminder') return '新增提醒'
  if (editingKind.value === 'reminder') return '編輯提醒'
  return '編輯任務'
})

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

function updateReminderStartTime(event: Event) {
  const input = event.target
  if (!(input instanceof HTMLInputElement)) return

  editForm.startTime = input.value
  if (!editForm.endTime || editForm.endTime <= input.value) {
    editForm.endTime = defaultReminderEndTime(input.value)
  }
}

function openTimePicker(input: HTMLInputElement | null) {
  if (!input) return

  input.focus()
  try {
    input.showPicker?.()
  } catch {
    // Some browsers only allow native pickers from direct user activation.
  }
}

function openStartTimePicker() {
  openTimePicker(startTimeInput.value)
}

function openEndTimePicker() {
  openTimePicker(endTimeInput.value)
}

function reminderTimeRange(reminder: ReminderItem) {
  return reminder.endTime ? `${reminder.time} - ${reminder.endTime}` : reminder.time
}

function handleHorizontalRowWheel(event: WheelEvent) {
  const row = event.currentTarget
  if (!(row instanceof HTMLElement)) return

  event.preventDefault()
  row.scrollLeft += event.deltaY
}

function shiftDate(offset: number) {
  const nextDate = parseDateKey(props.dateKey)
  nextDate.setDate(nextDate.getDate() + offset)
  cancelEdit()
  emit('update:dateKey', formatLocalDateKey(nextDate))
}

function startAddReminder() {
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
  editingKind.value = 'reminder'
  editingId.value = reminder.id
  editForm.title = reminder.title
  editForm.note = reminder.note
  editForm.priority = 'medium'
  editForm.date = reminder.date
  editForm.startTime = reminder.time
  editForm.endTime = reminder.endTime ?? defaultReminderEndTime(reminder.time)
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

  if (kind === 'new-reminder') {
    addReminder({
      title,
      date,
      time: startTime,
      endTime,
      note: editForm.note.trim(),
    })
  } else {
    const id = editingId.value
    if (!id) return false

    updateReminder(id, {
      title,
      date,
      time: startTime,
      endTime,
      note: editForm.note.trim(),
    })
  }

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

  updateTask(id, {
    title,
    note: editForm.note.trim(),
    priority: editForm.priority,
  })

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

const handleKeyDown = (e: KeyboardEvent) => {
  if (isDailyTaskPromptOpen.value) return

  if (e.key === 'Escape') {
    if (editingKind.value) {
      cancelEdit()
      return
    }

    emit('close')
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
})
</script>

<template>
  <Teleport to="body">
    <div class="date-schedule-backdrop" role="presentation" @click.self="emit('close')">
      <section
        class="date-schedule-modal"
        :class="{
          'date-schedule-modal-reminder-editing': isReminderEditorOpen,
          'date-schedule-modal-task-prompt-open': isDailyTaskPromptOpen,
        }"
        role="dialog"
        aria-modal="true"
        aria-labelledby="date-schedule-title"
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

          <button type="button" class="date-schedule-close" aria-label="關閉" @click="emit('close')">
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

        <section ref="reminderRow" class="date-schedule-reminders" aria-label="提醒" @wheel="handleHorizontalRowWheel">
          <article v-for="reminder in dateReminders" :key="reminder.id" class="date-reminder-card">
            <h3>{{ reminder.title }}</h3>
            <p class="date-reminder-note">{{ reminder.note || '（無備註）' }}</p>
            <p class="date-reminder-time">{{ reminderTimeRange(reminder) }}</p>

            <div class="date-card-actions">
              <button type="button" @click="startEditReminder(reminder)">編輯</button>
              <button type="button" @click="removeReminder(reminder.id)">刪除</button>
            </div>
          </article>

          <button
            type="button"
            class="date-add-reminder-card"
            :class="{ 'date-add-reminder-card-empty': dateReminders.length === 0 }"
            @click="startAddReminder"
          >
            <span class="date-add-reminder-empty-label">這天沒有提醒</span>
            <span class="date-add-reminder-add-label">新增提醒</span>
            <strong aria-hidden="true">+</strong>
          </button>
        </section>

        <section ref="taskRow" class="date-schedule-tasks" aria-label="任務" @wheel="handleHorizontalRowWheel">
          <article v-for="task in dateTasks" :key="task.id" class="date-task-card">
            <div class="date-task-priority" :aria-label="priorityMeta[task.priority].label">
              <span
                v-for="dot in taskImportanceCount(task)"
                :key="dot"
                class="filled"
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

          <button
            type="button"
            class="date-add-task-card"
            :class="{ 'date-add-task-card-empty': dateTasks.length === 0 }"
            @click="openAddTaskPrompt"
          >
            <span class="date-add-task-empty-label">這天還沒有任務</span>
            <span class="date-add-task-add-label">新增任務</span>
            <strong aria-hidden="true">+</strong>
          </button>
        </section>
      </section>

      <form
        v-if="isReminderEditorOpen"
        class="reminder-edit-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="reminder-edit-title"
        @submit.prevent="saveEdit"
      >
        <div class="reminder-edit-calendar-mark" aria-hidden="true">
          <span>{{ reminderEditorDay }}</span>
        </div>

        <header class="reminder-edit-header">
          <h3 id="reminder-edit-title">{{ editPanelTitle }}</h3>
          <button type="button" class="reminder-edit-close" aria-label="關閉" @click="cancelEdit">
            <span aria-hidden="true">&times;</span>
          </button>
        </header>

        <div class="reminder-edit-body">
          <label class="reminder-edit-field">
            <span>標題：</span>
            <input v-model="editForm.title" type="text" placeholder="節日、行程..." required autofocus>
          </label>

          <label class="reminder-edit-field reminder-edit-note-field">
            <span>備註說明：</span>
            <span v-if="!editForm.note" class="reminder-note-optional" aria-hidden="true">（選填）</span>
            <textarea v-model="editForm.note" rows="3"></textarea>
          </label>

          <section class="reminder-time-panel" aria-label="提醒時間">
            <span class="reminder-time-icon" aria-hidden="true"></span>

            <div class="reminder-time-card">
              <label class="reminder-date-picker">
                <span>{{ reminderEditorDateLabel }}</span>
                <input v-model="editForm.date" type="date" aria-label="提醒日期">
              </label>
              <label class="reminder-clock-picker" @click="openStartTimePicker">
                <strong>{{ editForm.startTime }}</strong>
                <input
                  ref="startTimeInput"
                  v-model="editForm.startTime"
                  type="time"
                  aria-label="開始時間"
                  @change="updateReminderStartTime"
                >
              </label>
            </div>

            <span class="reminder-time-arrow" aria-hidden="true">&rarr;</span>

            <div class="reminder-time-card">
              <label class="reminder-date-picker">
                <span>{{ reminderEditorDateLabel }}</span>
                <input v-model="editForm.date" type="date" aria-label="提醒日期">
              </label>
              <label class="reminder-clock-picker" @click="openEndTimePicker">
                <strong>{{ editForm.endTime }}</strong>
                <input ref="endTimeInput" v-model="editForm.endTime" type="time" aria-label="結束時間">
              </label>
            </div>
          </section>

          <div class="reminder-edit-actions">
            <button
              v-if="editingKind === 'new-reminder'"
              type="button"
              class="reminder-save-next"
              :disabled="!editForm.title.trim()"
              @click="saveReminderAndCreateNext"
            >
              儲存並新增下一項
            </button>

            <div class="reminder-edit-action-row">
              <button type="submit" class="reminder-save" :disabled="!editForm.title.trim()">儲存</button>
              <button type="button" class="reminder-cancel" @click="cancelEdit">取消</button>
            </div>
          </div>
        </div>
      </form>

      <DailyTaskPrompt
        v-if="isDailyTaskPromptOpen"
        :task-date="dateKey"
        :edit-task-id="taskPromptEditId"
        @close="closeTaskPrompt"
      />
    </div>
  </Teleport>
</template>
