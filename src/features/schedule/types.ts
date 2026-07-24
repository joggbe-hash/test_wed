export type Priority = 'high' | 'medium' | 'low'
export type TaskImportance = 1 | 2 | 3 | 4 | 5

export interface TaskItem {
  id: number
  title: string
  note?: string
  date: string
  time: string
  endTime?: string
  priority: Priority
  importance?: TaskImportance
  completed: boolean
  order: number
}

export interface ReminderItem {
  id: number
  title: string
  date: string
  time: string
  endTime?: string
  note: string
}

export interface StoredSchedule {
  tasks: TaskItem[]
  reminders: ReminderItem[]
}

export type LegacyScheduleDecisionResult =
  | { ok: true }
  | { ok: false; message: string }

export const priorityMeta: Record<Priority, { label: string; rank: number }> = {
  high: { label: '高', rank: 1 },
  medium: { label: '中', rank: 2 },
  low: { label: '低', rank: 3 },
}

export function priorityToImportance(priority: Priority): TaskImportance {
  if (priority === 'high') return 5
  if (priority === 'medium') return 3
  return 1
}

export function taskImportanceCount(
  task: Pick<TaskItem, 'importance' | 'priority'>,
): TaskImportance {
  return task.importance ?? priorityToImportance(task.priority)
}
