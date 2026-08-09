import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import './style.css'
import router from './router'
import { setUnauthorizedHandler } from './api/unauthorizedHandler'
import { useSession } from './composables/useSession'

const app = createApp(App)
app.use(createPinia())
app.use(router)

const { restoreCurrentSession } = useSession()
let sessionSynchronization: Promise<void> | null = null

function synchronizeSession() {
  if (sessionSynchronization) return

  sessionSynchronization = restoreCurrentSession({ force: true })
    .then(async (user) => {
      const currentRoute = router.currentRoute.value
      if (!user && currentRoute.meta.requiresAuth) {
        await router.replace({ path: '/login', query: { redirect: currentRoute.fullPath } })
      }
    })
    .catch((error: unknown) => {
      console.error('Unable to synchronize the current session', error)
    })
    .finally(() => {
      sessionSynchronization = null
    })
}

window.addEventListener('focus', synchronizeSession)
document.addEventListener('visibilitychange', () => {
  if (document.visibilityState === 'visible') synchronizeSession()
})

setUnauthorizedHandler(async () => {
  const currentRoute = router.currentRoute.value
  await router.push({
    path: '/login',
    query: currentRoute.path === '/login' ? {} : { redirect: currentRoute.fullPath },
  })
})

app.mount('#app')
