import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import QuickCreateInput from '@/components/QuickCreateInput.vue'

const toastMock = { warning: vi.fn(), error: vi.fn(), success: vi.fn(), info: vi.fn() }
const createIssueMock = vi.fn()

vi.mock('@/composables/useI18n', () => ({
  useI18n: () => ({ t: (k: string) => k }),
}))
vi.mock('@/composables/useToast', () => ({
  useToast: () => toastMock,
}))
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { slug: 'acme' }, query: { view: 'list' } }),
}))

vi.mock('@/api/work-item-template', () => ({
  listWorkItemTemplates: vi.fn(() => Promise.resolve([])),
}))

const checkDuplicatesMock = vi.fn(() => Promise.resolve({ duplicates: [] }))
vi.mock('@/api/issue', () => ({
  default: {
    createIssue: (...args: unknown[]) => createIssueMock(...args),
    checkDuplicates: (...args: unknown[]) => checkDuplicatesMock(...args),
  },
}))

const routerLinkStub = {
  props: ['to'],
  template: '<a data-test="quick-create-full-form-link" :data-payload="JSON.stringify(to)"><slot /></a>',
}

describe('QuickCreateInput', () => {
  beforeEach(() => {
    toastMock.warning.mockClear()
    toastMock.error.mockClear()
    createIssueMock.mockReset()
  })

  const baseProps = {
    projectId: 10,
    workspaceId: 20,
    issueTypes: [
      { id: 1, name: 'Bug', is_default: true, color: '#ef4444' },
      { id: 2, name: 'Feature', is_default: false, color: '#10b981' },
    ],
  }

  function mountQuick(props: Record<string, unknown> = {}) {
    return mount(QuickCreateInput, {
      props: { ...baseProps, ...props },
      global: { stubs: { RouterLink: routerLinkStub } },
    })
  }

  it('shows validation and does not call create when title is empty', async () => {
    createIssueMock.mockResolvedValue({ id: 99 })
    const wrapper = mountQuick()
    await flushPromises()

    await wrapper.find('button').trigger('click')
    expect(toastMock.warning).toHaveBeenCalledWith('quickCreate.titleRequired')
    expect(wrapper.find('[data-test="quick-create-title-error"]').exists()).toBe(true)
    expect(createIssueMock).not.toHaveBeenCalled()
  })

  it('toasts server message and shows full-form link on 4xx', async () => {
    createIssueMock.mockRejectedValue({
      response: { status: 422, data: { message: 'CF required' } },
    })
    const wrapper = mountQuick({ defaultTypeId: 1, defaultStateId: 5 })
    await flushPromises()

    const input = wrapper.find('input[type="text"]')
    await input.setValue('My title')
    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(toastMock.error).toHaveBeenCalledWith('CF required')
    expect(wrapper.find('[data-test="quick-create-api-error"]').text()).toContain('CF required')
    const raw = wrapper.find('[data-test="quick-create-full-form-link"]').attributes('data-payload')
    expect(raw).toBeTruthy()
    const payload = JSON.parse(raw!)
    expect(payload.path).toContain('/issues/new')
    expect(payload.query.name).toBe('My title')
    expect(payload.query.type_id).toBe('1')
    expect(payload.query.state_id).toBe('5')
    expect((wrapper.find('input[type="text"]').element as HTMLInputElement).value).toBe('My title')
  })

  it('sends resolved default type_id when type is “all types”', async () => {
    createIssueMock.mockResolvedValue({ id: 42 })
    const wrapper = mountQuick()
    await flushPromises()

    const input = wrapper.find('input[type="text"]')
    await input.setValue('Hello')
    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(createIssueMock).toHaveBeenCalled()
    const issuePayload = (createIssueMock as Mock).mock.calls[0][2]
    expect(issuePayload.type_id).toBe(1)
  })
})
