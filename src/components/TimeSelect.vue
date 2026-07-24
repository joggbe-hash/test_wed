<script setup lang="ts">
import { useTemplateRef } from 'vue'

const model = defineModel<string>({ required: true })

defineProps<{
  label: string
}>()

const timeInput = useTemplateRef<HTMLInputElement>('timeInput')

function openTimePicker() {
  const input = timeInput.value
  if (!input) return

  input.focus()
  try {
    input.showPicker?.()
  } catch {
    // 瀏覽器不允許自動開啟選擇器時，保留原生聚焦行為。
  }
}
</script>

<template>
  <label class="schedule-field">
    <span>{{ label }}</span>
    <input ref="timeInput" v-model="model" type="time" @focus="openTimePicker">
  </label>
</template>
