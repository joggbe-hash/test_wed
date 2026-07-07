<script setup lang="ts">
import { computed, ref, shallowRef, watch, onMounted, onUnmounted } from 'vue'
import { formatLocalDateKey } from '../utils/date'
import DailyTaskPrompt from './DailyTaskPrompt.vue'

interface InspirationItem {
  id: number
  date: string
  text: string
  imageLabel?: string
}

type SortOrder = 'newest' | 'oldest'

const emit = defineEmits<{
  close: []
}>()

const inspirationTextMaxLength = 700
const searchText = shallowRef('')
const draftText = shallowRef('')
const editText = shallowRef('')
const taskDraftTitle = shallowRef('')
const isTaskPromptOpen = shallowRef(false)
const nextId = shallowRef(6)
const isDateFilterOpen = shallowRef(false)
const defaultStartDate = '2026-06-18'
const todayDateKey = shallowRef(formatLocalDateKey(new Date()))
const [defaultStartYear, defaultStartMonth, defaultStartDay] = parseDateKey(defaultStartDate)
const [initialEndYear, initialEndMonth, initialEndDay] = parseDateKey(todayDateKey.value)
const appliedStartDate = shallowRef(defaultStartDate)
const appliedEndDate = shallowRef(todayDateKey.value)
const appliedSortOrder = shallowRef<SortOrder>('newest')
const pendingStartMonth = shallowRef(defaultStartMonth)
const pendingStartDay = shallowRef(defaultStartDay)
const pendingStartYear = shallowRef(defaultStartYear)
const pendingEndMonth = shallowRef(initialEndMonth)
const pendingEndDay = shallowRef(initialEndDay)
const pendingEndYear = shallowRef(initialEndYear)
const pendingSortOrder = shallowRef<SortOrder>('newest')
const pendingDeleteItemId = shallowRef<number | null>(null)
const editingItemId = shallowRef<number | null>(null)
const items = ref<InspirationItem[]>([
  {
    id: 1,
    date: '2026-06-25',
    text: '去租書店把柯南漫畫裡的壞人都圈出來 *要帶紅筆',
  },
  {
    id: 2,
    date: '2026-06-24',
    text: '舉辦吃麻糬大賽可以解決人口老化問題',
  },
  {
    id: 3,
    date: '2026-06-24',
    text: '蛇的尿道跟肛門是同一個，尿液結晶有時會導致出口堵塞',
    imageLabel: '某貼文縮圖連結',
  },
  {
    id: 4,
    date: '2026-06-18',
    text: '買電擊棒偷電路人',
  },
])

const monthOptions = Array.from({ length: 12 }, (_, index) => index + 1)

const startDayOptions = computed(() => {
  const days = new Date(pendingStartYear.value, pendingStartMonth.value, 0).getDate()
  return Array.from({ length: days }, (_, index) => index + 1)
})

const endDayOptions = computed(() => {
  const days = new Date(pendingEndYear.value, pendingEndMonth.value, 0).getDate()
  return Array.from({ length: days }, (_, index) => index + 1)
})

watch([pendingStartYear, pendingStartMonth], ([year, month]) => {
  const maxDays = new Date(year, month, 0).getDate()
  if (pendingStartDay.value > maxDays) {
    pendingStartDay.value = maxDays
  }
})

watch([pendingEndYear, pendingEndMonth], ([year, month]) => {
  const maxDays = new Date(year, month, 0).getDate()
  if (pendingEndDay.value > maxDays) {
    pendingEndDay.value = maxDays
  }
})

const yearOptions = computed(() => {
  const years = new Set(items.value.map((item) => Number(item.date.slice(0, 4))))
  years.add(Number(todayDateKey.value.slice(0, 4)))
  return Array.from(years).sort((a, b) => b - a)
})

