import { describe, expect, it } from 'vitest'
import { cloneSeedSchedule } from './scheduleSeed'

describe('cloneSeedSchedule', () => {
  it('starts a new schedule without sample tasks or reminders', () => {
    const schedule = cloneSeedSchedule()

    expect(schedule.tasks).toEqual([])
    expect(schedule.reminders).toEqual([])
  })
})
