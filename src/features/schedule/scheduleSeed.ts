import { getLocalTodayKey } from '../../utils/date'
import type { ReminderItem, StoredSchedule, TaskItem } from './types'

const seedTasks: Array<Omit<TaskItem, 'date'>> = [
  { id: 1, title: '完成專題流程圖', time: '09:30', priority: 'high', importance: 5, completed: false, order: 1 },
  { id: 2, title: '整理貼文互動版面', time: '11:00', priority: 'high', importance: 5, completed: false, order: 2 },
  { id: 3, title: '確認左側邊欄內容', time: '15:20', priority: 'medium', importance: 3, completed: true, order: 3 },
]

const seedReminders: Array<Omit<ReminderItem, 'date'>> = [
  { id: 1, title: '提醒', time: '08:50', note: '檢查今天任務' },
  { id: 2, title: '回顧', time: '21:00', note: '記下今天想做的事' },
]

export function cloneSeedSchedule(date = getLocalTodayKey()): StoredSchedule {
  return {
    tasks: seedTasks.map((task) => ({ ...task, date })),
    reminders: seedReminders.map((reminder) => ({ ...reminder, date })),
  }
}
