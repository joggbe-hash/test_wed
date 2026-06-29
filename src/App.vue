<script setup lang="ts">
import { watch } from 'vue'
import { useRoute } from 'vue-router'
import { RouterView } from 'vue-router'
import ComposeModal from './components/ComposeModal.vue'
import DailyTaskPrompt from './components/DailyTaskPrompt.vue'
import IntroducePage from './views/IntroducePage.vue'
import { showComposeModal } from './composables/useComposeModal'
import { maybeOpenDailyTaskPrompt, showDailyTaskPrompt } from './composables/useDailyTaskPrompt'
import { showIntroduceModal } from './composables/useModal'

const route = useRoute()

watch(
  () => route.path,
  (path) => {
    maybeOpenDailyTaskPrompt(path)
  },
  { immediate: true },
)
</script>

<template>
  <RouterView v-slot="{ Component, route }">
    <transition name="page-slide" mode="out-in">
      <component :is="Component" :key="route.path" />
    </transition>
  </RouterView>
  
  <transition name="page-slide">
    <IntroducePage v-if="showIntroduceModal" overlay />
  </transition>

  <ComposeModal v-if="showComposeModal" />
  <DailyTaskPrompt v-if="showDailyTaskPrompt" />
</template>

