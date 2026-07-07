<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, useTemplateRef } from 'vue'
import type { TaskItem } from '../composables/useScheduleMock'

const props = defineProps<{
  task: Pick<TaskItem, 'id' | 'title'>
}>()

const emit = defineEmits<{
  cancel: []
  confirm: []
}>()

const dialogPanel = useTemplateRef<HTMLElement>('dialogPanel')
const cancelButton = useTemplateRef<HTMLButtonElement>('cancelButton')
const titleId = 'task-delete-confirm-title'
const descriptionId = 'task-delete-confirm-description'
let previousActiveElement: HTMLElement | null = null

const taskTitle = computed(() => props.task.title.trim() || '這個任務')

function getFocusableElements() {
  const panel = dialogPanel.value
  if (!panel) return []

  return Array.from(
    panel.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
    ),
  ).filter((element) => !element.hasAttribute('disabled') && element.tabIndex !== -1)
}

function closeDialog() {
  emit('cancel')
}

function confirmDelete() {
  emit('confirm')
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    closeDialog()
    return
  }

  if (event.key !== 'Tab') return

  const focusableElements = getFocusableElements()
  if (focusableElements.length === 0) {
    event.preventDefault()
    return
  }

  const firstElement = focusableElements[0]
  const lastElement = focusableElements[focusableElements.length - 1]

  if (event.shiftKey && document.activeElement === firstElement) {
    event.preventDefault()
    lastElement.focus()
    return
  }

  if (!event.shiftKey && document.activeElement === lastElement) {
    event.preventDefault()
    firstElement.focus()
  }
}

onMounted(() => {
  previousActiveElement = document.activeElement instanceof HTMLElement
    ? document.activeElement
    : null
  document.addEventListener('keydown', handleKeydown)
  nextTick(() => {
    cancelButton.value?.focus()
  })
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleKeydown)
  previousActiveElement?.focus()
})
</script>

<template>
  <Teleport to="body">
    <div class="task-delete-backdrop" @click.self="closeDialog">
      <section
        ref="dialogPanel"
        class="task-delete-dialog"
        role="alertdialog"
        aria-modal="true"
        :aria-labelledby="titleId"
        :aria-describedby="descriptionId"
      >
        <div class="task-delete-content">
          <span class="task-delete-kicker">確認刪除</span>
          <h2 :id="titleId">要刪除這項任務嗎？</h2>
          <div class="task-delete-task-card">
            <span>今日任務</span>
            <strong>{{ taskTitle }}</strong>
          </div>
          <p :id="descriptionId">
            刪除後會從今日任務移除，這個動作不能復原。
          </p>
        </div>
        <div class="task-delete-actions">
          <button
            ref="cancelButton"
            type="button"
            class="task-delete-cancel"
            @click="closeDialog"
          >
            取消
          </button>
          <button
            type="button"
            class="task-delete-confirm"
            @click="confirmDelete"
          >
            刪除
          </button>
        </div>
      </section>
    </div>
  </Teleport>
</template>
