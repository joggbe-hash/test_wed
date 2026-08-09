<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, shallowRef, useId, useTemplateRef } from 'vue'

const model = defineModel<string>({ required: true })

type TimeSelectVariant = 'field' | 'pill' | 'editor-card'

const props = withDefaults(defineProps<{
  describedBy?: string
  invalid?: boolean
  label: string
  subtitle?: string
  variant?: TimeSelectVariant
}>(), {
  describedBy: '',
  invalid: false,
  subtitle: '',
  variant: 'field',
})

const root = useTemplateRef<HTMLDivElement>('root')
const trigger = useTemplateRef<HTMLButtonElement>('trigger')
const isOpen = shallowRef(false)
const labelId = `time-label-${useId()}`
const panelId = `time-panel-${useId()}`
const hours = Array.from({ length: 12 }, (_, index) => index + 1)
const minutes = Array.from({ length: 60 }, (_, index) => index)

const parsedTime = computed(() => {
  const match = /^(\d{2}):(\d{2})$/.exec(model.value)
  const hour = Number(match?.[1] ?? 9)
  const minute = Number(match?.[2] ?? 0)

  return {
    hour: hour >= 0 && hour <= 23 ? hour : 9,
    minute: minute >= 0 && minute <= 59 ? minute : 0,
  }
})

const selectedPeriod = computed<'am' | 'pm'>(() => parsedTime.value.hour >= 12 ? 'pm' : 'am')
const selectedHour = computed(() => parsedTime.value.hour % 12 || 12)
const selectedMinute = computed(() => parsedTime.value.minute)
const displayTime = computed(() => {
  const period = selectedPeriod.value === 'am' ? '上午' : '下午'
  return `${period} ${pad(selectedHour.value)}:${pad(selectedMinute.value)}`
})

function pad(value: number) {
  return String(value).padStart(2, '0')
}

function updateTime(hour: number, minute: number) {
  model.value = `${pad(hour)}:${pad(minute)}`
}

function selectPeriod(period: 'am' | 'pm') {
  const hour = selectedHour.value % 12 + (period === 'pm' ? 12 : 0)
  updateTime(hour, selectedMinute.value)
}

function selectHour(hour: number) {
  const hour24 = hour % 12 + (selectedPeriod.value === 'pm' ? 12 : 0)
  updateTime(hour24, selectedMinute.value)
}

function selectMinute(minute: number) {
  updateTime(parsedTime.value.hour, minute)
}

async function openPicker() {
  if (isOpen.value) return
  isOpen.value = true
  await nextTick()
  root.value?.querySelectorAll<HTMLElement>('.time-picker-option[aria-selected="true"]')
    .forEach((option) => option.scrollIntoView?.({ block: 'center' }))
}

function closePicker(returnFocus = false) {
  if (!isOpen.value) return
  isOpen.value = false
  if (returnFocus) nextTick(() => trigger.value?.focus())
}

function togglePicker() {
  if (isOpen.value) closePicker()
  else openPicker()
}

function handlePointerDown(event: PointerEvent) {
  if (root.value && !root.value.contains(event.target as Node)) closePicker()
}

onMounted(() => document.addEventListener('pointerdown', handlePointerDown))
onBeforeUnmount(() => document.removeEventListener('pointerdown', handlePointerDown))
</script>

