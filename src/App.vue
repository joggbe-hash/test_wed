<script setup lang="ts">
import { RouterView } from 'vue-router'
import ComposeModal from './components/ComposeModal.vue'
import IntroducePage from './views/IntroducePage.vue'
import { showComposeModal } from './composables/useComposeModal'
import { showIntroduceModal } from './composables/useModal'
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
</template>

