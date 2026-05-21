<script setup lang="ts">
import { useRouter } from 'vue-router'

const props = defineProps({
  active: {
    type: String,
    default: '',
  },
})

const router = useRouter()
</script>

<template>
  <div class="navbar">
    <div class="nav-logo" @click="router.push('/home')"></div>
    <div class="nav-actions">
      <div class="nav-square" @click="router.push('/explore')">
        <svg
          class="size-6"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="10"></circle>
          <polygon points="16.24 7.76 14.12 14.12 7.76 16.24 9.88 9.88 16.24 7.76"></polygon>
        </svg>
      </div>
      <div
        :class="['nav-icon', props.active === 'personal' ? 'bg-white' : '']"
        @click="router.push('/personal')"
      ></div>
      <div class="nav-badge" @click="router.push('/freq')">999+</div>
    </div>
  </div>
</template>
