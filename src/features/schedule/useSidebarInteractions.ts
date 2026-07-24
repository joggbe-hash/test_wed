import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  shallowRef,
  type Ref,
} from 'vue'
import type { ReminderItem, TaskItem } from './types'

export function useSidebarInteractions(
  tasks: Readonly<Ref<readonly TaskItem[]>>,
  reminders: Readonly<Ref<readonly ReminderItem[]>>,
) {
  const isReminderListExpanded = shallowRef(false)
  const isTaskListExpanded = shallowRef(false)
  const openTaskMenuId = shallowRef<number | null>(null)
  const openReminderMenuId = shallowRef<number | null>(null)
  const taskMenuTop = shallowRef(0)
  const taskMenuLeft = shallowRef(0)
  const activeReminderNoteId = shallowRef<number | null>(null)
  const activeReminderNoteTop = shallowRef(0)
  const activeReminderNoteLeft = shallowRef(0)
  const activeTaskNoteId = shallowRef<number | null>(null)
  const activeTaskNoteTop = shallowRef(0)
  const activeTaskNoteLeft = shallowRef(0)

  const reminderPanelHeadingId = 'sidebar-reminder-heading'
  const taskPanelHeadingId = 'sidebar-task-heading'
  const activeReminderNote = computed(() => {
    if (openReminderMenuId.value !== null) return ''
    return reminders.value
      .find((item) => item.id === activeReminderNoteId.value)
      ?.note?.trim() ?? ''
  })
  const activeTaskNote = computed(() => {
    if (openTaskMenuId.value !== null) return ''
    return tasks.value
      .find((item) => item.id === activeTaskNoteId.value)
      ?.note?.trim() ?? ''
  })
  const activeReminderNoteStyle = computed(() => ({
    top: `${activeReminderNoteTop.value}px`,
    left: `${activeReminderNoteLeft.value}px`,
  }))
  const activeTaskNoteStyle = computed(() => ({
    top: `${activeTaskNoteTop.value}px`,
    left: `${activeTaskNoteLeft.value}px`,
  }))
  const taskMenuStyle = computed(() => ({
    top: `${taskMenuTop.value}px`,
    left: `${taskMenuLeft.value}px`,
  }))

  function taskMenuTriggerId(taskId: number) {
    return `sidebar-task-menu-trigger-${taskId}`
  }

  function reminderMenuTriggerId(reminderId: number) {
    return `sidebar-reminder-menu-trigger-${reminderId}`
  }

  function previewPosition(row: HTMLElement, text: string) {
    const rect = row.getBoundingClientRect()
    const tooltipWidth = Math.min(260, Math.max(96, text.length * 8 + 28))
    const rightLeft = rect.right + 10
    const leftLeft = rect.left - tooltipWidth - 10
    const hasRightSpace = rightLeft + tooltipWidth <= window.innerWidth - 8

    return {
      top: Math.max(12, Math.min(rect.top + rect.height / 2, window.innerHeight - 12)),
      left: hasRightSpace ? rightLeft : Math.max(8, leftLeft),
    }
  }

  function showReminderNote(reminder: ReminderItem, event: MouseEvent | FocusEvent) {
    const note = reminder.note?.trim()
    if (!note) {
      activeReminderNoteId.value = null
      return
    }
    const row = event.currentTarget
    if (row instanceof HTMLElement) {
      const position = previewPosition(row, note)
      activeReminderNoteTop.value = position.top
      activeReminderNoteLeft.value = position.left
    }
    activeTaskNoteId.value = null
    activeReminderNoteId.value = reminder.id
  }

  function hideReminderNote(reminderId: number) {
    if (activeReminderNoteId.value === reminderId) {
      activeReminderNoteId.value = null
    }
  }

  function handleReminderFocusOut(event: FocusEvent, reminderId: number) {
    const nextTarget = event.relatedTarget
    const currentTarget = event.currentTarget
    if (
      nextTarget instanceof Node
      && currentTarget instanceof HTMLElement
      && currentTarget.contains(nextTarget)
    ) {
      return
    }
    hideReminderNote(reminderId)
  }

  function showTaskNote(task: TaskItem, event: MouseEvent | FocusEvent) {
    const note = task.note?.trim()
    if (!note) {
      activeTaskNoteId.value = null
      return
    }
    const row = event.currentTarget
    if (row instanceof HTMLElement) {
      const position = previewPosition(row, note)
      activeTaskNoteTop.value = position.top
      activeTaskNoteLeft.value = position.left
    }
    activeTaskNoteId.value = task.id
    activeReminderNoteId.value = null
  }

  function hideTaskNote(taskId: number) {
    if (activeTaskNoteId.value === taskId) {
      activeTaskNoteId.value = null
    }
  }

  function handleTaskFocusOut(event: FocusEvent, taskId: number) {
    const nextTarget = event.relatedTarget
    const currentTarget = event.currentTarget
    if (
      nextTarget instanceof Node
      && currentTarget instanceof HTMLElement
      && currentTarget.contains(nextTarget)
    ) {
      return
    }
    hideTaskNote(taskId)
  }

  function closeTaskMenu() {
    openTaskMenuId.value = null
    openReminderMenuId.value = null
  }

  function closeTaskMenuAndRestoreFocus() {
    const triggerId = openTaskMenuId.value !== null
      ? taskMenuTriggerId(openTaskMenuId.value)
      : openReminderMenuId.value !== null
        ? reminderMenuTriggerId(openReminderMenuId.value)
        : null

    closeTaskMenu()
    if (triggerId) {
      void nextTick(() => document.getElementById(triggerId)?.focus())
    }
  }

  function updateTaskMenuPosition(trigger: HTMLElement) {
    const rect = trigger.getBoundingClientRect()
    const menuWidth = 96
    const menuHeight = 76
    const gap = 6
    const viewportPadding = 8
    const containerRect = trigger.closest<HTMLElement>('.sidebar-panel')?.getBoundingClientRect()
    const boundaryLeft = Math.max(viewportPadding, (containerRect?.left ?? viewportPadding) + 4)
    const boundaryRight = Math.min(
      window.innerWidth - viewportPadding,
      (containerRect?.right ?? window.innerWidth - viewportPadding) - 4,
    )
    const preferredLeft = rect.right + gap
    const fallbackLeft = rect.left - gap - menuWidth
    const hasRightSpace = preferredLeft + menuWidth <= boundaryRight
    const hasLeftSpace = fallbackLeft >= boundaryLeft

    taskMenuTop.value = Math.max(
      viewportPadding,
      Math.min(
        rect.top + rect.height / 2 - menuHeight / 2,
        window.innerHeight - viewportPadding - menuHeight,
      ),
    )
    taskMenuLeft.value = hasRightSpace
      ? preferredLeft
      : hasLeftSpace
        ? fallbackLeft
        : Math.max(boundaryLeft, boundaryRight - menuWidth)
  }

  function toggleTaskMenu(taskId: number, event: MouseEvent) {
    if (openTaskMenuId.value === taskId) {
      closeTaskMenu()
      return
    }
    if (event.currentTarget instanceof HTMLElement) {
      updateTaskMenuPosition(event.currentTarget)
    }
    activeTaskNoteId.value = null
    openReminderMenuId.value = null
    openTaskMenuId.value = taskId
  }

  function toggleReminderMenu(reminderId: number, event: MouseEvent) {
    if (openReminderMenuId.value === reminderId) {
      closeTaskMenu()
      return
    }
    if (event.currentTarget instanceof HTMLElement) {
      updateTaskMenuPosition(event.currentTarget)
    }
    activeReminderNoteId.value = null
    openTaskMenuId.value = null
    openReminderMenuId.value = reminderId
  }

  function toggleTaskListExpansion() {
    isTaskListExpanded.value = !isTaskListExpanded.value
    closeTaskMenu()
    activeTaskNoteId.value = null
    activeReminderNoteId.value = null
  }

  function toggleReminderListExpansion() {
    isReminderListExpanded.value = !isReminderListExpanded.value
    closeTaskMenu()
    activeTaskNoteId.value = null
    activeReminderNoteId.value = null
  }

  onMounted(() => {
    document.addEventListener('click', closeTaskMenu)
    document.addEventListener('scroll', closeTaskMenu, true)
    window.addEventListener('resize', closeTaskMenu)
  })

  onBeforeUnmount(() => {
    document.removeEventListener('click', closeTaskMenu)
    document.removeEventListener('scroll', closeTaskMenu, true)
    window.removeEventListener('resize', closeTaskMenu)
  })

  return {
    isReminderListExpanded,
    isTaskListExpanded,
    openTaskMenuId,
    openReminderMenuId,
    activeReminderNoteId,
    activeTaskNoteId,
    reminderPanelHeadingId,
    taskPanelHeadingId,
    activeReminderNote,
    activeTaskNote,
    activeReminderNoteStyle,
    activeTaskNoteStyle,
    taskMenuStyle,
    taskMenuTriggerId,
    reminderMenuTriggerId,
    showReminderNote,
    hideReminderNote,
    handleReminderFocusOut,
    showTaskNote,
    hideTaskNote,
    handleTaskFocusOut,
    closeTaskMenu,
    closeTaskMenuAndRestoreFocus,
    toggleTaskMenu,
    toggleReminderMenu,
    toggleTaskListExpansion,
    toggleReminderListExpansion,
  }
}
