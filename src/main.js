import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import { preloadPageCss } from './composables/usePageCss'
import LoginPage from './views/LoginPage.vue'
import FirstPage from './views/FirstPage.vue'
import ExplorePage from './views/ExplorePage.vue'
import PersonalPage from './views/PersonalPage.vue'
import FreqPage from './views/FreqPage.vue'
import IntroducePage from './views/IntroducePage.vue'
import SocialPage from './views/SocialPage.vue'

const routes = [
  { path: '/', redirect: '/login' },
  { path: '/login', component: LoginPage },
  { path: '/home', component: FirstPage },
  { path: '/explore', component: ExplorePage },
  { path: '/personal', component: PersonalPage },
  { path: '/freq', component: FreqPage },
  { path: '/introduce', component: IntroducePage },
  { path: '/social', component: SocialPage },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

preloadPageCss([
  'index.css',
  'first_page.css',
  'explore_page.css',
  'personal_page.css',
  'freq_page.css',
  'introduce_page.css',
  'social_page.css',
])

createApp(App).use(router).mount('#app')
