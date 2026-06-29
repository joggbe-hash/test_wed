<script setup lang="ts">
import { computed, reactive } from 'vue'
import { type Priority, useScheduleMock } from '../composables/useScheduleMock'
import { getLocalTodayKey } from '../utils/date'

type Importance = 1 | 2 | 3 | 4 | 5

const emit = defineEmits<{
  close: []
}>()

const importanceOptions: Importance[] = [1, 2, 3, 4, 5]
const { addTask } = useScheduleMock()

const form = reactive({
  title: '',
  note: '',
  importance: 3 as Importance,
})

const canSubmit = computed(() => form.title.trim().length > 0)

function toPriority(importance: Importance): Priority {
  if (importance >= 4) return 'high'
  if (importance >= 2) return 'medium'
  return 'low'
}

function resetForm() {
  form.title = ''
  form.note = ''
  form.importance = 3
}

function saveTask() {
  const title = form.title.trim()
  if (!title) return false

  addTask({
    title,
    note: form.note.trim(),
    date: getLocalTodayKey(),
    time: '09:00',
    priority: toPriority(form.importance),
  })

  return true
}

function saveAndClose() {
  if (!saveTask()) return
  emit('close')
}

function saveAndCreateNext() {
  if (!saveTask()) return
  resetForm()
}
</script>

<template>
  <Teleport to="body">
    <div class="add-task-backdrop" role="presentation" @click.self="emit('close')">
      <section class="add-task-modal" role="dialog" aria-modal="true" aria-labelledby="add-task-title">
        <header class="add-task-header">
          <h2 id="add-task-title">新增任務</h2>
          <button type="button" class="add-task-close" aria-label="關閉" @click="emit('close')">
            <span aria-hidden="true">&times;</span>
          </button>
        </header>

        <form class="add-task-form" @submit.prevent="saveAndClose">
          <label class="add-task-field add-task-title-field">
            <span>任務標題:</span>
            <input v-model="form.title" type="text" placeholder="作業、家事、購物清單..." autofocus>
          </label>

          <label class="add-task-field add-task-note-field">
            <span>備註說明:</span>
            <textarea v-model="form.note" rows="3"></textarea>
          </label>

          <div class="add-task-bottom">
            <fieldset class="add-task-importance">
              <legend>重要度選擇</legend>
              <div class="add-task-importance-options">
                <button
                  v-for="importance in importanceOptions"
                  :key="importance"
                  type="button"
                  class="add-task-importance-dot"
                  :class="{ selected: form.importance === importance }"
                  :aria-pressed="form.importance === importance"
                  :aria-label="`重要度 ${importance}`"
                  @click="form.importance = importance"
                ></button>
              </div>
            </fieldset>

            <div class="add-task-actions">
              <button
                type="button"
                class="add-task-save-next"
                :disabled="!canSubmit"
                @click="saveAndCreateNext"
              >
                儲存並新增下一項
              </button>
              <div class="add-task-action-row">
                <button type="submit" class="add-task-save" :disabled="!canSubmit">儲存</button>
                <button type="button" class="add-task-cancel" @click="emit('close')">取消</button>
              </div>
            </div>
          </div>
        </form>
      </section>
    </div>
  </Teleport>
</template>
