<script setup lang="ts">
import { computed, onMounted, shallowRef } from 'vue'
import { useRouter } from 'vue-router'
import MainLayout from '../layouts/MainLayout.vue'
import AccountSessionPanel from '../components/settings/AccountSessionPanel.vue'
import SettingsActionCard from '../components/settings/SettingsActionCard.vue'
import { logoutAccount } from '../api/backendApi'
import {
  fetchFreqData,
  type FreqActionId,
  type FreqActionSummary,
} from '../api/timedApi'
import { showIntroduceModal } from '../composables/useModal'
import { useSession } from '../composables/useSession'

interface SettingsAction {
  id: FreqActionId
  title: string
  description: string
  icon: string
  actionLabel: string
  path?: string
  notice?: string
}

const router = useRouter()
const { clearCurrentSession, currentUser } = useSession()
const isLoading = shallowRef(true)
const isLoggingOut = shallowRef(false)
const loadErrorMessage = shallowRef('')
const logoutErrorMessage = shallowRef('')
const actionNotice = shallowRef('')
const actionSummaries = shallowRef<FreqActionSummary[]>([])

const actionMetadata: Record<FreqActionId, Omit<SettingsAction, 'id' | 'title'>> = {
  profile: {
    description: '查看個人頁面，管理公開身分與已發布的內容。',
    icon: 'account_circle',
    actionLabel: '查看個人頁',
    path: '/personal',
  },
  explore: {
    description: '回到探索頁，繼續尋找感興趣的主題與社群。',
    icon: 'article',
    actionLabel: '前往探索',
    path: '/explore',
  },
  notifications: {
    description: '集中查看提醒、互動消息與重要系統通知。',
    icon: 'notifications',
    actionLabel: '查看狀態',
    notice: '通知中心正在準備中，之後會在這裡集中顯示所有提醒。',
  },
  settings: {
    description: '調整帳號、隱私與個人化使用體驗。',
    icon: 'format_list_bulleted',
    actionLabel: '查看狀態',
    notice: '更多帳號與隱私設定正在準備中。',
  },
}

const settingsActions = computed<SettingsAction[]>(() =>
  actionSummaries.value.map(({ id, title }) => ({
    id,
    title,
    ...actionMetadata[id],
  })),
)

const displayName = computed(() => currentUser.value?.username?.trim() || '使用者')
const displayEmail = computed(() => currentUser.value?.email || '目前帳號')
const profileInitial = computed(() => Array.from(displayName.value)[0]?.toLocaleUpperCase() || 'U')

function resetScreenState() {
  actionSummaries.value = []
  loadErrorMessage.value = ''
  logoutErrorMessage.value = ''
  actionNotice.value = ''
  showIntroduceModal.value = false
}

async function handleAction(action: SettingsAction) {
  actionNotice.value = ''

  if (action.path) {
    await router.push(action.path)
    return
  }

  actionNotice.value = action.notice ?? '此功能正在準備中。'
}

async function handleLogout() {
  if (isLoggingOut.value) return

  isLoggingOut.value = true
  logoutErrorMessage.value = ''

  try {
    await logoutAccount()
    clearCurrentSession()
    resetScreenState()
    await router.replace('/login')
  } catch (error) {
    logoutErrorMessage.value = error instanceof Error ? error.message : '登出失敗，請稍後再試'
  } finally {
    isLoggingOut.value = false
  }
}

async function loadSettings() {
  isLoading.value = true
  loadErrorMessage.value = ''

  try {
    const response = await fetchFreqData()
    actionSummaries.value = response.data.actions
  } catch (error) {
    actionSummaries.value = []
    loadErrorMessage.value = error instanceof Error ? error.message : '載入資料失敗'
  } finally {
    isLoading.value = false
  }
}

onMounted(() => {
  void loadSettings()
})
</script>

<template>
  <MainLayout
    active-nav="freq"
    layout-class="freq-settings-layout"
    sidebar-class="freq-settings-sidebar"
    feed-class="freq-settings-feed"
  >
    <div v-if="isLoading" class="settings-loading" role="status" aria-live="polite">
      <span class="sr-only">設定中心載入中...</span>
      <div class="settings-loading__heading"></div>
      <div class="settings-loading__profile"></div>
      <div class="settings-loading__grid" aria-hidden="true">
        <div v-for="index in 4" :key="index" class="settings-loading__card"></div>
      </div>
    </div>

    <main v-else class="settings-shell">
      <header class="settings-heading">
        <div>
          <p class="settings-heading__eyebrow">ACCOUNT CENTER</p>
          <h1 class="settings-heading__title text-balance">個人與設定</h1>
        </div>
      </header>

      <section class="profile-summary" aria-label="目前登入帳號">
        <div class="profile-summary__identity">
          <div class="profile-summary__avatar" aria-hidden="true">{{ profileInitial }}</div>
          <div class="profile-summary__copy">
            <span class="profile-summary__label">目前登入</span>
            <strong class="profile-summary__name text-balance">{{ displayName }}</strong>
            <span class="profile-summary__email text-pretty">{{ displayEmail }}</span>
          </div>
        </div>
        <div class="profile-summary__status">
          <span class="profile-summary__status-dot" aria-hidden="true"></span>
          帳號連線正常
        </div>
      </section>

      <section class="settings-directory" aria-labelledby="settings-directory-title">
        <div class="settings-section-heading">
          <div>
            <p class="settings-section-heading__eyebrow">DIRECTORY</p>
            <h2 id="settings-directory-title" class="settings-section-heading__title text-balance">
              設定目錄
            </h2>
          </div>
          <p class="settings-section-heading__count tabular-nums">
            {{ String(settingsActions.length).padStart(2, '0') }} ITEMS
          </p>
        </div>

        <div v-if="loadErrorMessage" class="settings-load-error" role="alert">
          <div>
            <strong class="settings-load-error__title text-balance">設定資料暫時無法載入</strong>
            <p class="settings-load-error__description text-pretty">{{ loadErrorMessage }}</p>
          </div>
          <button type="button" class="settings-load-error__retry" @click="loadSettings">
            重新載入
          </button>
        </div>

        <div v-else class="settings-action-grid">
          <SettingsActionCard
            v-for="(action, index) in settingsActions"
            :key="action.id"
            :title="action.title"
            :description="action.description"
            :icon="action.icon"
            :action-label="action.actionLabel"
            :position="index + 1"
            @select="handleAction(action)"
          />
        </div>

        <p v-if="actionNotice" class="settings-action-notice text-pretty" role="status" aria-live="polite">
          {{ actionNotice }}
        </p>
      </section>

      <AccountSessionPanel
        :is-logging-out="isLoggingOut"
        :error-message="logoutErrorMessage"
        @logout="handleLogout"
      />
    </main>
  </MainLayout>
