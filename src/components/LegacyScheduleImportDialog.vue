<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, shallowRef, useTemplateRef } from 'vue'
import {
  type LegacyScheduleDecisionResult,
  useScheduleMock,
} from '../composables/useScheduleMock'

const { declineLegacySchedule, importLegacySchedule } = useScheduleMock()
const dialogPanel = useTemplateRef<HTMLElement>('dialogPanel')
const declineButton = useTemplateRef<HTMLButtonElement>('declineButton')
const isSubmitting = shallowRef(false)
const errorMessage = shallowRef('')
let previousActiveElement = typeof document !== 'undefined' && document.activeElement instanceof HTMLElement
  ? document.activeElement
  : null

const titleId = 'legacy-schedule-import-title'
const descriptionId = 'legacy-schedule-import-description'
const focusableSelector = [
  'button:not([disabled])',
  '[href]',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(', ')

function focusWithoutScroll(element: HTMLElement | null) {
  if (!element || !element.isConnected || element === document.body) return
  element.focus({ preventScroll: true })
}

function getFocusableElements() {
  const panel = dialogPanel.value
  if (!panel) return []

  return Array.from(panel.querySelectorAll<HTMLElement>(focusableSelector))
    .filter((element) => element.getClientRects().length > 0)
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    focusWithoutScroll(declineButton.value)
    return
  }

  if (event.key !== 'Tab') return

  const focusableElements = getFocusableElements()
  const panel = dialogPanel.value
  if (!panel || focusableElements.length === 0) {
    event.preventDefault()
    focusWithoutScroll(panel)
    return
  }

  const firstElement = focusableElements[0]
  const lastElement = focusableElements[focusableElements.length - 1]
  const activeElement = document.activeElement

  if (!(activeElement instanceof Node) || !panel.contains(activeElement)) {
    event.preventDefault()
    focusWithoutScroll(event.shiftKey ? lastElement : firstElement)
    return
  }

  if (event.shiftKey && activeElement === firstElement) {
    event.preventDefault()
    focusWithoutScroll(lastElement)
    return
  }

  if (!event.shiftKey && activeElement === lastElement) {
    event.preventDefault()
    focusWithoutScroll(firstElement)
  }
}

function completeDecision(decide: () => LegacyScheduleDecisionResult) {
  if (isSubmitting.value) return

  isSubmitting.value = true
  errorMessage.value = ''
  const result = decide()
  if (!result.ok) {
    errorMessage.value = result.message
    isSubmitting.value = false
  }
}

onMounted(() => {
  void nextTick(() => focusWithoutScroll(declineButton.value))
})

onBeforeUnmount(() => {
  const fallback = document.querySelector<HTMLElement>('[data-app-route-content]')
  focusWithoutScroll(
    previousActiveElement?.isConnected && previousActiveElement !== document.body
      ? previousActiveElement
      : fallback,
  )
  previousActiveElement = null
})
</script>

<template>
  <Teleport to="body">
    <div class="task-delete-backdrop" role="presentation">
      <section
        ref="dialogPanel"
        class="task-delete-dialog"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="titleId"
        :aria-describedby="descriptionId"
        tabindex="-1"
        @keydown="handleKeydown"
      >
        <header class="task-delete-header">
          <h2 :id="titleId">找到舊版排程資料</h2>
        </header>

        <div class="task-delete-content">
          <p :id="descriptionId">
            只有在這些舊資料確實屬於目前登入帳號時才匯入。匯入成功後，資料會移至目前帳號；選擇不匯入則會建立一份新的排程，舊資料仍會保留在此瀏覽器。
          </p>

          <p v-if="errorMessage" class="text-xl font-bold text-red-700" role="alert">
            {{ errorMessage }}
          </p>

          <div class="task-delete-actions">
            <button
              type="button"
              class="task-delete-confirm"
              :disabled="isSubmitting"
              @click="completeDecision(importLegacySchedule)"
            >
              這是我的資料，匯入
            </button>
            <button
              ref="declineButton"
              type="button"
              class="task-delete-cancel"
              style="color: var(--color-ink-strong)"
              :disabled="isSubmitting"
              @click="completeDecision(declineLegacySchedule)"
            >
              不匯入，建立新排程
            </button>
          </div>
        </div>
      </section>
    </div>
  </Teleport>
</template>
