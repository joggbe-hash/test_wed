import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const api = vi.hoisted(() => ({
  fetchInspirations: vi.fn(),
  createInspiration: vi.fn(),
  updateInspiration: vi.fn(),
  deleteInspiration: vi.fn(),
}))

vi.mock('../api/backendApi', () => api)
vi.mock('../composables/useSession', () => ({
  useSession: () => ({ currentUser: { value: { id: 1 } } }),
}))
vi.mock('../composables/useAccessibleDialog', () => ({
  useAccessibleDialog: vi.fn(),
}))

import InspirationListModal from './InspirationListModal.vue'

describe('InspirationListModal', () => {
  beforeEach(() => {
    api.fetchInspirations.mockReset()
  })

  it('includes older items in the initial date range after the API load completes', async () => {
    api.fetchInspirations.mockResolvedValue({
      items: [{ id: 1, date: '2000-01-01', text: '較早的靈感' }],
    })

    const wrapper = mount(InspirationListModal, {
      global: {
        stubs: {
          Teleport: true,
          DailyTaskPrompt: true,
          InspirationDateFilter: true,
        },
      },
    })
    await flushPromises()

    expect(api.fetchInspirations).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('較早的靈感')
  })
})
