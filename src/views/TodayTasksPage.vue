<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import MainLayout from '../layouts/MainLayout.vue'
import SidebarWidgets from '../components/SidebarWidgets.vue'
import DateSelect from '../components/DateSelect.vue'
import TimeSelect from '../components/TimeSelect.vue'
import { priorityMeta, type Priority, useScheduleMock } from '../composables/useScheduleMock'
import { getLocalTodayKey } from '../utils/date'

const today = getLocalTodayKey()
const { sortedTasks, addTask, toggleTask, reorderTaskWithinPriority } = useScheduleMock()
const draggingTaskId = ref<number | null>(null)
const blockedDropId = ref<number | null>(null)

const taskForm = reactive({
  title: '',
  date: today,
  time: '09:00',
  priority: 'medium' as Priority,
})

// 頁首統計直接由模擬狀態計算，方便檢測新增與勾選完成後是否即時更新。
const remainingCount = computed(() => sortedTasks.value.filter((task) => !task.completed).length)
const completedCount = computed(() => sortedTasks.value.filter((task) => task.completed).length)

function submitTask() {
  const title = taskForm.title.trim()
  if (!title) return

  addTask({
    title,
    date: taskForm.date,
    time: taskForm.time,
    priority: taskForm.priority,
  })
  taskForm.title = ''
}

function handleTaskDrop(targetId: number) {
  if (draggingTaskId.value === null) return
  const changed = reorderTaskWithinPriority(draggingTaskId.value, targetId)
  blockedDropId.value = changed ? null : targetId
  draggingTaskId.value = null
  if (!changed) {
    window.setTimeout(() => {
      blockedDropId.value = null
    }, 700)
  }
}
</script>

<template>
  <MainLayout active-nav="freq" feed-class="schedule-feed">
    <template #sidebar>
      <SidebarWidgets />
    </template>

    <section class="schedule-page">
      <header class="schedule-page-header">
        <div>
          <p>Today's Plan</p>
          <h1>今日任務</h1>
        </div>
        <div class="schedule-summary">
          <span>{{ remainingCount }} 未完成</span>
          <span>{{ completedCount }} 已完成</span>
        </div>
      </header>

      <form class="schedule-create-panel" @submit.prevent="submitTask">
        <label class="schedule-field schedule-field-wide">
          <span>任務名稱</span>
          <input v-model="taskForm.title" type="text" placeholder="輸入今天要完成的事">
        </label>

        <DateSelect v-model="taskForm.date" label="日期" />
        <TimeSelect v-model="taskForm.time" label="時間" />

        <label class="schedule-field">
          <span>優先級</span>
          <select v-model="taskForm.priority">
            <option value="high">重要</option>
            <option value="medium">普通</option>
            <option value="low">稍後</option>
          </select>
        </label>

        <button type="submit" class="schedule-submit-btn">新增任務</button>
      </form>

      <div class="task-board">
        <!-- 同優先級拖移排序檢測區：跨優先級放下時會閃出阻擋樣式。 -->
        <article
          v-for="task in sortedTasks"
          :key="task.id"
          class="task-item"
          :class="[
            `priority-${task.priority}`,
            { completed: task.completed, blocked: blockedDropId === task.id },
          ]"
          draggable="true"
          @dragstart="draggingTaskId = task.id"
          @dragend="draggingTaskId = null"
          @dragover.prevent
          @drop="handleTaskDrop(task.id)"
        >
          <button type="button" class="task-drag-handle" aria-label="拖移排序">
            <span aria-hidden="true"></span>
          </button>

          <button
            type="button"
            class="task-check"
            :aria-pressed="task.completed"
            @click="toggleTask(task.id)"
          ></button>

          <div class="task-item-body">
            <div class="task-item-title-row">
              <h2>{{ task.title }}</h2>
              <span class="priority-badge">{{ priorityMeta[task.priority].label }}</span>
            </div>
            <p>{{ task.date }} / {{ task.time }}</p>
            <p v-if="task.note" class="task-item-note">{{ task.note }}</p>
          </div>
        </article>
      </div>
    </section>
  </MainLayout>
</template>
