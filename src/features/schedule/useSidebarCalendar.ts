import { computed, onBeforeUnmount, onMounted, ref, shallowRef, type Ref } from 'vue'

export function useSidebarCalendar(todayKey: Readonly<Ref<string>>) {
  const weekDays = ['SUN', 'MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT']
  const currentDate = shallowRef(new Date())
  const visibleMonth = shallowRef(
    new Date(currentDate.value.getFullYear(), currentDate.value.getMonth(), 1),
  )
  const selectedDateKey = ref<string | null>(null)
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
        isToday: dateKey === todayKey.value,
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

  function setSelectedDateKey(dateKey: string | null) {
    selectedDateKey.value = dateKey
    if (!dateKey) return

    const [year, month] = dateKey.split('-').map(Number)
    if (!year || !month) return
    visibleMonth.value = new Date(year, month - 1, 1)
  }

  function refreshCurrentDate() {
    const previousDate = currentDate.value
    const nextDate = new Date()
    const wasViewingCurrentMonth =
      visibleMonth.value.getFullYear() === previousDate.getFullYear()
      && visibleMonth.value.getMonth() === previousDate.getMonth()

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

  return {
    weekDays,
    selectedDateKey,
    monthLabel,
    calendarCells,
    shiftMonth,
    setSelectedDateKey,
  }
}
