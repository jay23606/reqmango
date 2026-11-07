import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import MergeIntoIssueDialog from '@/components/MergeIntoIssueDialog.vue'

vi.mock('@/composables/useI18n', () => ({
  useI18n: () => ({ t: (k: string) => k }),
}))
vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ success: vi.fn(), error: vi.fn() }),
}))

const mockSuggest = vi.fn()
const mockGetIssue = vi.fn()
const mockMerge = vi.fn()

vi.mock('@/api/issue', () => ({
  suggestIssues: (...args: unknown[]) => mockSuggest(...args),
  getIssue: (...args: unknown[]) => mockGetIssue(...args),
  mergeIssues: (...args: unknown[]) => mockMerge(...args),
}))

const baseProps = {
  visible: true,
  projectId: 5,
  sourceIssueId: 10,
  sourceLabel: 'DEV-10 Source issue',
  projectIdentifier: 'DEV',
}

describe('MergeIntoIssueDialog', () => {
  beforeEach(() => {
    mockSuggest.mockReset()
    mockGetIssue.mockReset()
    mockMerge.mockReset()
    mockSuggest.mockResolvedValue([
      { id: 10, sequence_id: 10, name: 'Self' },
      { id: 20, sequence_id: 20, name: 'Target' },
    ])
    mockGetIssue.mockResolvedValue({ id: 20, sequence_id: 20, name: 'Target', project_id: 5 })
  })

  it('skips prefill when suggested target is the source issue', async () => {
    mockGetIssue.mockResolvedValueOnce({ id: 10, sequence_id: 10, name: 'Self', project_id: 5 })
    const wrapper = mount(MergeIntoIssueDialog, {
      props: { ...baseProps, initialTargetId: 10 },
    })
    await flushPromises()
    expect(wrapper.find('[data-test="merge-into-search"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="merge-into-confirm"]').exists()).toBe(false)
  })

  it('opens on confirm step when prefill target is valid', async () => {
    const wrapper = mount(MergeIntoIssueDialog, {
      props: { ...baseProps, initialTargetId: 20 },
    })
    await flushPromises()
    const btn = wrapper.find('[data-test="merge-into-confirm-btn"]')
    expect(btn.exists()).toBe(true)
    expect((btn.element as HTMLButtonElement).disabled).toBe(false)
  })

  it('disables confirm while merge request is in flight', async () => {
    const wrapper = mount(MergeIntoIssueDialog, {
      props: { ...baseProps, initialTargetId: 20 },
    })
    await flushPromises()
    const btn = wrapper.find('[data-test="merge-into-confirm-btn"]')
    mockMerge.mockImplementation(() => new Promise(() => {}))
    await btn.trigger('click')
    await flushPromises()
    expect((btn.element as HTMLButtonElement).disabled).toBe(true)
  })

  it('excludes source id from search results', async () => {
    const wrapper = mount(MergeIntoIssueDialog, { props: baseProps })
    await wrapper.find('[data-test="merge-into-search"]').setValue('tar')
    await new Promise((r) => setTimeout(r, 350))
    await flushPromises()
    const resultBtns = wrapper.findAll('[data-test="merge-into-result"]')
    expect(resultBtns.some((b) => b.text().includes('#10'))).toBe(false)
    expect(resultBtns.some((b) => b.text().includes('#20'))).toBe(true)
  })
})
