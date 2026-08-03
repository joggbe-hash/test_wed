import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DailyTaskListView from './DailyTaskListView.vue'

describe('DailyTaskListView', () => {
  it('renders tasks and emits public actions', async () => {
    const wrapper = mount(DailyTaskListView, {
      props: {
        tasks: [{
          id: 1,
          title: '完成報告',
          note: '第三章',
          date: '2026-08-04',
          time: '08:00',
          priority: 'high',
          completed: false,
          order: 0,
        }],
      },
    })

    expect(wrapper.text()).toContain('今天有1項任務')
    expect(wrapper.text()).toContain('完成報告')
    await wrapper.get('[data-daily-list-add]').trigger('click')
    await wrapper.get('.daily-task-save-list').trigger('click')
    expect(wrapper.emitted('add')).toHaveLength(1)
    expect(wrapper.emitted('save')).toHaveLength(1)
  })
})