const groupedItems = computed(() => {
  const query = searchText.value.trim().toLowerCase()
  const groups = new Map<string, InspirationItem[]>()
  const startDate = appliedStartDate.value
  const endDate = appliedEndDate.value

  for (const item of items.value) {
    if (item.date < startDate || item.date > endDate) continue
    if (query && !item.text.toLowerCase().includes(query)) continue

    const group = groups.get(item.date) ?? []
    group.push(item)
    groups.set(item.date, group)
  }

  return Array.from(groups.entries())
    .sort(([dateA], [dateB]) =>
      appliedSortOrder.value === 'newest' ? dateB.localeCompare(dateA) : dateA.localeCompare(dateB),
    )
    .map(([date, groupItems]) => ({
      date,
      items: groupItems,
    }))
})

const isDeleteConfirmOpen = computed(() => pendingDeleteItemId.value !== null)

function normalizeInspirationText(text: string) {
  return text.trim().slice(0, inspirationTextMaxLength)
}

function formatDate(date: string) {
  return date.replace(/-/g, '/')
}

function makeDateKey(year: number, month: number, day: number) {
  return `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`
}

function parseDateKey(dateKey: string): [number, number, number] {
  const [year, month, day] = dateKey.split('-').map(Number)
  return [year, month, day]
}

function refreshTodayDateKey() {
  todayDateKey.value = formatLocalDateKey(new Date())
  return todayDateKey.value
}

function setPendingRange(startDate: string, endDate: string) {
  const [startYear, startMonth, startDay] = parseDateKey(startDate)
  const [endYear, endMonth, endDay] = parseDateKey(endDate)

  pendingStartYear.value = startYear
  pendingStartMonth.value = startMonth
  pendingStartDay.value = startDay
  pendingEndYear.value = endYear
  pendingEndMonth.value = endMonth
  pendingEndDay.value = endDay
}

function syncPendingFilter() {
  setPendingRange(appliedStartDate.value, appliedEndDate.value)
  pendingSortOrder.value = appliedSortOrder.value
}

function openDateFilter() {
  syncPendingFilter()
  isDateFilterOpen.value = true
}

function resetDateFilter() {
  setPendingRange(defaultStartDate, refreshTodayDateKey())
  pendingSortOrder.value = 'newest'
}

function applyDateFilter() {
  const startDate = makeDateKey(pendingStartYear.value, pendingStartMonth.value, pendingStartDay.value)
  const endDate = makeDateKey(pendingEndYear.value, pendingEndMonth.value, pendingEndDay.value)

  if (startDate <= endDate) {
    appliedStartDate.value = startDate
    appliedEndDate.value = endDate
  } else {
    appliedStartDate.value = endDate
    appliedEndDate.value = startDate
  }

  appliedSortOrder.value = pendingSortOrder.value
  isDateFilterOpen.value = false
}

function cancelDateFilter() {
  syncPendingFilter()
  isDateFilterOpen.value = false
}

function addDraft() {
  const text = normalizeInspirationText(draftText.value)
  if (!text) return

  const draftDate = refreshTodayDateKey()

  items.value = [
    {
      id: nextId.value,
      date: draftDate,
      text,
    },
    ...items.value,
  ]
  if (draftDate < appliedStartDate.value) {
    appliedStartDate.value = draftDate
  }
  if (draftDate > appliedEndDate.value) {
    appliedEndDate.value = draftDate
  }
  appliedSortOrder.value = 'newest'
  if (isDateFilterOpen.value) {
    syncPendingFilter()
  }
  nextId.value += 1
  draftText.value = ''
}

function deleteItem(id: number) {
  items.value = items.value.filter((item) => item.id !== id)
  if (editingItemId.value === id) {
    cancelEditItem()
  }
}

function requestDeleteItem(id: number) {
  pendingDeleteItemId.value = id
}

function cancelDeleteItem() {
  pendingDeleteItemId.value = null
}

function confirmDeleteItem() {
  if (pendingDeleteItemId.value === null) return

  deleteItem(pendingDeleteItemId.value)
  pendingDeleteItemId.value = null
}

function openTaskPrompt(item: InspirationItem) {
  const title = normalizeInspirationText(item.text)
  if (!title) return

  pendingDeleteItemId.value = null
  editingItemId.value = null
  taskDraftTitle.value = title
  isTaskPromptOpen.value = true
}

function closeTaskPrompt() {
  isTaskPromptOpen.value = false
  taskDraftTitle.value = ''
}

