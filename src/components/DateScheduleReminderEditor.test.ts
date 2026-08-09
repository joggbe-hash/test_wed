import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DateScheduleReminderEditor from './DateScheduleReminderEditor.vue'
import TimeSelect from './TimeSelect.vue'

describe('DateScheduleReminderEditor', () => {
  it('uses the custom black-and-white picker for both reminder times', () => {
    const wrapper = mount(DateScheduleReminderEditor, {
      props: {
        editorTitle: '新增提醒',
        day: 8,
        dateLabel: '8月8日 週六',
        isNew: true,
        inactive: false,
        title: '',
        note: '',
        date: '2026-08-08',
        startTime: '08:00',
        endTime: '09:00',
      },
    })

    expect(wrapper.findAll('input[type="time"]')).toHaveLength(0)
    expect(wrapper.findAllComponents(TimeSelect)).toHaveLength(2)
    expect(wrapper.findAllComponents(TimeSelect).every(component => component.props('variant') === 'editor-card')).toBe(true)
    expect(wrapper.findAllComponents(TimeSelect).every(component => component.props('subtitle') === '8月8日 週六')).toBe(true)
  })
})
