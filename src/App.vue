<template>
  <RouterView v-slot="{ Component, route }">
    <transition name="page-slide" mode="out-in">
      <component :is="Component" :key="route.path" />
    </transition>
  </RouterView>
</template>

<style>
/* 現代化的高級轉場動畫 (Modern Page Transition) */
.page-slide-enter-active,
.page-slide-leave-active {
  transition: opacity 0.35s cubic-bezier(0.25, 1, 0.5, 1), 
              transform 0.35s cubic-bezier(0.25, 1, 0.5, 1),
              filter 0.35s cubic-bezier(0.25, 1, 0.5, 1);
}

.page-slide-enter-from {
  opacity: 0;
  transform: translateY(10px) scale(0.98);
  filter: blur(4px);
}

.page-slide-leave-to {
  opacity: 0;
  transform: translateY(-10px) scale(0.98);
  filter: blur(4px);
}
</style>
