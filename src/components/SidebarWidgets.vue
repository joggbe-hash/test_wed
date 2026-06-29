<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import { useRouter } from 'vue-router'
import AddTaskModal from './AddTaskModal.vue'
import DateScheduleModal from './DateScheduleModal.vue'
import InspirationListModal from './InspirationListModal.vue'
import { useScheduleMock } from '../composables/useScheduleMock'

const weekDays = ['SUN', 'MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT']
const currentDate = shallowRef(new Date())
const visibleMonth = shallowRef(new Date(currentDate.value.getFullYear(), currentDate.value.getMonth(), 1))
const selectedDateKey = ref<string | null>(null)
const isAddTaskModalOpen = shallowRef(false)
const isInspirationListOpen = shallowRef(false)
const router = useRouter()
const { sortedTasks, sortedReminders, toggleTask } = useScheduleMock()

const sidebarTasks = computed(() => sortedTasks.value)
const firstReminder = computed(() => sortedReminders.value[0])
let calendarTimer: number | undefined

const monthLabel = computed(() =>
  new Intl.DateTimeFormat('en-US', {
    month: 'long',
    year: 'numeric',
  }).format(visibleMonth.value),
)

const calendarCells = computed(() => {
  const year = visibleMonth.value.getFullYear()
  const month = visibleMonth.value.getMonth()
  const today = currentDate.value
  const firstDay = new Date(year, month, 1).getDay()
  const daysInMonth = new Date(year, month + 1, 0).getDate()
  const cells = Array.from({ length: firstDay }, (_, index) => ({
    key: `blank-${index}`,
    label: '',
    dateKey: '',
    isToday: false,
    isCurrentMonth: false,
  }))

  for (let day = 1; day <= daysInMonth; day += 1) {
    const dateKey = [
      year,
      String(month + 1).padStart(2, '0'),
      String(day).padStart(2, '0'),
    ].join('-')

    cells.push({
      key: dateKey,
      label: String(day),
      dateKey,
      isToday:
        today.getFullYear() === year &&
        today.getMonth() === month &&
        today.getDate() === day,
      isCurrentMonth: true,
    })
  }

  return cells
})

function shiftMonth(offset: number) {
  visibleMonth.value = new Date(
    visibleMonth.value.getFullYear(),
    visibleMonth.value.getMonth() + offset,
    1,
  )
}

function openDateSchedule(dateKey: string) {
  if (!dateKey) return
  selectedDateKey.value = dateKey
}

function refreshCurrentDate() {
  const previousDate = currentDate.value
  const nextDate = new Date()
  const wasViewingCurrentMonth =
    visibleMonth.value.getFullYear() === previousDate.getFullYear() &&
    visibleMonth.value.getMonth() === previousDate.getMonth()

  currentDate.value = nextDate

  if (wasViewingCurrentMonth) {
    visibleMonth.value = new Date(nextDate.getFullYear(), nextDate.getMonth(), 1)
  }
}

onMounted(() => {
  calendarTimer = window.setInterval(refreshCurrentDate, 60_000)
})

onBeforeUnmount(() => {
  if (calendarTimer !== undefined) {
    window.clearInterval(calendarTimer)
  }
})
</script>

<template>
  <aside class="sidebar-widgets" aria-label="側邊工具">
    <section class="sidebar-panel calendar-panel" aria-label="月曆">
      <header class="calendar-header">
        <button type="button" class="calendar-nav-btn" aria-label="上一個月" @click="shiftMonth(-1)">
          <span aria-hidden="true">&lsaquo;</span>
        </button>
        <h2>{{ monthLabel }}</h2>
        <button type="button" class="calendar-nav-btn" aria-label="下一個月" @click="shiftMonth(1)">
          <span aria-hidden="true">&rsaquo;</span>
        </button>
      </header>

      <div class="calendar-grid calendar-weekdays" aria-hidden="true">
        <span v-for="day in weekDays" :key="day">{{ day }}</span>
      </div>

      <div class="calendar-grid calendar-days">
        <button
          v-for="cell in calendarCells"
          :key="cell.key"
          type="button"
          class="calendar-day"
          :class="{ active: cell.isToday, muted: !cell.isCurrentMonth }"
          :disabled="!cell.isCurrentMonth"
          :aria-label="cell.isCurrentMonth ? `查看 ${cell.dateKey} 的行程` : undefined"
          @click="openDateSchedule(cell.dateKey)"
        >
          {{ cell.label }}
        </button>
      </div>
    </section>

    <section class="sidebar-panel reminder-panel">
      <h2>今日提醒</h2>
      <p>{{ firstReminder ? firstReminder.title : ':p' }}</p>
    </section>

    <section class="sidebar-panel task-panel">
      <button type="button" class="sidebar-panel-heading" @click="router.push('/tasks/today')">
        今日任務
      </button>
      <div class="task-list">
        <label v-for="task in sidebarTasks" :key="task.id" class="task-row">
          <input type="checkbox" :checked="task.completed" @change="toggleTask(task.id)">
          <span>{{ task.title }}</span>
        </label>
      </div>
      <button type="button" class="add-task-btn" @click="isAddTaskModalOpen = true">+新增任務</button>
    </section>

    <button type="button" class="sidebar-panel note-panel" @click="isInspirationListOpen = true">
      記下想做的事
    </button>

    <DateScheduleModal
      v-if="selectedDateKey"
      :date-key="selectedDateKey"
      @update:date-key="selectedDateKey = $event"
      @close="selectedDateKey = null"
    />

    <AddTaskModal v-if="isAddTaskModalOpen" @close="isAddTaskModalOpen = false" />
    <InspirationListModal v-if="isInspirationListOpen" @close="isInspirationListOpen = false" />
  </aside>
</template>
