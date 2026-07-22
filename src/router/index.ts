import { createRouter, createWebHashHistory } from 'vue-router'
import LoginPage from '../views/LoginPage.vue'
import ForgotPasswordPage from '../views/ForgotPasswordPage.vue'
import FirstPage from '../views/FirstPage.vue'
import ExplorePage from '../views/ExplorePage.vue'
import PersonalPage from '../views/PersonalPage.vue'
import FreqPage from '../views/FreqPage.vue'
import IntroducePage from '../views/IntroducePage.vue'
import SocialPage from '../views/SocialPage.vue'
import TodayTasksPage from '../views/TodayTasksPage.vue'
import TodayRemindersPage from '../views/TodayRemindersPage.vue'
import { useSession } from '../composables/useSession'

const { restoreCurrentSession } = useSession()

const routes = [
  { path: '/', redirect: '/login' },
  { path: '/login', component: LoginPage, meta: { title: '登入｜Type WSP' } },
  { path: '/forgot-password', component: ForgotPasswordPage, meta: { title: '忘記密碼｜Type WSP' } },
  { path: '/home', component: FirstPage, meta: { requiresAuth: true, title: '首頁｜Type WSP' } },
  { path: '/explore', component: ExplorePage, meta: { requiresAuth: true, title: '探索｜Type WSP' } },
  { path: '/personal', component: PersonalPage, meta: { requiresAuth: true, title: '個人頁｜Type WSP' } },
  { path: '/freq', component: FreqPage, meta: { requiresAuth: true, title: '常用功能｜Type WSP' } },
  { path: '/tasks/today', component: TodayTasksPage, meta: { requiresAuth: true, title: '今日任務｜Type WSP' } },
  { path: '/reminders/today', component: TodayRemindersPage, meta: { requiresAuth: true, title: '今日提醒｜Type WSP' } },
  { path: '/introduce', component: IntroducePage, meta: { requiresAuth: true, title: '使用說明｜Type WSP' } },
  { path: '/social', component: SocialPage, meta: { requiresAuth: true, title: '社群｜Type WSP' } },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach(async (to) => {
  if (!to.meta.requiresAuth) return true

  try {
    const user = await restoreCurrentSession()
    if (user) return true
  } catch (error) {
    console.error('Unable to verify the current session', error)
    return {
      path: '/login',
      query: {
        redirect: to.fullPath,
        sessionError: 'unavailable',
      },
    }
  }

  return {
    path: '/login',
    query: { redirect: to.fullPath },
  }
})

router.afterEach((to) => {
  document.title = typeof to.meta.title === 'string'
    ? to.meta.title
    : 'Type WSP｜社群與每日排程'
})

export default router
