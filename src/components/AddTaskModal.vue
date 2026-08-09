<script setup lang="ts">
import { computed, reactive, shallowRef, onMounted, onUnmounted, watch } from 'vue'
import {
  priorityToImportance,
  type Priority,
  type TaskImportance,
  useSchedule,
} from '../composables/useSchedule'
import { getLocalTodayKey } from '../utils/date'

type Importance = TaskImportance
type ImportanceSelection = TaskImportance

const props = defineProps<{
  editTaskId?: number | null
}>()
const emit = defineEmits<{
  close: []
}>()

const importanceOptions: Importance[] = [1, 2, 3, 4, 5]
const {
  addTask,
  sortedTasks,
  updateTask,
} = useSchedule()

const form = reactive({
  title: '',
  note: '',
  importance: 1 as ImportanceSelection,
})
const hoverImportance = shallowRef<Importance | null>(null)

const canSubmit = computed(() => form.title.trim().length > 0)
const displayedImportance = computed(() => hoverImportance.value ?? form.importance)
const isEditMode = computed(() => props.editTaskId !== undefined && props.editTaskId !== null)
const editingTask = computed(() =>
  isEditMode.value
    ? sortedTasks.value.find((task) => task.id === props.editTaskId)
    : undefined,
)
const modalTitle = computed(() => isEditMode.value ? '編輯任務' : '新增任務')
const primaryActionLabel = computed(() => isEditMode.value ? '儲存修改' : '儲存')

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
}

function previewImportance(importance: Importance) {
  hoverImportance.value = importance
}

function clearImportancePreview() {
  hoverImportance.value = null
}

function saveTask() {
  const title = form.title.trim()
  if (!title) return false

  if (isEditMode.value) {
    const task = editingTask.value
    if (!task) return false

    return updateTask(task.id, {
      title,
      note: form.note.trim(),
      priority: toPriority(form.importance),
      importance: form.importance,
    })
  }

  return addTask({
    title,
    note: form.note.trim(),
    date: getLocalTodayKey(),
    time: '09:00',
    priority: toPriority(form.importance),
    importance: form.importance,
  })
}

function saveAndClose() {
  if (!saveTask()) return
  emit('close')
}

function saveAndCreateNext() {
  if (isEditMode.value) return
  if (!saveTask()) return
  resetForm()
}

const handleKeyDown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    emit('close')
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
})

watch(
  editingTask,
  (task) => {
    if (!isEditMode.value) {
      resetForm()
      return
    }

    if (!task) {
      emit('close')
      return
    }

    form.title = task.title
    form.note = task.note ?? ''
    form.importance = task.importance ?? priorityToImportance(task.priority)
  },
  { immediate: true },
)
</script>

<template>
  <Teleport to="body">
    <div class="add-task-backdrop" role="presentation" @click.self="emit('close')">
      <section class="add-task-modal" role="dialog" aria-modal="true" aria-labelledby="add-task-title">
        <header class="add-task-header">
          <h2 id="add-task-title">{{ modalTitle }}</h2>
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
              <div
                class="add-task-importance-options"
                @mouseleave="clearImportancePreview"
                @focusout="clearImportancePreview"
              >
                <button
                  v-for="importance in importanceOptions"
                  :key="importance"
                  type="button"
                  class="add-task-importance-dot"
                  :class="{ selected: importance <= displayedImportance }"
                  :aria-pressed="form.importance === importance"
                  :aria-label="`重要度 ${importance}`"
                  @mouseenter="previewImportance(importance)"
                  @focus="previewImportance(importance)"
                  @click="form.importance = importance"
                ></button>
              </div>
            </fieldset>

            <div class="add-task-actions">
              <button
                v-if="!isEditMode"
                type="button"
                class="add-task-save-next"
                :disabled="!canSubmit"
                @click="saveAndCreateNext"
              >
                儲存並新增下一項
              </button>
              <div class="add-task-action-row">
                <button type="submit" class="add-task-save" :disabled="!canSubmit">{{ primaryActionLabel }}</button>
                <button type="button" class="add-task-cancel" @click="emit('close')">取消</button>
              </div>
            </div>
          </div>
        </form>
      </section>
    </div>
  </Teleport>
</template>
