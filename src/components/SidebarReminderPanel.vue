<script setup lang="ts">
import type { CSSProperties } from 'vue'
import type { ReminderItem } from '../composables/useScheduleMock'

defineProps<{
  reminders: ReminderItem[]
  expanded: boolean
  openMenuId: number | null
  menuStyle: CSSProperties
  activeNote: string
  activeNoteStyle: CSSProperties
}>()

const emit = defineEmits<{
  toggleExpansion: []
  add: []
  showNote: [reminder: ReminderItem, event: MouseEvent | FocusEvent]
  hideNote: [id: number]
  focusOut: [event: FocusEvent, id: number]
  toggleMenu: [id: number, event: MouseEvent]
  closeMenu: []
  edit: [reminder: ReminderItem]
  remove: [reminder: ReminderItem]
}>()

function timeRange(reminder: ReminderItem) {
  return reminder.endTime ? `${reminder.time} - ${reminder.endTime}` : reminder.time
}
</script>

<template>
  <section class="sidebar-panel reminder-panel">
    <button id="sidebar-reminder-heading" type="button" class="sidebar-panel-heading" :aria-expanded="expanded" aria-controls="sidebar-reminder-list" :aria-label="expanded ? '收合今日提醒清單' : '展開今日提醒清單'" @click="emit('toggleExpansion')">今日提醒</button>
    <div id="sidebar-reminder-list" class="task-list sidebar-reminder-list" :class="{ 'task-list-expanded': expanded }">
      <div
        v-for="reminder in reminders"
        :key="reminder.id"
        class="task-row sidebar-reminder-row"
        :tabindex="reminder.note.trim() ? 0 : undefined"
        @mouseenter="emit('showNote', reminder, $event)"
        @mouseleave="emit('hideNote', reminder.id)"
        @focusin="emit('showNote', reminder, $event)"
        @focusout="emit('focusOut', $event, reminder.id)"
      >
        <span class="sidebar-reminder-time" :title="timeRange(reminder)">{{ timeRange(reminder) }}</span>
        <span class="sidebar-reminder-title" :title="reminder.title">{{ reminder.title }}</span>
        <div class="task-row-actions" @click.stop>
          <button :id="`sidebar-reminder-menu-trigger-${reminder.id}`" type="button" class="task-menu-trigger" :aria-label="`${reminder.title} 更多操作`" aria-haspopup="menu" :aria-expanded="openMenuId === reminder.id" :aria-controls="`sidebar-reminder-menu-${reminder.id}`" @click="emit('toggleMenu', reminder.id, $event)" @keydown.escape.stop.prevent="emit('closeMenu')"><span aria-hidden="true">⋮</span></button>
          <div v-if="openMenuId === reminder.id" :id="`sidebar-reminder-menu-${reminder.id}`" class="task-menu-panel" :style="menuStyle" role="menu" @keydown.escape.stop.prevent="emit('closeMenu')">
            <button type="button" role="menuitem" class="task-menu-edit" @click="emit('edit', reminder)">編輯</button>
            <button type="button" role="menuitem" class="task-menu-delete" @click="emit('remove', reminder)">刪除</button>
          </div>
        </div>
      </div>
    </div>
    <div v-if="activeNote" class="task-note-preview" :style="activeNoteStyle" role="tooltip" aria-live="polite">{{ activeNote }}</div>
    <button type="button" class="add-task-btn" @click="emit('add')">+新增提醒</button>
  </section>
</template>
