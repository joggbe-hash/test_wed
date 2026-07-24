import { beforeEach, describe, expect, it } from 'vitest'
import {
  readStoredSchedule,
  scheduleStorageKeyFor,
  writeStoredSchedule,
} from './scheduleRepository'

describe('scheduleRepository', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('round-trips a valid schedule', () => {
    const key = scheduleStorageKeyFor(7)
    const schedule = {
      tasks: [{
        id: 1,
        title: '測試任務',
        date: '2026-07-24',
        time: '09:00',
        priority: 'high' as const,
        completed: false,
        order: 1,
      }],
      reminders: [],
    }

    expect(writeStoredSchedule(localStorage, key, schedule)).toBe(true)
    expect(readStoredSchedule(key, localStorage)).toEqual({
      ...schedule,
      tasks: [{ ...schedule.tasks[0], importance: 5 }],
    })
  })

  it('rejects malformed persisted data', () => {
    const key = scheduleStorageKeyFor(7)
    localStorage.setItem(key, JSON.stringify({ tasks: [{ id: 'bad' }], reminders: [] }))

    expect(readStoredSchedule(key, localStorage)).toBeNull()
  })
})
