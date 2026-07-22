<script setup lang="ts">
import { useTemplateRef } from 'vue'

defineProps<{
  editorTitle: string
  day: number
  dateLabel: string
  isNew: boolean
  inactive: boolean
}>()

const emit = defineEmits<{ submit: [], cancel: [], saveNext: [] }>()
const title = defineModel<string>('title', { required: true })
const note = defineModel<string>('note', { required: true })
const date = defineModel<string>('date', { required: true })
const startTime = defineModel<string>('startTime', { required: true })
const endTime = defineModel<string>('endTime', { required: true })
const dialog = useTemplateRef<HTMLElement>('dialog')
const titleInput = useTemplateRef<HTMLInputElement>('titleInput')
const startTimeInput = useTemplateRef<HTMLInputElement>('startTimeInput')
const endTimeInput = useTemplateRef<HTMLInputElement>('endTimeInput')

function addMinutes(time: string, amount: number) {
  const [hours = 0, minutes = 0] = time.split(':').map(Number)
  const total = ((hours * 60 + minutes + amount) % 1440 + 1440) % 1440
  return `${String(Math.floor(total / 60)).padStart(2, '0')}:${String(total % 60).padStart(2, '0')}`
}

function updateStartTime() {
  if (!endTime.value || endTime.value <= startTime.value) endTime.value = addMinutes(startTime.value, 60)
}

function openPicker(input: HTMLInputElement | null) {
  input?.showPicker?.()
}

defineExpose({ dialog, titleInput })
</script>

<template>
  <form
    ref="dialog"
    class="reminder-edit-modal"
    role="dialog"
    :inert="inactive"
    :aria-hidden="inactive ? 'true' : undefined"
    :aria-modal="inactive ? undefined : 'true'"
    aria-labelledby="reminder-edit-title"
    tabindex="-1"
    @submit.prevent="emit('submit')"
  >
    <div class="reminder-edit-calendar-mark" aria-hidden="true"><span>{{ day }}</span></div>
    <header class="reminder-edit-header">
      <h3 id="reminder-edit-title">{{ editorTitle }}</h3>
      <button type="button" class="reminder-edit-close" aria-label="關閉" @click="emit('cancel')"><span aria-hidden="true">&times;</span></button>
    </header>

    <div class="reminder-edit-body">
      <label class="reminder-edit-field">
        <span>標題：</span>
        <input ref="titleInput" v-model="title" type="text" placeholder="節日、行程..." maxlength="120" required>
      </label>
      <label class="reminder-edit-field reminder-edit-note-field">
        <span>備註說明：</span><span v-if="!note" class="reminder-note-optional" aria-hidden="true">（選填）</span>
        <textarea v-model="note" rows="3" maxlength="1000"></textarea>
      </label>

      <section class="reminder-time-panel" aria-label="提醒時間">
        <span class="reminder-time-icon" aria-hidden="true"></span>
        <div class="reminder-time-card">
          <label class="reminder-date-picker"><span>{{ dateLabel }}</span><input v-model="date" type="date" aria-label="提醒日期"></label>
          <label class="reminder-clock-picker" @click="openPicker(startTimeInput)">
            <strong>{{ startTime }}</strong>
            <input ref="startTimeInput" v-model="startTime" type="time" aria-label="開始時間" @change="updateStartTime">
          </label>
        </div>
        <span class="reminder-time-arrow" aria-hidden="true">&rarr;</span>
        <div class="reminder-time-card">
          <label class="reminder-date-picker"><span>{{ dateLabel }}</span><input v-model="date" type="date" aria-label="提醒日期"></label>
          <label class="reminder-clock-picker" @click="openPicker(endTimeInput)">
            <strong>{{ endTime }}</strong><input ref="endTimeInput" v-model="endTime" type="time" aria-label="結束時間">
          </label>
        </div>
      </section>

      <div class="reminder-edit-actions">
        <button v-if="isNew" type="button" class="reminder-save-next" :disabled="!title.trim()" @click="emit('saveNext')">儲存並新增下一項</button>
        <div class="reminder-edit-action-row">
          <button type="submit" class="reminder-save" :disabled="!title.trim()">儲存</button>
          <button type="button" class="reminder-cancel" @click="emit('cancel')">取消</button>
        </div>
      </div>
    </div>
  </form>
</template>
