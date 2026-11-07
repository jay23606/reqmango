import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

const mockPush = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
}))

const { mockListIssuePages, mockAddIssuePage, mockRemoveIssuePage, mockSearchPages } = vi.hoisted(() => ({
  mockListIssuePages: vi.fn(),
  mockAddIssuePage: vi.fn(),
  mockRemoveIssuePage: vi.fn(),
  mockSearchPages: vi.fn(),
}))

vi.mock('@/api/issue', () => ({
  listIssuePages: (...args: unknown[]) => mockListIssuePages(...args),
  addIssuePage: (...args: unknown[]) => mockAddIssuePage(...args),
  removeIssuePage: (...args: unknown[]) => mockRemoveIssuePage(...args),
}))

vi.mock('@/api/page', () => ({
  searchPages: (...args: unknown[]) => mockSearchPages(...args),
}))

vi.mock('@/composables/useI18n', () => ({
  useI18n: () => ({ t: (k: string) => k }),
}))

const toastSuccess = vi.fn()
const toastError = vi.fn()
vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ success: toastSuccess, error: toastError }),
}))

const mockCan = vi.fn(() => false)
vi.mock('@/composables/usePermission', () => ({
  useProjectPermissions: () => ({ can: (c: string) => mockCan(c) }),
}))

import IssueLinkedPagesCard from '@/components/IssueLinkedPagesCard.vue'

beforeEach(() => {
  vi.clearAllMocks()
  mockCan.mockReset()
  mockCan.mockImplementation(() => false)
  mockListIssuePages.mockResolvedValue([])
})

describe('IssueLinkedPagesCard', () => {
  it('hides mutate controls without issue:edit', async () => {
    mockCan.mockImplementation(() => false)
    const w = mount(IssueLinkedPagesCard, {
      props: { issueId: 1, projectId: 2, slug: 'ws' },
    })
    await flushPromises()
    expect(w.find('[data-test="linked-pages-add-toggle"]').exists()).toBe(false)
    expect(w.find('[data-test="linked-pages-unlink"]').exists()).toBe(false)
  })

  it('shows add when user has issue:edit', async () => {
    mockCan.mockImplementation((c: string) => c === 'issue:edit')
    const w = mount(IssueLinkedPagesCard, {
      props: { issueId: 1, projectId: 2, slug: 'ws' },
    })
    await flushPromises()
    expect(w.find('[data-test="linked-pages-add-toggle"]').exists()).toBe(true)
  })

  it('navigates to Project Pages with page query', async () => {
    mockCan.mockImplementation(() => false)
    mockListIssuePages.mockResolvedValue([{ id: 9, title: 'Spec', project_id: 2, workspace_id: 1, depth: 0 }])
    const w = mount(IssueLinkedPagesCard, {
      props: { issueId: 1, projectId: 2, slug: 'myws' },
    })
    await flushPromises()
    await w.find('button.text-left').trigger('click')
    expect(mockPush).toHaveBeenCalledWith({
      path: '/workspace/myws/project/2/pages',
      query: { page: '9' },
    })
  })
})
