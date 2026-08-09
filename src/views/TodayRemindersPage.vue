<script setup lang="ts">
import { reactive, watch } from 'vue'
import MainLayout from '../layouts/MainLayout.vue'
import SidebarWidgets from '../components/SidebarWidgets.vue'
import DateSelect from '../components/DateSelect.vue'
import TimeSelect from '../components/TimeSelect.vue'
import { useSchedule } from '../composables/useSchedule'

const { todayKey, todayReminders, addReminder } = useSchedule()

const reminderForm = reactive({
  title: '',
  date: todayKey.value,
  time: '10:00',
  note: '',
})

watch(todayKey, (nextDate, previousDate) => {
  if (reminderForm.date === previousDate) {
    reminderForm.date = nextDate
  }
})

// 純前端建立流程：新增到模擬狀態後重設表單，不會呼叫後端介面。
function submitReminder() {
  const title = reminderForm.title.trim()
  if (!title) return

  const saved = addReminder({
    title,
    date: reminderForm.date,
    time: reminderForm.time,
    note: reminderForm.note.trim(),
  })
  if (saved) {
    reminderForm.title = ''
    reminderForm.note = ''
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
          <p>Reminder List</p>
          <h1>今日提醒</h1>
        </div>
        <div class="schedule-summary">
          <span>{{ todayReminders.length }} 則提醒</span>
        </div>
      </header>

      <form class="schedule-create-panel reminder-create-panel" @submit.prevent="submitReminder">
        <label class="schedule-field schedule-field-wide">
          <span>提醒標題</span>
          <input v-model="reminderForm.title" type="text" placeholder="輸入提醒事項">
        </label>

        <DateSelect v-model="reminderForm.date" label="日期" />
        <TimeSelect v-model="reminderForm.time" label="時間" />

        <label class="schedule-field schedule-field-full">
          <span>備註</span>
          <textarea v-model="reminderForm.note" rows="3" placeholder="補充提醒內容"></textarea>
        </label>

        <button type="submit" class="schedule-submit-btn">新增提醒</button>
      </form>

      <div class="reminder-list">
        <article v-for="reminder in todayReminders" :key="reminder.id" class="reminder-item">
          <div class="reminder-time">
            <strong>{{ reminder.time }}</strong>
            <span>{{ reminder.date }}</span>
          </div>
          <div class="reminder-body">
            <h2>{{ reminder.title }}</h2>
            <p>{{ reminder.note || '無備註' }}</p>
          </div>
        </article>
      </div>
    </section>
  </MainLayout>
</template>
