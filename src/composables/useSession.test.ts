import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const dependencies = vi.hoisted(() => ({
  checkCurrentSession: vi.fn(),
  clearScheduleOwner: vi.fn(),
  setScheduleOwner: vi.fn(),
}))

vi.mock('../api/backendApi', () => {
  class ApiError extends Error {
    constructor(public status: number, message: string) {
      super(message)
    }
  }

  return {
    ApiError,
    checkCurrentSession: dependencies.checkCurrentSession,
  }
})

vi.mock('./useScheduleMock', () => ({
  clearScheduleOwner: dependencies.clearScheduleOwner,
  setScheduleOwner: dependencies.setScheduleOwner,
}))

import { ApiError } from '../api/backendApi'
import { useSession } from './useSession'

describe('useSession', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    dependencies.checkCurrentSession.mockReset()
    dependencies.clearScheduleOwner.mockReset()
    dependencies.setScheduleOwner.mockReset()
  })

  it('turns an unauthorized session check into logged-out state', async () => {
    dependencies.checkCurrentSession.mockRejectedValue(new ApiError(401, 'unauthorized'))
    const session = useSession()

    await expect(session.restoreCurrentSession({ force: true })).resolves.toBeNull()
    expect(session.currentUser.value).toBeNull()
    expect(session.isSessionInitialized.value).toBe(true)
    expect(dependencies.clearScheduleOwner).toHaveBeenCalledOnce()
  })
})
