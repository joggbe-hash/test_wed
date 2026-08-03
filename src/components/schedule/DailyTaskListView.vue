<script setup lang="ts">
import { taskImportanceCount, type TaskItem } from '../../features/schedule/types'

defineProps<{
  tasks: TaskItem[]
}>()

const emit = defineEmits<{
  add: []
  save: []
}>()

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
  <div class="daily-task-list-view">
    <button
      type="button"
      class="daily-task-add-again"
      data-daily-list-add
      @click="emit('add')"
    >
      新增任務 ＋
    </button>

    <section class="daily-task-list-panel" aria-label="今日任務清單">
      <header class="daily-task-list-header">
        <span aria-hidden="true">▾</span>
        <h3>今天有{{ tasks.length }}項任務</h3>
      </header>

      <div class="daily-task-card-row" @wheel="handleTaskCardWheel">
        <article v-for="task in tasks" :key="task.id" class="daily-task-card">
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

    <button type="button" class="daily-task-save-list" @click="emit('save')">儲存</button>
  </div>
</template>
