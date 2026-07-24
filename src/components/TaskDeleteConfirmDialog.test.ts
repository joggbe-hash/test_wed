import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import TaskDeleteConfirmDialog from './TaskDeleteConfirmDialog.vue'

describe('TaskDeleteConfirmDialog', () => {
  it('describes the target and emits confirm from the user action', async () => {
    const wrapper = mount(TaskDeleteConfirmDialog, {
      props: {
        item: { id: 9, title: '完成測試' },
      },
      global: {
        stubs: { Teleport: true },
      },
    })

    expect(wrapper.get('[role="alertdialog"]').text()).toContain('完成測試')
    await wrapper.get('.task-delete-confirm').trigger('click')
    expect(wrapper.emitted('confirm')).toHaveLength(1)
  })
})
