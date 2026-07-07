<script setup lang="ts">
import { computed, reactive, shallowRef, onMounted, onUnmounted, useTemplateRef, watch } from 'vue'
import {
  markDailyTaskPromptHandled,
} from '../composables/useDailyTaskPrompt'
import {
  priorityToImportance,
  taskImportanceCount,
  type Priority,
  type TaskImportance,
  useScheduleMock,
} from '../composables/useScheduleMock'

type Importance = TaskImportance
type ImportanceSelection = TaskImportance
type PromptMode = 'form' | 'list'

const importanceOptions: Importance[] = [1, 2, 3, 4, 5]
const props = defineProps<{
  editTaskId?: number | null
  initialNote?: string
  initialTitle?: string
  closeAfterSubmit?: boolean
  taskDate?: string
}>()
const emit = defineEmits<{
  close: []
}>()
const mode = shallowRef<PromptMode>('form')
const { addTask, sortedTasks, todayKey, updateTask } = useScheduleMock()

const form = reactive({
  title: '',
  note: '',
  importance: 1 as ImportanceSelection,
  startTime: '08:00',
  endTime: '09:00',
})
const hoverImportance = shallowRef<Importance | null>(null)
const startTimeInput = useTemplateRef<HTMLInputElement>('dailyStartTimeInput')
const endTimeInput = useTemplateRef<HTMLInputElement>('dailyEndTimeInput')

const canSubmit = computed(() => form.title.trim().length > 0)
const displayedImportance = computed(() => hoverImportance.value ?? form.importance)
const isEditMode = computed(() => props.editTaskId !== undefined && props.editTaskId !== null)
const isScheduleTaskEditor = computed(() => Boolean(props.taskDate) || isEditMode.value)
const editingTask = computed(() =>
  isEditMode.value
    ? sortedTasks.value.find((task) => task.id === props.editTaskId)
    : undefined,
)
const promptTaskDate = computed(() => props.taskDate ?? todayKey.value)
const taskEditorTitle = computed(() => isEditMode.value ? '編輯任務' : '新增任務')
const taskEditorDate = computed(() => parseLocalDateKey(promptTaskDate.value))
const taskEditorDay = computed(() => taskEditorDate.value.getDate())
const taskEditorDateLabel = computed(() => formatTaskDateLabel(taskEditorDate.value))
const todayTasks = computed(() =>
  sortedTasks.value.filter((task) => task.date === promptTaskDate.value),
)
const shouldEmitClose = computed(() => isEditMode.value || Boolean(props.taskDate) || Boolean(props.closeAfterSubmit))

function parseLocalDateKey(dateKey: string) {
  const [year, month, day] = dateKey.split('-').map(Number)
  return new Date(year, month - 1, day)
}

function formatTaskDateLabel(date: Date) {
  const weekdayLabels = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  return `${date.getMonth() + 1}月${date.getDate()}日  ${weekdayLabels[date.getDay()]}`
}

function toPriority(importance: ImportanceSelection): Priority {
  if (importance >= 4) return 'high'
  if (importance >= 2) return 'medium'
  return 'low'
}

function resetForm() {
  form.title = ''
  form.note = ''
  form.importance = 1
  hoverImportance.value = null
  form.startTime = '08:00'
  form.endTime = defaultTaskEndTime(form.startTime)
}

function applyInitialDraft() {
  if (isEditMode.value) return

  resetForm()
  form.title = props.initialTitle?.trim() ?? ''
  form.note = props.initialNote?.trim() ?? ''
  if (form.title) {
    form.importance = 3
  }
  mode.value = 'form'
}

function previewImportance(importance: Importance) {
  hoverImportance.value = importance
}

function clearImportancePreview() {
  hoverImportance.value = null
}

