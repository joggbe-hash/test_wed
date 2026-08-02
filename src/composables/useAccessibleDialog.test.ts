import { defineComponent, h, nextTick, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { useAccessibleDialog } from './useAccessibleDialog'

function mountDialog() {
  return mount(defineComponent({
    setup() {
      const dialog = ref<HTMLElement | null>(null)

      useAccessibleDialog({
        dialog,
        onClose: () => undefined,
        backgroundSelector: '[data-app-route-content]',
      })

      return () => h('section', { ref: dialog, role: 'dialog', tabindex: -1 })
    },
  }), {
    attachTo: document.body,
  })
}

function createBackground() {
  const background = document.createElement('main')
  background.dataset.appRouteContent = ''
  document.body.append(background)
  return background
}

describe('useAccessibleDialog', () => {
  afterEach(() => {
    document.body.replaceChildren()
    document.body.style.overflow = ''
  })

  it('releases background state that it added itself', async () => {
    const background = createBackground()
    const wrapper = mountDialog()
    await nextTick()

    expect(background.inert).toBe(true)
    expect(background.getAttribute('aria-hidden')).toBe('true')

    wrapper.unmount()

    expect(background.inert).toBe(false)
    expect(background.hasAttribute('aria-hidden')).toBe(false)
  })

  it('does not restore a lock owned and already released by the parent', async () => {
    const background = createBackground()
    background.inert = true
    background.setAttribute('aria-hidden', 'true')
    const wrapper = mountDialog()
    await nextTick()

    background.inert = false
    background.removeAttribute('aria-hidden')
    wrapper.unmount()

    expect(background.inert).toBe(false)
    expect(background.hasAttribute('aria-hidden')).toBe(false)
  })
})
