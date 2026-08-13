import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import LoginOwnershipStep from './LoginOwnershipStep.vue'

describe('LoginOwnershipStep', () => {
  it('normalizes the 16-character ownership code and exposes OTP autocomplete', async () => {
    const wrapper = mount(LoginOwnershipStep, {
      props: {
        modelValue: '',
        email: 'user@example.com',
        isSubmitting: false,
        statusMessage: '',
        errorMessage: '',
      },
    })

    const input = wrapper.get('#login-ownership-code')
    await input.setValue('abcdefghjklmnpq2')

    const modelUpdates = wrapper.emitted('update:modelValue') ?? []
    expect(modelUpdates[modelUpdates.length - 1]).toEqual(['ABCDEFGHJKLMNPQ2'])
    expect(input.attributes('autocomplete')).toBe('one-time-code')
    expect(input.attributes('pattern')).toBe('[A-HJ-NP-Z2-9]{16}')
    expect(input.attributes('maxlength')).toBe('16')
  })

  it('keeps cancellation available while a request is in flight', async () => {
    const wrapper = mount(LoginOwnershipStep, {
      props: {
        modelValue: 'ABCDEFGHJKLMNPQ2',
        email: 'user@example.com',
        isSubmitting: true,
        statusMessage: '驗證中',
        errorMessage: '',
      },
    })

    const cancel = wrapper.get('[data-testid="cancel-ownership"]')
    expect(cancel.attributes('disabled')).toBeUndefined()
    await cancel.trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeDefined()
  })
})
