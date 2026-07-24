export function formatLocalDateKey(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')

  return `${year}-${month}-${day}`
}

const scheduleDayStartHour = 4

export function getLocalTodayKey(date = new Date()) {
  const scheduleDate = new Date(date)
  scheduleDate.setHours(scheduleDate.getHours() - scheduleDayStartHour)

  return formatLocalDateKey(scheduleDate)
}
