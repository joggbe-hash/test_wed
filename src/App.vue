<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { RouterView } from 'vue-router'
import ComposeModal from './components/ComposeModal.vue'
import DailyTaskPrompt from './components/DailyTaskPrompt.vue'
import IntroducePage from './views/IntroducePage.vue'
import { showComposeModal } from './composables/useComposeModal'
import { maybeOpenDailyTaskPrompt, showDailyTaskPrompt } from './composables/useDailyTaskPrompt'
import { showIntroduceModal } from './composables/useModal'
import { useSchedule } from './composables/useSchedule'
import { useSession } from './composables/useSession'

const route = useRoute()
const { currentUser } = useSession()
const { isScheduleReady, scheduleErrorMessage } = useSchedule()
const routedComponentKey = computed(() => route.meta.requiresAuth
  ? `${route.path}:${currentUser.value?.id ?? 'signed-out'}`
  : route.path)
const isRoutedContentInactive = computed(() =>
  showComposeModal.value
    || showIntroduceModal.value
    || showDailyTaskPrompt.value,
)

watch(
  () => [route.path, isScheduleReady.value] as const,
  ([path, scheduleReady]) => {
    maybeOpenDailyTaskPrompt(path, scheduleReady)
  },
  { immediate: true },
)
</script>

<template>
  <div
    data-app-route-content
    :inert="isRoutedContentInactive"
    :aria-hidden="isRoutedContentInactive ? 'true' : undefined"
    tabindex="-1"
  >
    <RouterView v-slot="{ Component }">
      <transition name="page-slide" mode="out-in">
        <component :is="Component" :key="routedComponentKey" />
      </transition>
    </RouterView>

  </div>

  <transition name="page-slide">
    <IntroducePage v-if="showIntroduceModal" overlay />
  </transition>
  <ComposeModal v-if="showComposeModal" />
  <DailyTaskPrompt v-if="showDailyTaskPrompt" />
  <p
    v-if="scheduleErrorMessage"
    class="schedule-storage-error"
    role="alert"
    aria-live="assertive"
  >
    {{ scheduleErrorMessage }}
  </p>
</template>

