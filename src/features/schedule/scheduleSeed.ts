import type { StoredSchedule } from './types'

export function cloneSeedSchedule(): StoredSchedule {
  return {
    tasks: [],
    reminders: [],
  }
}
