import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import InspirationDateFilter from './InspirationDateFilter.vue'

const requiredProps = {
  open: true,
  monthOptions: [1, 2],
  startDayOptions: [1, 2],
  endDayOptions: [1, 2],
  yearOptions: [2025, 2026],
  startMonth: 1,
  startDay: 1,
  startYear: 2025,
  endMonth: 2,
  endDay: 2,
  endYear: 2026,
  sortOrder: 'newest' as const,
}

describe('InspirationDateFilter', () => {
  it('emits filter actions through its public interface', async () => {
    const wrapper = mount(InspirationDateFilter, { props: requiredProps })

    await wrapper.get('.inspiration-filter-reset').trigger('click')
    await wrapper.get('.inspiration-apply-filter').trigger('click')
    await wrapper.get('.inspiration-cancel-filter').trigger('click')

    expect(wrapper.emitted('reset')).toHaveLength(1)
    expect(wrapper.emitted('apply')).toHaveLength(1)
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })

  it('emits typed model updates when the user changes a date', async () => {
    const wrapper = mount(InspirationDateFilter, { props: requiredProps })
    await wrapper.get('select[aria-label="開始月份"]').setValue('2')
    expect(wrapper.emitted('update:startMonth')?.[0]).toEqual([2])
  })

  it('blocks applying when the end date is earlier than the start date', async () => {
    const wrapper = mount(InspirationDateFilter, {
      props: {
        ...requiredProps,
        startMonth: 8,
        startDay: 8,
        startYear: 2026,
        endMonth: 8,
        endDay: 1,
        endYear: 2026,
      },
    })

    expect(wrapper.get('[role="alert"]').text()).toBe('結束日期不得早於開始日期')
    expect(wrapper.get('select[aria-label="結束日期"]').attributes('aria-invalid')).toBe('true')
    expect(wrapper.get('.inspiration-apply-filter').attributes('disabled')).toBeDefined()

    await wrapper.get('.inspiration-apply-filter').trigger('click')
    expect(wrapper.emitted('apply')).toBeUndefined()

    await wrapper.setProps({ endDay: 8 })

    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.get('select[aria-label="結束日期"]').attributes('aria-invalid')).toBe('false')
    expect(wrapper.get('.inspiration-apply-filter').attributes('disabled')).toBeUndefined()

    await wrapper.get('.inspiration-apply-filter').trigger('click')
    expect(wrapper.emitted('apply')).toHaveLength(1)
  })
})