<template>
  <div
    ref="root"
    class="time-select"
    :class="[`time-select-${props.variant}`, { 'schedule-field': props.variant === 'field' }]"
  >
    <span :id="labelId" :class="{ 'sr-only': props.variant !== 'field' }">{{ props.label }}</span>
    <button
      ref="trigger"
      type="button"
      class="time-picker-trigger"
      aria-haspopup="dialog"
      :aria-expanded="isOpen"
      :aria-controls="panelId"
      :aria-describedby="props.describedBy || undefined"
      :aria-invalid="props.invalid || undefined"
      :aria-labelledby="`${labelId} ${panelId}-value`"
      @click="togglePicker"
      @keydown.down.prevent="openPicker"
    >
      <svg v-if="props.variant === 'field'" class="time-picker-icon" viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="12" cy="12" r="8.5" />
        <path d="M12 7.5v5l3.25 2" />
      </svg>
      <span v-if="props.subtitle" class="time-picker-subtitle">{{ props.subtitle }}</span>
      <span :id="`${panelId}-value`" class="time-picker-value">{{ displayTime }}</span>
      <svg class="time-picker-chevron" viewBox="0 0 20 20" aria-hidden="true">
        <path d="m6.5 8 3.5 3.5L13.5 8" />
      </svg>
    </button>

    <Transition name="time-picker">
      <div
        v-if="isOpen"
        :id="panelId"
        class="time-picker-panel"
        role="dialog"
        :aria-labelledby="labelId"
        @keydown.escape.stop.prevent="closePicker(true)"
      >
        <div class="time-picker-period" role="group" aria-label="時段">
          <button
            type="button"
            data-period="am"
            :class="{ selected: selectedPeriod === 'am' }"
            :aria-pressed="selectedPeriod === 'am'"
            @click="selectPeriod('am')"
          >上午</button>
          <button
            type="button"
            data-period="pm"
            :class="{ selected: selectedPeriod === 'pm' }"
            :aria-pressed="selectedPeriod === 'pm'"
            @click="selectPeriod('pm')"
          >下午</button>
        </div>

        <div class="time-picker-columns">
          <div class="time-picker-column">
            <span class="time-picker-column-label">時</span>
            <div class="time-picker-list" role="listbox" aria-label="小時">
              <button
                v-for="hour in hours"
                :key="hour"
                type="button"
                class="time-picker-option"
                :class="{ selected: selectedHour === hour }"
                :data-hour="hour"
                role="option"
                :aria-selected="selectedHour === hour"
                @click="selectHour(hour)"
              >{{ pad(hour) }}</button>
            </div>
          </div>

          <div class="time-picker-column">
            <span class="time-picker-column-label">分</span>
            <div class="time-picker-list" role="listbox" aria-label="分鐘">
              <button
                v-for="minute in minutes"
                :key="minute"
                type="button"
                class="time-picker-option"
                :class="{ selected: selectedMinute === minute }"
                :data-minute="minute"
                role="option"
                :aria-selected="selectedMinute === minute"
                @click="selectMinute(minute)"
              >{{ pad(minute) }}</button>
            </div>
          </div>
        </div>

        <button type="button" class="time-picker-done" @click="closePicker(true)">完成</button>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.time-select {
  position: relative;
}

.time-select-pill,
.time-select-editor-card {
  min-width: 0;
}

.time-picker-trigger {
  display: flex;
  min-height: 2.75rem;
  width: 100%;
  cursor: pointer;
  align-items: center;
  gap: 0.625rem;
  border: 1px solid #ded5cc;
  border-radius: 0.75rem;
  background: var(--color-surface-warm);
  padding: 0 0.75rem;
  color: #2f2822;
  font-size: 1rem;
  font-weight: 500;
  text-align: left;
  transition: border-color 160ms ease, box-shadow 160ms ease, background-color 160ms ease;
}

.time-picker-trigger:hover {
  border-color: #cdbdac;
}

.time-picker-trigger:focus-visible,
.time-picker-trigger[aria-expanded="true"] {
  border-color: #b99878;
  background: #fff;
  outline: none;
  box-shadow: 0 0 0 3px rgb(185 152 120 / 14%);
}

.time-picker-trigger[aria-invalid="true"] {
  border-color: #000;
  outline: 2px solid #000;
  outline-offset: 3px;
}

.time-picker-icon,
.time-picker-chevron {
  flex: none;
  fill: none;
  stroke: #8a6a52;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 1.7;
}

.time-picker-icon {
  width: 1.125rem;
  height: 1.125rem;
}

.time-picker-chevron {
  width: 1rem;
  height: 1rem;
  margin-left: auto;
  transition: transform 160ms ease;
}

.time-picker-trigger[aria-expanded="true"] .time-picker-chevron {
  transform: rotate(180deg);
}

.time-select-pill .time-picker-trigger,
.time-select-editor-card .time-picker-trigger {
  justify-content: center;
  border-color: #000;
  border-radius: 1.25rem;
  background: #fff;
  color: #000;
  font-weight: 900;
}

.time-select-pill .time-picker-trigger {
  height: 4.125rem;
  padding: 0 2.5rem;
}

