<script setup lang="ts">
import { computed } from 'vue'

type SortOrder = 'newest' | 'oldest'

const props = defineProps<{
  open: boolean
  monthOptions: number[]
  startDayOptions: number[]
  endDayOptions: number[]
  yearOptions: number[]
}>()

const emit = defineEmits<{
  open: []
  reset: []
  apply: []
  cancel: []
}>()

const startMonth = defineModel<number>('startMonth', { required: true })
const startDay = defineModel<number>('startDay', { required: true })
const startYear = defineModel<number>('startYear', { required: true })
const endMonth = defineModel<number>('endMonth', { required: true })
const endDay = defineModel<number>('endDay', { required: true })
const endYear = defineModel<number>('endYear', { required: true })
const sortOrder = defineModel<SortOrder>('sortOrder', { required: true })

const startDateKey = computed(() => startYear.value * 10_000 + startMonth.value * 100 + startDay.value)
const endDateKey = computed(() => endYear.value * 10_000 + endMonth.value * 100 + endDay.value)
const isDateRangeValid = computed(() => endDateKey.value >= startDateKey.value)

function applyFilter() {
  if (!isDateRangeValid.value) return
  emit('apply')
}
</script>

<template>
  <div class="inspiration-date-filter">
    <button
      type="button"
      class="inspiration-date-filter-trigger"
      :aria-expanded="open"
      aria-controls="inspiration-date-panel"
      @click="open ? emit('cancel') : emit('open')"
    >
      所有日期 <span aria-hidden="true">&gt;</span>
    </button>

    <div v-if="props.open" id="inspiration-date-panel" class="inspiration-date-panel">
      <button type="button" class="inspiration-filter-reset" @click="emit('reset')">重設</button>

      <fieldset class="inspiration-date-section">
        <legend>開始日期</legend>
        <div class="inspiration-date-selects">
          <select v-model.number="startMonth" aria-label="開始月份">
            <option v-for="month in monthOptions" :key="month" :value="month">{{ month }}月</option>
          </select>
          <select v-model.number="startDay" aria-label="開始日期">
            <option v-for="day in startDayOptions" :key="day" :value="day">{{ day }}</option>
          </select>
          <select v-model.number="startYear" aria-label="開始年份">
            <option v-for="year in yearOptions" :key="year" :value="year">{{ year }}</option>
          </select>
        </div>
      </fieldset>

      <fieldset class="inspiration-date-section">
        <legend>結束日期</legend>
        <div class="inspiration-date-selects">
          <select
            v-model.number="endMonth"
            aria-label="結束月份"
            :aria-invalid="!isDateRangeValid"
            :aria-describedby="!isDateRangeValid ? 'inspiration-date-error' : undefined"
          >
            <option v-for="month in monthOptions" :key="month" :value="month">{{ month }}月</option>
          </select>
          <select
            v-model.number="endDay"
            aria-label="結束日期"
            :aria-invalid="!isDateRangeValid"
            :aria-describedby="!isDateRangeValid ? 'inspiration-date-error' : undefined"
          >
            <option v-for="day in endDayOptions" :key="day" :value="day">{{ day }}</option>
          </select>
          <select
            v-model.number="endYear"
            aria-label="結束年份"
            :aria-invalid="!isDateRangeValid"
            :aria-describedby="!isDateRangeValid ? 'inspiration-date-error' : undefined"
          >
            <option v-for="year in yearOptions" :key="year" :value="year">{{ year }}</option>
          </select>
        </div>
      </fieldset>

      <p
        v-if="!isDateRangeValid"
        id="inspiration-date-error"
        class="inspiration-date-error"
        role="alert"
        aria-live="polite"
      >
        結束日期不得早於開始日期
      </p>

      <fieldset class="inspiration-date-section">
        <legend>排序依據</legend>
        <div class="inspiration-sort-options">
          <button type="button" :class="{ active: sortOrder === 'newest' }" @click="sortOrder = 'newest'">
            從新到舊
          </button>
          <button type="button" :class="{ active: sortOrder === 'oldest' }" @click="sortOrder = 'oldest'">
            從舊到新
          </button>
        </div>
      </fieldset>

      <div class="inspiration-date-actions">
        <button
          type="button"
          class="inspiration-apply-filter"
          :disabled="!isDateRangeValid"
          @click="applyFilter"
        >
          套用
        </button>
        <button type="button" class="inspiration-cancel-filter" @click="emit('cancel')">取消</button>
      </div>
    </div>
  </div>
</template>
