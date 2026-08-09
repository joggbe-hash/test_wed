import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it } from 'vitest'
import DailyTaskPrompt from './DailyTaskPrompt.vue'
import TimeSelect from './TimeSelect.vue'

describe('DailyTaskPrompt', () => {
  it('uses the rounded time picker for both daily task times', () => {
    const wrapper = mount(DailyTaskPrompt, {
      global: {
        stubs: {
          Teleport: true,
          DailyTaskListView: true,
        },
      },
    })

    expect(wrapper.findAll('input[type="time"]')).toHaveLength(0)
    expect(wrapper.findAllComponents(TimeSelect)).toHaveLength(2)
  })

  it('blocks submission and explains when the end time is not later than the start time', async () => {
    const wrapper = mount(DailyTaskPrompt, {
      global: {
        stubs: {
          Teleport: true,
          DailyTaskListView: true,
        },
      },
    })

    await wrapper.find('input[type="text"]').setValue('測試任務')
    const [startTime] = wrapper.findAllComponents(TimeSelect)

    startTime.vm.$emit('update:modelValue', '06:02')
    await nextTick()
    const endTime = wrapper.findAllComponents(TimeSelect)[1]
    endTime.vm.$emit('update:modelValue', '05:02')
    await nextTick()

    expect(wrapper.get('[role="alert"]').text()).toBe('結束時間必須晚於開始時間')
    expect(wrapper.get('.daily-task-time-panel').find('[role="alert"]').exists()).toBe(true)
    expect(wrapper.get('.daily-task-confirm').attributes('disabled')).toBeDefined()
    expect(wrapper.findAllComponents(TimeSelect)[1].get('button[aria-haspopup="dialog"]').attributes('aria-invalid')).toBe('true')
  })

  it('clears the time error and enables submission after the range becomes valid', async () => {
    const wrapper = mount(DailyTaskPrompt, {
      global: {
        stubs: {
          Teleport: true,
          DailyTaskListView: true,
        },
      },
    })

    await wrapper.find('input[type="text"]').setValue('測試任務')
    const [startTime] = wrapper.findAllComponents(TimeSelect)

    startTime.vm.$emit('update:modelValue', '06:02')
    await nextTick()
    let endTime = wrapper.findAllComponents(TimeSelect)[1]
    endTime.vm.$emit('update:modelValue', '05:02')
    await nextTick()
    endTime = wrapper.findAllComponents(TimeSelect)[1]
    endTime.vm.$emit('update:modelValue', '07:02')
    await nextTick()

    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.get('.daily-task-confirm').attributes('disabled')).toBeUndefined()
    expect(wrapper.findAllComponents(TimeSelect)[1].get('button[aria-haspopup="dialog"]').attributes('aria-invalid')).toBeUndefined()
  })
})
