import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import AccountSessionPanel from './AccountSessionPanel.vue'

describe('AccountSessionPanel', () => {
  it('offers only the current-device logout action', async () => {
    const wrapper = mount(AccountSessionPanel, {
      props: {
        isLoggingOut: false,
        errorMessage: '',
      },
    })

    await wrapper.get('[data-logout-current]').trigger('click')

    expect(wrapper.emitted('logout')).toHaveLength(1)
    expect(wrapper.find('[data-logout-all]').exists()).toBe(false)
  })

  it('disables logout while the current session is being revoked', () => {
    const wrapper = mount(AccountSessionPanel, {
      props: {
        isLoggingOut: true,
        errorMessage: '',
      },
    })

    expect(wrapper.get('[data-logout-current]').attributes('disabled')).toBeDefined()
  })
})