.time-select-pill .time-picker-value,
.time-select-editor-card .time-picker-value {
  font-size: 1.75rem;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.time-select-editor-card .time-picker-trigger {
  height: 10.75rem;
  flex-direction: column;
  gap: 1.75rem;
  padding: 1.25rem 2.5rem;
  text-align: center;
}

.time-picker-subtitle {
  white-space: nowrap;
  font-size: 1.625rem;
  line-height: 1;
}

.time-select-pill .time-picker-chevron,
.time-select-editor-card .time-picker-chevron {
  position: absolute;
  right: 1rem;
  margin-left: 0;
  stroke: #000;
}

.time-picker-panel {
  position: absolute;
  z-index: 120;
  top: calc(100% + 0.625rem);
  right: 0;
  width: min(18rem, calc(100vw - 2rem));
  overflow: hidden;
  border: 1px solid #000;
  border-radius: 1rem;
  background: #fff;
  padding: 0.75rem;
  box-shadow: 0 1rem 2.5rem rgb(0 0 0 / 18%), 0 0.25rem 0.75rem rgb(0 0 0 / 8%);
  transform-origin: top right;
}

.time-picker-period {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.25rem;
  border-radius: 0.75rem;
  background: #f2f2f2;
  padding: 0.25rem;
}

.time-picker-period button {
  min-height: 2.25rem;
  cursor: pointer;
  border: 0;
  border-radius: 0.625rem;
  background: transparent;
  color: #000;
  font-size: 0.875rem;
  font-weight: 700;
  transition: background-color 140ms ease, color 140ms ease, box-shadow 140ms ease;
}

.time-picker-period button.selected {
  background: #fff;
  color: #000;
  box-shadow: 0 0.125rem 0.5rem rgb(0 0 0 / 14%);
}

.time-picker-columns {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.625rem;
  margin-top: 0.75rem;
}

.time-picker-column {
  min-width: 0;
}

.time-picker-column-label {
  display: block;
  padding: 0 0.5rem 0.375rem;
  color: #000;
  font-size: 0.75rem;
  font-weight: 700;
}

.time-picker-list {
  height: 9.75rem;
  overflow-y: auto;
  overscroll-behavior: contain;
  border-radius: 0.75rem;
  border: 1px solid #000;
  background: #fff;
  padding: 0.25rem;
  scrollbar-color: #888 transparent;
  scrollbar-width: thin;
}

.time-picker-option {
  display: block;
  width: 100%;
  min-height: 2.25rem;
  cursor: pointer;
  border: 0;
  border-radius: 0.625rem;
  background: transparent;
  color: #000;
  font-size: 0.9375rem;
  font-variant-numeric: tabular-nums;
  transition: background-color 120ms ease, color 120ms ease, transform 120ms ease;
}

.time-picker-option:hover {
  background: #ededed;
}

.time-picker-option.selected {
  background: #000;
  color: #fff;
  font-weight: 700;
  box-shadow: 0 0.25rem 0.625rem rgb(0 0 0 / 18%);
}

.time-picker-option:focus-visible,
.time-picker-period button:focus-visible,
.time-picker-done:focus-visible {
  outline: 2px solid #000;
  outline-offset: 2px;
}

.time-picker-done {
  width: 100%;
  min-height: 2.5rem;
  margin-top: 0.75rem;
  cursor: pointer;
  border: 0;
  border-radius: 0.75rem;
  background: #000;
  color: #fff;
  font-size: 0.875rem;
  font-weight: 700;
  transition: background-color 140ms ease, transform 140ms ease;
}

.time-picker-done:hover {
  background: #222;
  transform: translateY(-1px);
}

.time-picker-enter-active,
.time-picker-leave-active {
  transition: opacity 140ms ease, transform 140ms ease;
}

.time-picker-enter-from,
.time-picker-leave-to {
  opacity: 0;
  transform: translateY(-0.375rem) scale(0.98);
}

@media (max-width: 767px) {
  .time-select-pill .time-picker-trigger {
    height: 3.5rem;
  }

  .time-select-pill .time-picker-value,
  .time-select-editor-card .time-picker-value {
    font-size: 1.5rem;
  }

  .time-select-editor-card .time-picker-trigger {
    height: 8.25rem;
    gap: 1.25rem;
  }

  .time-picker-subtitle {
    font-size: 1.25rem;
  }

  .time-picker-panel {
    right: auto;
    left: 0;
    transform-origin: top left;
  }
}

@media (prefers-reduced-motion: reduce) {
  .time-picker-trigger,
  .time-picker-chevron,
  .time-picker-option,
  .time-picker-done,
  .time-picker-enter-active,
  .time-picker-leave-active {
    transition: none;
  }
}
</style>