</template>

<style scoped>
@reference "../style.css";

.settings-shell,
.settings-loading {
  @apply mx-auto flex w-full max-w-[1180px] flex-col gap-8;
}

.settings-heading {
  @apply grid grid-cols-[minmax(0,1fr)_minmax(320px,0.75fr)] items-end gap-12 border-b border-[#b9aa9b] pb-8 max-md:grid-cols-1 max-md:gap-4 max-md:pb-6;
}

.settings-heading__eyebrow,
.settings-section-heading__eyebrow {
  @apply text-xs font-bold text-[#705141];
}

.settings-heading__title {
  @apply mt-3 text-5xl font-bold leading-none text-ink-warm max-md:text-4xl;
}

.settings-heading__description {
  @apply max-w-[44ch] text-base leading-7 text-[#6f665f];
}

.profile-summary {
  @apply flex min-h-52 items-center justify-between gap-10 rounded-3xl border border-brown-dark bg-brown px-9 py-8 text-white shadow-md max-md:flex-col max-md:items-start max-md:gap-7 max-md:px-6 max-md:py-7;
}

.profile-summary__identity {
  @apply flex min-w-0 items-center gap-5;
}

.profile-summary__avatar {
  @apply flex size-20 shrink-0 items-center justify-center rounded-full border border-white/40 bg-brown-dark text-3xl font-bold shadow-sm max-md:size-16 max-md:text-2xl;
}

.profile-summary__copy {
  @apply flex min-w-0 flex-col;
}

.profile-summary__label {
  @apply text-xs font-semibold text-white/70;
}

.profile-summary__name {
  @apply mt-2 truncate text-3xl font-bold leading-tight max-md:text-2xl;
}

.profile-summary__email {
  @apply mt-2 truncate text-sm text-white/75;
}

.profile-summary__status {
  @apply flex shrink-0 items-center gap-3 rounded-full border border-white/30 bg-white/10 px-4 py-2 text-xs font-semibold;
}

.profile-summary__status-dot {
  @apply size-2 rounded-full bg-[#b9d6a3];
}

.settings-directory {
  @apply flex flex-col gap-5;
}

.settings-section-heading {
  @apply flex items-end justify-between gap-6;
}

.settings-section-heading__title {
  @apply mt-2 text-2xl font-bold text-ink-warm;
}

.settings-section-heading__count {
  @apply text-xs font-semibold text-[#625a54];
}

.settings-action-grid {
  @apply grid grid-cols-2 gap-5 max-md:grid-cols-1;
}

.settings-action-notice {
  @apply rounded-xl border border-border-soft bg-surface-warm px-5 py-4 text-sm leading-6 text-brown shadow-sm;
}

.settings-load-error {
  @apply flex items-center justify-between gap-8 rounded-2xl border border-[#e3c7c7] bg-[#fff7f7] p-6 max-md:flex-col max-md:items-stretch max-md:gap-5;
}

.settings-load-error__title {
  @apply text-base font-bold text-danger-strong;
}

.settings-load-error__description {
  @apply mt-2 text-sm leading-6 text-[#7b5555];
}

.settings-load-error__retry {
  @apply min-h-11 shrink-0 rounded-xl border border-[#cc8f8f] bg-white px-5 py-3 text-sm font-bold text-danger-strong hover:border-danger-strong hover:bg-[#fff0f0] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-danger-strong;
  font: inherit;
}

.settings-loading__heading {
  @apply h-24 w-full rounded-2xl bg-white/60;
}

.settings-loading__profile {
  @apply h-52 w-full rounded-3xl bg-brown/35;
}

.settings-loading__grid {
  @apply grid grid-cols-2 gap-5 max-md:grid-cols-1;
}

.settings-loading__card {
  @apply h-64 rounded-2xl border border-border-soft bg-white/55;
}
</style>
