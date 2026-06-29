<script setup lang="ts">
import { computed, reactive, shallowRef } from 'vue'
import {
  getTodayTaskDate,
  markDailyTaskPromptHandled,
} from '../composables/useDailyTaskPrompt'
import { type Priority, useScheduleMock } from '../composables/useScheduleMock'

type Importance = 1 | 2 | 3 | 4 | 5
type PromptMode = 'form' | 'list'

const importanceOptions: Importance[] = [1, 2, 3, 4, 5]
const mode = shallowRef<PromptMode>('form')
const { addTask, sortedTasks } = useScheduleMock()

const form = reactive({
  title: '',
  note: '',
  importance: 3 as Importance,
})

const canSubmit = computed(() => form.title.trim().length > 0)
const todayTaskDate = computed(() => getTodayTaskDate())
const todayTasks = computed(() =>
  sortedTasks.value.filter((task) => task.date === todayTaskDate.value),
)

function toPriority(importance: Importance): Priority {
  if (importance >= 4) return 'high'
  if (importance >= 2) return 'medium'
  return 'low'
}

function priorityDotCount(priority: Priority) {
  if (priority === 'high') return 5
  if (priority === 'medium') return 3
  return 1
}

function resetForm() {
  form.title = ''
  form.note = ''
  form.importance = 3
}

function submitTask() {
  const title = form.title.trim()
  if (!title) return

  addTask({
    title,
    note: form.note.trim(),
    date: todayTaskDate.value,
    time: '09:00',
    priority: toPriority(form.importance),
  })

  resetForm()
  mode.value = 'list'
}

function openAddForm() {
  resetForm()
  mode.value = 'form'
}

function saveAndClose() {
  markDailyTaskPromptHandled()
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
</script>

<template>
  <div class="daily-task-backdrop" role="presentation">
    <section
      class="daily-task-modal"
      :class="{ 'daily-task-modal-list': mode === 'list' }"
      role="dialog"
      aria-modal="true"
      aria-labelledby="daily-task-title"
    >
      <h2 id="daily-task-title">今天有甚麼任務嗎? ___幫你記住!</h2>

      <form v-if="mode === 'form'" class="daily-task-form" @submit.prevent="submitTask">
        <label class="daily-task-field">
          <span>任務標題:</span>
          <input v-model="form.title" type="text" placeholder="作業、家事、購物清單..." autofocus>
        </label>

        <label class="daily-task-field daily-task-note-field">
          <span>備註說明:</span>
          <textarea v-model="form.note" rows="2"></textarea>
        </label>

        <div class="daily-task-footer">
          <fieldset class="daily-task-importance">
            <legend>重要度選擇</legend>
            <div class="daily-task-importance-options">
              <button
                v-for="importance in importanceOptions"
                :key="importance"
                type="button"
                class="daily-task-importance-dot"
                :class="{ selected: form.importance === importance }"
                :aria-pressed="form.importance === importance"
                :aria-label="`重要度 ${importance}`"
                @click="form.importance = importance"
              ></button>
            </div>
          </fieldset>

          <div class="daily-task-actions">
            <button type="submit" class="daily-task-confirm" :disabled="!canSubmit">確認</button>
            <button type="button" class="daily-task-cancel" @click="markDailyTaskPromptHandled">取消</button>
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
                  v-for="dot in importanceOptions"
                  :key="dot"
                  :class="{ filled: dot <= priorityDotCount(task.priority) }"
                ></span>
              </div>
              <h4>{{ task.title }}</h4>
              <p>{{ task.note || '（無備註）' }}</p>
            </article>
          </div>
        </section>

        <button type="button" class="daily-task-save-list" @click="saveAndClose">儲存</button>
      </div>

      <button v-if="mode === 'form'" type="button" class="daily-task-skip" @click="markDailyTaskPromptHandled">
        跳過
      </button>
    </section>
  </div>
</template>
