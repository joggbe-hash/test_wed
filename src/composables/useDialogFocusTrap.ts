const focusableSelector = [
  'button:not([disabled])',
  '[href]',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(', ')

export function isUsableFocusTarget(element: HTMLElement | null): element is HTMLElement {
  return Boolean(element && element.isConnected && element !== document.body)
}

export function focusWithoutScroll(element: HTMLElement | null) {
  if (!isUsableFocusTarget(element)) return
  element.focus({ preventScroll: true })
}

export function trapDialogFocus(event: KeyboardEvent, panel: HTMLElement | null) {
  const focusableElements = panel
    ? Array.from(panel.querySelectorAll<HTMLElement>(focusableSelector))
      .filter((element) => element.getClientRects().length > 0)
    : []

  if (!panel || focusableElements.length === 0) {
    event.preventDefault()
    focusWithoutScroll(panel)
    return
  }

  const firstElement = focusableElements[0]
  const lastElement = focusableElements[focusableElements.length - 1]
  const activeElement = document.activeElement

  if (!(activeElement instanceof Node) || !panel.contains(activeElement)) {
    event.preventDefault()
    focusWithoutScroll(event.shiftKey ? lastElement : firstElement)
    return
  }

  if (event.shiftKey && activeElement === firstElement) {
    event.preventDefault()
    focusWithoutScroll(lastElement)
  } else if (!event.shiftKey && activeElement === lastElement) {
    event.preventDefault()
    focusWithoutScroll(firstElement)
  }
}
