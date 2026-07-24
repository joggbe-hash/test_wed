<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, useTemplateRef } from 'vue'

const props = defineProps<{
  item: { id: number; title: string }
  kind?: 'task' | 'reminder' | 'post'
  returnFocusId?: string
  confirmFocusId?: string
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
let wasConfirmed = false

const itemKindLabel = computed(() => {
  if (props.kind === 'reminder') return '提醒'
  if (props.kind === 'post') return '貼文'
  return '任務'
})
const itemTitle = computed(() => props.item.title.trim() || `這個${itemKindLabel.value}`)

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
  wasConfirmed = true
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

  const preferredFocusId = wasConfirmed
    ? props.confirmFocusId ?? props.returnFocusId
    : props.returnFocusId
  const preferredFocusTarget = preferredFocusId
    ? document.getElementById(preferredFocusId)
    : null
  const fallbackFocusTarget = previousActiveElement?.isConnected && previousActiveElement !== document.body
    ? previousActiveElement
    : null
  const focusTarget = preferredFocusTarget ?? fallbackFocusTarget

  focusTarget?.focus()
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
        <header class="task-delete-header">
          <h2 :id="titleId">確認刪除{{ itemKindLabel }}</h2>
          <button
            type="button"
            class="task-delete-close"
            aria-label="取消刪除"
            @click="closeDialog"
          >
            <span aria-hidden="true">&times;</span>
          </button>
        </header>

        <div class="task-delete-content">
          <p :id="descriptionId">
            <template v-if="props.kind === 'post'">永久刪除貼文，將無法復原</template>
            <template v-else>即將永久刪除{{ itemKindLabel }}「{{ itemTitle }}」，此動作無法復原。</template>
          </p>
          <div class="task-delete-actions">
            <button
              type="button"
              class="task-delete-confirm"
              @click="confirmDelete"
            >
              確定刪除
            </button>
            <button
              ref="cancelButton"
              type="button"
              class="task-delete-cancel"
              @click="closeDialog"
            >
              取消
            </button>
          </div>
        </div>
      </section>
    </div>
  </Teleport>
</template>
