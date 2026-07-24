<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  title: string
  description: string
  icon: string
  actionLabel: string
  position: number
}>()

const emit = defineEmits<{
  select: []
}>()

const positionLabel = computed(() => String(props.position).padStart(2, '0'))
</script>

<template>
  <button type="button" class="settings-action-card" @click="emit('select')">
    <span class="settings-action-card__top">
      <span class="settings-action-card__icon" aria-hidden="true">
        <span class="material-symbols-outlined">{{ icon }}</span>
      </span>
      <span class="settings-action-card__number">{{ positionLabel }}</span>
    </span>

    <span class="settings-action-card__content">
      <strong class="settings-action-card__title text-balance">{{ title }}</strong>
      <span class="settings-action-card__description text-pretty">{{ description }}</span>
    </span>

    <span class="settings-action-card__footer">
      <span>{{ actionLabel }}</span>
      <span class="settings-action-card__arrow" aria-hidden="true">↗</span>
    </span>
  </button>
</template>

<style scoped>
@reference "../../style.css";

.settings-action-card {
  @apply flex min-h-64 w-full flex-col rounded-2xl border border-border-soft bg-card p-6 text-left text-brown shadow-sm hover:border-nav hover:bg-surface-warm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-nav max-md:min-h-56 max-md:p-5;
  font: inherit;
}

.settings-action-card__top,
.settings-action-card__footer {
  @apply flex w-full items-center justify-between;
}

.settings-action-card__icon {
  @apply flex size-12 items-center justify-center rounded-full bg-content text-brown;
}

.settings-action-card__icon .material-symbols-outlined {
  @apply text-[26px];
}

.settings-action-card__number {
  @apply text-sm font-semibold text-[#625a54] tabular-nums;
}

.settings-action-card__content {
  @apply mt-8 flex flex-1 flex-col gap-3;
}

.settings-action-card__title {
  @apply text-2xl font-bold leading-tight text-ink-warm;
}

.settings-action-card__description {
  @apply max-w-[34ch] text-sm leading-6 text-[#625a54];
}

.settings-action-card__footer {
  @apply mt-8 border-t border-border-soft pt-4 text-sm font-semibold;
}

.settings-action-card__arrow {
  @apply text-xl leading-none;
}
</style>
