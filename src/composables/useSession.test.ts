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

vi.mock('./useSchedule', () => ({
  clearScheduleOwner: dependencies.clearScheduleOwner,
  setScheduleOwner: dependencies.setScheduleOwner,
}))

import { ApiError } from '../api/backendApi'
import { useSession } from './useSession'
import { useFeedStore } from '../stores/useFeedStore'

const firstUser = { id: 1, username: 'first', email: 'first@example.com' }
const secondUser = { id: 2, username: 'second', email: 'second@example.com' }

describe('useSession', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    useSession().clearCurrentSession()
    dependencies.checkCurrentSession.mockReset()
    dependencies.clearScheduleOwner.mockReset()
    dependencies.setScheduleOwner.mockReset()
  })

  it('clears account-scoped feed data when a forced session refresh changes users', async () => {
    const session = useSession()
    session.setCurrentSession(firstUser)
    useFeedStore().prependPost({
      id: 10,
      user_id: firstUser.id,
      username: firstUser.username,
      visibility: 'public',
      image_status: 'none',
      created_at: '',
    })
    dependencies.checkCurrentSession.mockResolvedValue({ user: secondUser })

    await session.restoreCurrentSession({ force: true })

    expect(session.currentUser.value).toEqual(secondUser)
    expect(useFeedStore().posts).toEqual([])
  })

  it('preserves feed data when a forced session refresh confirms the same user', async () => {
    const session = useSession()
    session.setCurrentSession(firstUser)
    const feedStore = useFeedStore()
    feedStore.prependPost({
      id: 10,
      user_id: firstUser.id,
      username: firstUser.username,
      visibility: 'public',
      image_status: 'none',
      created_at: '',
    })
    dependencies.checkCurrentSession.mockResolvedValue({ user: firstUser })

    await session.restoreCurrentSession({ force: true })

    expect(feedStore.posts).toHaveLength(1)
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
