<script setup lang="ts">
import { useTemplateRef } from 'vue'

const model = defineModel<string>({ required: true })

defineProps<{
  label: string
}>()

const dateInput = useTemplateRef<HTMLInputElement>('dateInput')

function openDatePicker() {
  const input = dateInput.value
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
    <input ref="dateInput" v-model="model" type="date" @focus="openDatePicker">
  </label>
</template>
