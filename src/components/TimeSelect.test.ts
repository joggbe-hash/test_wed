import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import TimeSelect from './TimeSelect.vue'

describe('TimeSelect', () => {
  it('opens a rounded custom picker instead of the native time popup', async () => {
    const wrapper = mount(TimeSelect, {
      props: {
        label: '時間',
        modelValue: '09:05',
      },
    })

    await wrapper.get('button[aria-haspopup="dialog"]').trigger('click')

    expect(wrapper.get('[role="dialog"]').classes()).toContain('time-picker-panel')
    expect(wrapper.find('input[type="time"]').exists()).toBe(false)
  })

  it('converts afternoon selections back to the 24-hour model value', async () => {
    const wrapper = mount(TimeSelect, {
      props: {
        label: '時間',
        modelValue: '09:05',
      },
    })

    await wrapper.get('button[aria-haspopup="dialog"]').trigger('click')
    await wrapper.get('button[data-period="pm"]').trigger('click')
    await wrapper.get('button[data-hour="2"]').trigger('click')
    await wrapper.get('button[data-minute="30"]').trigger('click')

    const updates = wrapper.emitted('update:modelValue') ?? []
    expect(updates[updates.length - 1]).toEqual(['14:30'])
  })

  it('closes the picker with Escape and returns focus to the trigger', async () => {
    const wrapper = mount(TimeSelect, {
      attachTo: document.body,
      props: {
        label: '時間',
        modelValue: '09:05',
      },
    })

    const trigger = wrapper.get<HTMLButtonElement>('button[aria-haspopup="dialog"]')
    await trigger.trigger('click')
    await wrapper.get('[role="dialog"]').trigger('keydown', { key: 'Escape' })

    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(document.activeElement).toBe(trigger.element)
    wrapper.unmount()
  })
})
