import { nextTick, onBeforeUnmount, onMounted, type ShallowRef } from 'vue'

const focusableSelector = [
  'button:not([disabled])',
  '[href]',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

interface AccessibleDialogOptions {
  dialog: Readonly<ShallowRef<HTMLElement | null>>
  initialFocus?: Readonly<ShallowRef<HTMLElement | null>>
  onClose: () => void
  backgroundSelector?: string
  closeOnEscape?: boolean
}

function visibleFocusTargets(dialog: HTMLElement) {
  return Array.from(dialog.querySelectorAll<HTMLElement>(focusableSelector))
    .filter((element) => element.getClientRects().length > 0)
}

export function useAccessibleDialog(options: AccessibleDialogOptions) {
  let returnFocus: HTMLElement | null = null
  let previousBodyOverflow = ''
  let background: HTMLElement | null = null
  let backgroundWasInert = false
  let backgroundAriaHidden: string | null = null
  let ownsBackgroundInert = false
  let ownsBackgroundAriaHidden = false

  function focusInitialControl() {
    void nextTick(() => {
      const dialog = options.dialog.value
      if (!dialog) return
      const target = options.initialFocus?.value ?? visibleFocusTargets(dialog)[0] ?? dialog
      target.focus({ preventScroll: true })
    })
  }

  function handleKeydown(event: KeyboardEvent) {
    const dialog = options.dialog.value
    if (!dialog) return

    if (event.key === 'Escape' && options.closeOnEscape !== false) {
      event.preventDefault()
      options.onClose()
      return
    }
    if (event.key !== 'Tab') return

    const targets = visibleFocusTargets(dialog)
    if (targets.length === 0) {
      event.preventDefault()
      dialog.focus({ preventScroll: true })
      return
    }

    const first = targets[0]
    const last = targets[targets.length - 1]
    const active = document.activeElement
    if (!dialog.contains(active)) {
      event.preventDefault()
      ;(event.shiftKey ? last : first).focus({ preventScroll: true })
    } else if (event.shiftKey && active === first) {
      event.preventDefault()
      last.focus({ preventScroll: true })
    } else if (!event.shiftKey && active === last) {
      event.preventDefault()
      first.focus({ preventScroll: true })
    }
  }

  onMounted(() => {
    returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    previousBodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'

    if (options.backgroundSelector) {
      background = document.querySelector<HTMLElement>(options.backgroundSelector)
      if (background) {
        backgroundWasInert = background.inert
        backgroundAriaHidden = background.getAttribute('aria-hidden')
        ownsBackgroundInert = !backgroundWasInert
        ownsBackgroundAriaHidden = backgroundAriaHidden !== 'true'

        if (ownsBackgroundInert) background.inert = true
        if (ownsBackgroundAriaHidden) background.setAttribute('aria-hidden', 'true')
      }
    }

    window.addEventListener('keydown', handleKeydown)
    focusInitialControl()
  })

  onBeforeUnmount(() => {
    window.removeEventListener('keydown', handleKeydown)
    document.body.style.overflow = previousBodyOverflow
    if (background) {
      if (ownsBackgroundInert) background.inert = backgroundWasInert
      if (ownsBackgroundAriaHidden) {
        if (backgroundAriaHidden === null) background.removeAttribute('aria-hidden')
        else background.setAttribute('aria-hidden', backgroundAriaHidden)
      }
    }
    if (returnFocus?.isConnected) {
      returnFocus.focus({ preventScroll: true })
    }
  })

  return { focusInitialControl }
}