function addMinutesToTime(time: string, minutesToAdd: number) {
  const [rawHours = 0, rawMinutes = 0] = time.split(':').map(Number)
  const totalMinutes = ((rawHours * 60 + rawMinutes + minutesToAdd) % 1440 + 1440) % 1440
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60

  return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}`
}

function defaultTaskEndTime(startTime: string) {
  return addMinutesToTime(startTime || '08:00', 60)
}

function updateTaskStartTime(event: Event) {
  const input = event.target
  if (!(input instanceof HTMLInputElement)) return

  form.startTime = input.value
  if (!form.endTime || form.endTime <= input.value) {
    form.endTime = defaultTaskEndTime(input.value)
  }
}

function openTimePicker(input: HTMLInputElement | null) {
  if (!input) return

  input.focus()
  try {
    input.showPicker?.()
  } catch {
    // Native time pickers require direct user activation in some browsers.
  }
}

function openStartTimePicker() {
  openTimePicker(startTimeInput.value)
}

function openEndTimePicker() {
  openTimePicker(endTimeInput.value)
}

function closePrompt() {
  if (shouldEmitClose.value) {
    emit('close')
    return
  }

  markDailyTaskPromptHandled()
}

function submitTask() {
  const title = form.title.trim()
  if (!title) return

  if (isEditMode.value) {
    const task = editingTask.value
    if (!task) {
      closePrompt()
      return
    }

    updateTask(task.id, {
      title,
      note: form.note.trim(),
      time: form.startTime,
      endTime: form.endTime,
      priority: toPriority(form.importance),
      importance: form.importance,
    })
    closePrompt()
    return
  }

  addTaskFromForm(title)

  if (isScheduleTaskEditor.value || props.closeAfterSubmit) {
    closePrompt()
    return
  }

  resetForm()
  mode.value = 'list'
}

function addTaskFromForm(title: string) {
  addTask({
    title,
    note: form.note.trim(),
    date: promptTaskDate.value,
    time: form.startTime,
    endTime: form.endTime,
    priority: toPriority(form.importance),
    importance: form.importance,
  })
}

function saveTaskAndCreateNext() {
  const title = form.title.trim()
  if (!title || isEditMode.value) return

  addTaskFromForm(title)
  resetForm()
  mode.value = 'form'
}

function openAddForm() {
  resetForm()
  mode.value = 'form'
}

function saveAndClose() {
  closePrompt()
}

function handleTaskCardWheel(event: WheelEvent) {
  const container = event.currentTarget as HTMLElement | null
  if (!container) return

  const delta = Math.abs(event.deltaY) >= Math.abs(event.deltaX) ? event.deltaY : event.deltaX
  const maxScrollLeft = container.scrollWidth - container.clientWidth
  if (delta === 0 || maxScrollLeft <= 0) return

  const nextScrollLeft = Math.min(maxScrollLeft, Math.max(0, container.scrollLeft + delta))
  if (nextScrollLeft === container.scrollLeft) return

  event.preventDefault()
  container.scrollLeft = nextScrollLeft
}

const handleKeyDown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    closePrompt()
  }
}

watch(
  editingTask,
  (task) => {
    if (!isEditMode.value) return
    if (!task) {
      closePrompt()
      return
    }

    form.title = task.title
    form.note = task.note ?? ''
    form.importance = task.importance ?? priorityToImportance(task.priority)
    form.startTime = task.time
    form.endTime = task.endTime ?? defaultTaskEndTime(task.time)
    mode.value = 'form'
  },
  { immediate: true },
)

watch(
  () => [props.initialTitle, props.initialNote, props.closeAfterSubmit] as const,
  applyInitialDraft,
  { immediate: true },
)

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
})
</script>

<template>
  <Teleport to="body">
  <div
    class="daily-task-backdrop"
    :class="{ 'daily-task-editor-backdrop': isScheduleTaskEditor }"
    role="presentation"
  >
    <section
      class="daily-task-modal"
      :class="{
        'daily-task-modal-list': mode === 'list',
        'daily-task-editor-modal': isScheduleTaskEditor,
      }"
      role="dialog"
      aria-modal="true"
      aria-labelledby="daily-task-title"
    >
      <template v-if="isScheduleTaskEditor">
        <div class="daily-task-editor-calendar-mark" aria-hidden="true">
          <span>{{ taskEditorDay }}</span>
        </div>

        <header class="daily-task-editor-header">
          <h2 id="daily-task-title">{{ taskEditorTitle }}</h2>
          <button type="button" class="daily-task-editor-close" aria-label="關閉" @click="closePrompt">
            <span aria-hidden="true">&times;</span>
          </button>
        </header>

        <form class="daily-task-form daily-task-editor-form" @submit.prevent="submitTask">
          <div class="daily-task-field daily-task-editor-title-field">
            <label class="daily-task-editor-field-label" for="daily-task-editor-title">標題：</label>
            <input
              id="daily-task-editor-title"
              v-model="form.title"
              type="text"
              placeholder="作業、家事、購物清單..."
              autofocus
            >

            <fieldset class="daily-task-importance daily-task-editor-importance" aria-label="重要度選擇">
              <legend class="sr-only">重要度選擇</legend>
              <div
                class="daily-task-importance-options"
                @mouseleave="clearImportancePreview"
                @focusout="clearImportancePreview"
              >
                <button
                  v-for="importance in importanceOptions"
                  :key="importance"
                  type="button"
                  class="daily-task-importance-dot"
                  :class="{ selected: importance <= displayedImportance }"
                  :aria-pressed="form.importance === importance"
                  :aria-label="`重要度 ${importance}`"
                  @mouseenter="previewImportance(importance)"
                  @focus="previewImportance(importance)"
                  @click="form.importance = importance"
                ></button>
              </div>
            </fieldset>
          </div>

          <label class="daily-task-field daily-task-note-field daily-task-editor-note-field">
            <span>備註說明：</span>
            <span v-if="!form.note" class="daily-task-note-optional" aria-hidden="true">（選填）</span>
            <textarea v-model="form.note" rows="3"></textarea>
          </label>

          <section class="daily-task-time-panel daily-task-editor-time-panel" aria-label="任務時間">
            <span class="daily-task-time-icon" aria-hidden="true"></span>

            <label class="daily-task-time-pill daily-task-editor-time-card" @click="openStartTimePicker">
              <span class="daily-task-editor-date-label">{{ taskEditorDateLabel }}</span>
              <strong>{{ form.startTime }}</strong>
              <input
                ref="dailyStartTimeInput"
                v-model="form.startTime"
                type="time"
                aria-label="開始時間"
                @change="updateTaskStartTime"
              >
            </label>

            <span class="daily-task-time-arrow" aria-hidden="true">&rarr;</span>

            <label class="daily-task-time-pill daily-task-editor-time-card" @click="openEndTimePicker">
              <span class="daily-task-editor-date-label">{{ taskEditorDateLabel }}</span>
              <strong>{{ form.endTime }}</strong>
              <input ref="dailyEndTimeInput" v-model="form.endTime" type="time" aria-label="結束時間">
            </label>
          </section>

          <div class="daily-task-editor-actions">
            <button
              v-if="!isEditMode"
              type="button"
              class="daily-task-save-next"
              :disabled="!canSubmit"
              @click="saveTaskAndCreateNext"
            >
              儲存並新增下一項
            </button>

            <div class="daily-task-editor-action-row">
              <button type="submit" class="daily-task-editor-save" :disabled="!canSubmit">儲存</button>
              <button type="button" class="daily-task-editor-cancel" @click="closePrompt">取消</button>
            </div>
          </div>
        </form>
      </template>

      <template v-else>
      <h2 id="daily-task-title">今天有甚麼任務嗎? ___幫你記住!</h2>

      <form v-if="mode === 'form'" class="daily-task-form" @submit.prevent="submitTask">
        <label class="daily-task-field">
          <span>任務標題：</span>
          <input v-model="form.title" type="text" placeholder="作業、家事、購物清單..." autofocus>
        </label>

        <label class="daily-task-field daily-task-note-field">
          <span>備註說明：</span>
          <textarea v-model="form.note" rows="2"></textarea>
        </label>

        <section class="daily-task-time-panel" aria-label="任務時間">
          <span class="daily-task-time-icon" aria-hidden="true"></span>

          <label class="daily-task-time-pill" @click="openStartTimePicker">
            <strong>{{ form.startTime }}</strong>
            <input
              ref="dailyStartTimeInput"
              v-model="form.startTime"
              type="time"
              aria-label="開始時間"
              @change="updateTaskStartTime"
            >
          </label>

          <span class="daily-task-time-arrow" aria-hidden="true">&rarr;</span>

          <label class="daily-task-time-pill" @click="openEndTimePicker">
            <strong>{{ form.endTime }}</strong>
            <input ref="dailyEndTimeInput" v-model="form.endTime" type="time" aria-label="結束時間">
          </label>
        </section>

        <div class="daily-task-footer">
          <fieldset class="daily-task-importance">
            <legend>重要度選擇</legend>
            <div
              class="daily-task-importance-options"
              @mouseleave="clearImportancePreview"
              @focusout="clearImportancePreview"
            >
              <button
                v-for="importance in importanceOptions"
                :key="importance"
                type="button"
                class="daily-task-importance-dot"
                :class="{ selected: importance <= displayedImportance }"
                :aria-pressed="form.importance === importance"
                :aria-label="`重要度 ${importance}`"
                @mouseenter="previewImportance(importance)"
                @focus="previewImportance(importance)"
                @click="form.importance = importance"
              ></button>
            </div>
          </fieldset>

          <div class="daily-task-actions">
            <button type="submit" class="daily-task-confirm" :disabled="!canSubmit">確認</button>
            <button type="button" class="daily-task-cancel" @click="closePrompt">取消</button>
          </div>
        </div>
      </form>

      <div v-else class="daily-task-list-view">
        <button type="button" class="daily-task-add-again" @click="openAddForm">新增任務 ＋</button>

        <section class="daily-task-list-panel" aria-label="今日任務清單">
          <header class="daily-task-list-header">
            <span aria-hidden="true">▾</span>
            <h3>今天有{{ todayTasks.length }}項任務</h3>
          </header>

          <div class="daily-task-card-row" @wheel="handleTaskCardWheel">
            <article v-for="task in todayTasks" :key="task.id" class="daily-task-card">
              <div class="daily-task-card-priority" aria-hidden="true">
                <span
                  v-for="dot in taskImportanceCount(task)"
                  :key="dot"
                  class="filled"
                ></span>
              </div>
              <h4>{{ task.title }}</h4>
              <p>{{ task.note || '（無備註）' }}</p>
            </article>
          </div>
        </section>

        <button type="button" class="daily-task-save-list" @click="saveAndClose">儲存</button>
      </div>

      <button v-if="mode === 'form' && !isEditMode" type="button" class="daily-task-skip" @click="closePrompt">
        跳過
      </button>
      </template>
    </section>
  </div>
  </Teleport>
</template>
