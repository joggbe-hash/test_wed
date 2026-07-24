import { readonly, shallowRef } from 'vue'
import { ApiError, checkCurrentSession, type User } from '../api/backendApi'
import { clearScheduleOwner, setScheduleOwner } from './useScheduleMock'
import { useFeedStore } from '../stores/useFeedStore'

const currentUser = shallowRef<User | null>(null)
const isSessionInitialized = shallowRef(false)
let pendingSessionCheck: Promise<User | null> | null = null
let sessionRevision = 0

interface RestoreCurrentSessionOptions {
  force?: boolean
}

function commitCurrentSession(user: User) {
  currentUser.value = user
  isSessionInitialized.value = true
  setScheduleOwner(user.id)
}

function commitLoggedOutSession() {
  currentUser.value = null
  isSessionInitialized.value = true
  clearScheduleOwner()
  useFeedStore().reset()
}

function supersedePendingSessionCheck() {
  sessionRevision += 1
  pendingSessionCheck = null
}

function setCurrentSession(user: User) {
  supersedePendingSessionCheck()
  commitCurrentSession(user)
}

function clearCurrentSession() {
  supersedePendingSessionCheck()
  commitLoggedOutSession()
}

async function restoreCurrentSession(options: RestoreCurrentSessionOptions = {}) {
  if (!options.force && isSessionInitialized.value) return currentUser.value
  if (pendingSessionCheck) return pendingSessionCheck

  const requestRevision = sessionRevision
  const request: Promise<User | null> = checkCurrentSession()
    .then(({ user }) => {
      if (requestRevision !== sessionRevision) return currentUser.value

      commitCurrentSession(user)
      return user
    })
    .catch((error: unknown) => {
      if (requestRevision !== sessionRevision) return currentUser.value

      if (error instanceof ApiError && error.status === 401) {
        commitLoggedOutSession()
        return null
      }

      throw error
    })
    .finally(() => {
      if (pendingSessionCheck === request) {
        pendingSessionCheck = null
      }
    })

  pendingSessionCheck = request
  return request
}

const readonlyCurrentUser = readonly(currentUser)
const readonlySessionInitialized = readonly(isSessionInitialized)

export function useSession() {
  return {
    currentUser: readonlyCurrentUser,
    isSessionInitialized: readonlySessionInitialized,
    setCurrentSession,
    clearCurrentSession,
    restoreCurrentSession,
  }
}