function startEditItem(item: InspirationItem) {
  pendingDeleteItemId.value = null
  editingItemId.value = item.id
  editText.value = item.text.slice(0, inspirationTextMaxLength)
}

function cancelEditItem() {
  editingItemId.value = null
  editText.value = ''
}

function saveEditItem() {
  const text = normalizeInspirationText(editText.value)
  if (!text || editingItemId.value === null) return

  items.value = items.value.map((item) =>
    item.id === editingItemId.value
      ? { ...item, text }
      : item,
  )
  cancelEditItem()
}

const handleKeyDown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    if (isTaskPromptOpen.value) {
      closeTaskPrompt()
      return
    }

    if (isDeleteConfirmOpen.value) {
      cancelDeleteItem()
      return
    }

    if (editingItemId.value !== null) {
      cancelEditItem()
      return
    }

    emit('close')
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
})
</script>

<template>
  <Teleport to="body">
    <div class="inspiration-backdrop" role="presentation" @click.self="emit('close')">
      <section class="inspiration-modal" role="dialog" aria-modal="true" aria-labelledby="inspiration-title">
        <header class="inspiration-header">
          <h2 id="inspiration-title">靈感清單</h2>
          <button type="button" class="inspiration-close" aria-label="關閉" @click="emit('close')">
            <span aria-hidden="true">&times;</span>
          </button>
        </header>

        <div class="inspiration-content">
          <div class="inspiration-toolbar">
            <label class="inspiration-search">
              <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
                <circle cx="11" cy="11" r="7"></circle>
                <path d="M20 20L16.5 16.5"></path>
              </svg>
              <input v-model="searchText" type="search" placeholder="在清單內搜尋關鍵字">
            </label>

            <div class="inspiration-date-filter">
              <button
                type="button"
                class="inspiration-date-filter-trigger"
                :aria-expanded="isDateFilterOpen"
                aria-controls="inspiration-date-panel"
                @click="isDateFilterOpen ? cancelDateFilter() : openDateFilter()"
              >
                所有日期 <span aria-hidden="true">&gt;</span>
              </button>

              <div v-if="isDateFilterOpen" id="inspiration-date-panel" class="inspiration-date-panel">
                <button type="button" class="inspiration-filter-reset" @click="resetDateFilter">重設</button>

                <fieldset class="inspiration-date-section">
                  <legend>開始日期</legend>
                  <div class="inspiration-date-selects">
                    <select v-model.number="pendingStartMonth" aria-label="開始月份">
                      <option v-for="month in monthOptions" :key="month" :value="month">{{ month }}月</option>
                    </select>
                    <select v-model.number="pendingStartDay" aria-label="開始日期">
                      <option v-for="day in startDayOptions" :key="day" :value="day">{{ day }}</option>
                    </select>
                    <select v-model.number="pendingStartYear" aria-label="開始年份">
                      <option v-for="year in yearOptions" :key="year" :value="year">{{ year }}</option>
                    </select>
                  </div>
                </fieldset>

                <fieldset class="inspiration-date-section">
                  <legend>結束日期</legend>
                  <div class="inspiration-date-selects">
                    <select v-model.number="pendingEndMonth" aria-label="結束月份">
                      <option v-for="month in monthOptions" :key="month" :value="month">{{ month }}月</option>
                    </select>
                    <select v-model.number="pendingEndDay" aria-label="結束日期">
                      <option v-for="day in endDayOptions" :key="day" :value="day">{{ day }}</option>
                    </select>
                    <select v-model.number="pendingEndYear" aria-label="結束年份">
                      <option v-for="year in yearOptions" :key="year" :value="year">{{ year }}</option>
                    </select>
                  </div>
                </fieldset>

                <fieldset class="inspiration-date-section">
                  <legend>排序依據</legend>
                  <div class="inspiration-sort-options">
                    <button
                      type="button"
                      :class="{ active: pendingSortOrder === 'newest' }"
                      @click="pendingSortOrder = 'newest'"
                    >
                      從新到舊
                    </button>
                    <button
                      type="button"
                      :class="{ active: pendingSortOrder === 'oldest' }"
                      @click="pendingSortOrder = 'oldest'"
                    >
                      從舊到新
                    </button>
                  </div>
                </fieldset>

                <div class="inspiration-date-actions">
                  <button type="button" class="inspiration-apply-filter" @click="applyDateFilter">套用</button>
                  <button type="button" class="inspiration-cancel-filter" @click="cancelDateFilter">取消</button>
                </div>
              </div>
            </div>
          </div>

          <form class="inspiration-draft" @submit.prevent="addDraft">
            <input
              v-model="draftText"
              type="text"
              :maxlength="inspirationTextMaxLength"
              placeholder="今天有甚麼想法嗎?馬上記錄下來吧!"
            >
          </form>

          <div class="inspiration-groups">
            <section v-for="group in groupedItems" :key="group.date" class="inspiration-group">
              <h3>{{ formatDate(group.date) }}</h3>

              <article v-for="(item, index) in group.items" :key="item.id" class="inspiration-item">
                <span class="inspiration-index">{{ index + 1 }}.</span>
                <div v-if="item.imageLabel" class="inspiration-thumb">{{ item.imageLabel }}</div>
                <div v-if="editingItemId === item.id" class="inspiration-edit-panel">
                  <textarea
                    v-model="editText"
                    :maxlength="inspirationTextMaxLength"
                    rows="3"
                    aria-label="編輯靈感內容"
                    @keydown.ctrl.enter.prevent="saveEditItem"
                  ></textarea>
                  <div class="inspiration-edit-footer">
                    <span>{{ editText.length }}/{{ inspirationTextMaxLength }}</span>
                    <div>
                      <button type="button" @click="saveEditItem">儲存</button>
                      <button type="button" @click="cancelEditItem">取消</button>
                    </div>
                  </div>
                </div>
                <p v-else>{{ item.text }}</p>
                <div class="inspiration-item-actions" aria-label="靈感操作">
                  <button
                    type="button"
                    aria-label="推至今日任務"
                    title="推至今日任務"
                    @click="openTaskPrompt(item)"
                  >
                    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M12 4L5 11H9V20H15V11H19L12 4Z"></path>
                      <path d="M7 20H17"></path>
                    </svg>
                  </button>
                  <button type="button" aria-label="編輯" title="編輯" @click="startEditItem(item)">
                    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M4 20H8L18.5 9.5L14.5 5.5L4 16V20Z"></path>
                      <path d="M13.5 6.5L17.5 10.5"></path>
                    </svg>
                  </button>
                  <button type="button" aria-label="刪除" title="刪除" @click="requestDeleteItem(item.id)">
                    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M5 7H19"></path>
                      <path d="M10 11V17"></path>
                      <path d="M14 11V17"></path>
                      <path d="M8 7L9 20H15L16 7"></path>
                      <path d="M9 7V4H15V7"></path>
                    </svg>
                  </button>
                </div>
              </article>
            </section>
          </div>
        </div>

        <div v-if="isDeleteConfirmOpen" class="inspiration-delete-layer" @click.self="cancelDeleteItem">
          <section
            class="inspiration-delete-dialog"
            role="alertdialog"
            aria-modal="true"
            aria-labelledby="inspiration-delete-title"
            aria-describedby="inspiration-delete-description"
          >
            <header class="inspiration-delete-header">
              <h3 id="inspiration-delete-title">確定?</h3>
              <button type="button" class="inspiration-delete-close" aria-label="取消刪除" @click="cancelDeleteItem">
                <span aria-hidden="true">&times;</span>
              </button>
            </header>

            <div class="inspiration-delete-body">
              <p id="inspiration-delete-description">刪除的資料將一去不復返，確定刪除嗎?</p>
              <div class="inspiration-delete-actions">
                <button type="button" class="inspiration-delete-confirm" @click="confirmDeleteItem">確定刪除</button>
                <button type="button" class="inspiration-delete-cancel" @click="cancelDeleteItem">取消</button>
              </div>
            </div>
          </section>
        </div>
      </section>

      <DailyTaskPrompt
        v-if="isTaskPromptOpen"
        :initial-title="taskDraftTitle"
        close-after-submit
        @close="closeTaskPrompt"
      />
    </div>
  </Teleport>
</template>
