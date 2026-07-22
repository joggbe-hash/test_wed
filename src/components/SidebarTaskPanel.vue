<script setup lang="ts">
import type { CSSProperties } from 'vue'
import type { TaskItem } from '../composables/useScheduleMock'

defineProps<{
  tasks: TaskItem[]
  expanded: boolean
  openMenuId: number | null
  menuStyle: CSSProperties
  activeNote: string
  activeNoteStyle: CSSProperties
}>()

const emit = defineEmits<{
  toggleExpansion: []
  add: []
  toggleTask: [id: number]
  showNote: [task: TaskItem, event: MouseEvent | FocusEvent]
  hideNote: [id: number]
  focusOut: [event: FocusEvent, id: number]
  toggleMenu: [id: number, event: MouseEvent]
  closeMenu: []
  edit: [task: TaskItem]
  remove: [task: TaskItem]
}>()
</script>

<template>
  <section class="sidebar-panel task-panel">
    <button id="sidebar-task-heading" type="button" class="sidebar-panel-heading" :aria-expanded="expanded" aria-controls="sidebar-task-list" :aria-label="expanded ? '收合今日任務清單' : '展開今日任務清單'" @click="emit('toggleExpansion')">今日任務</button>
    <div id="sidebar-task-list" class="task-list" :class="{ 'task-list-expanded': expanded }">
      <div v-for="task in tasks" :key="task.id" class="task-row" @mouseenter="emit('showNote', task, $event)" @mouseleave="emit('hideNote', task.id)" @focusin="emit('showNote', task, $event)" @focusout="emit('focusOut', $event, task.id)">
        <input :id="`sidebar-task-${task.id}`" type="checkbox" :checked="task.completed" :aria-label="`${task.title} 完成狀態`" @change="emit('toggleTask', task.id)">
        <label class="task-row-title" :for="`sidebar-task-${task.id}`" :title="task.title">{{ task.title }}</label>
        <div class="task-row-actions" @click.stop>
          <button :id="`sidebar-task-menu-trigger-${task.id}`" type="button" class="task-menu-trigger" :aria-label="`${task.title} 更多操作`" aria-haspopup="menu" :aria-expanded="openMenuId === task.id" :aria-controls="`sidebar-task-menu-${task.id}`" @click="emit('toggleMenu', task.id, $event)" @keydown.escape.stop.prevent="emit('closeMenu')"><span aria-hidden="true">⋮</span></button>
          <div v-if="openMenuId === task.id" :id="`sidebar-task-menu-${task.id}`" class="task-menu-panel" :style="menuStyle" role="menu" @keydown.escape.stop.prevent="emit('closeMenu')">
            <button type="button" role="menuitem" class="task-menu-edit" @click="emit('edit', task)">編輯</button>
            <button type="button" role="menuitem" class="task-menu-delete" @click="emit('remove', task)">刪除</button>
          </div>
        </div>
      </div>
    </div>
    <div v-if="activeNote" class="task-note-preview" :style="activeNoteStyle" role="tooltip" aria-live="polite">{{ activeNote }}</div>
    <button type="button" class="add-task-btn" @click="emit('add')">+新增任務</button>
  </section>
</template>
