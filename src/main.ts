import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import './style.css'
import router from './router'
import { setUnauthorizedHandler } from './api/unauthorizedHandler'

const app = createApp(App)
app.use(createPinia())
app.use(router)

setUnauthorizedHandler(async () => {
  const currentRoute = router.currentRoute.value
  await router.push({
    path: '/login',
    query: currentRoute.path === '/login' ? {} : { redirect: currentRoute.fullPath },
  })
})

app.mount('#app')
