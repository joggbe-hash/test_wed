<script setup>
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
        <span class="material-symbols-outlined">explore</span>
      </div>
      <div
        class="nav-icon"
        :style="props.active === 'personal' ? { backgroundColor: 'white' } : null"
        @click="router.push('/personal')"
      ></div>
      <div class="nav-badge" @click="router.push('/freq')">999+</div>
    </div>
  </div>
</template>
