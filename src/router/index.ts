import { createRouter, createWebHashHistory } from 'vue-router'
import LoginPage from '../views/LoginPage.vue'
import FirstPage from '../views/FirstPage.vue'
import ExplorePage from '../views/ExplorePage.vue'
import PersonalPage from '../views/PersonalPage.vue'
import FreqPage from '../views/FreqPage.vue'
import IntroducePage from '../views/IntroducePage.vue'
import SocialPage from '../views/SocialPage.vue'
import TodayTasksPage from '../views/TodayTasksPage.vue'
import TodayRemindersPage from '../views/TodayRemindersPage.vue'

const routes = [
  { path: '/', redirect: '/login' },
  { path: '/login', component: LoginPage },
  { path: '/home', component: FirstPage },
  { path: '/explore', component: ExplorePage },
  { path: '/personal', component: PersonalPage },
  { path: '/freq', component: FreqPage },
  { path: '/tasks/today', component: TodayTasksPage },
  { path: '/reminders/today', component: TodayRemindersPage },
  { path: '/introduce', component: IntroducePage },
  { path: '/social', component: SocialPage },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

export default router
