import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'

vi.mock('@/composables/useI18n', () => ({
  useI18n: () => ({ t: (k: string) => k }),
}))

import RelationSubIssuesCard from '@/components/RelationSubIssuesCard.vue'

const mockSubIssues = [
  {
    id: 43,
    sequence_id: 43,
    name: 'Email validation',
    state_name: 'Done',
    state_group: 'done',
    priority: 'low',
    assignees: [{ id: 2, display_name: 'Bob' }],
    target_date: '2026-07-08T00:00:00Z',
    issue_type: { id: 1, name: 'Task', color: '#6366f1' },
  },
  {
    id: 44,
    sequence_id: 44,
    name: 'Password checker',
    state_name: 'Todo',
    state_group: 'todo',
    priority: 'high',
    assignees: [],
    target_date: null,
    issue_type: { id: 1, name: 'Task', color: '#6366f1' },
  },
]

function mountCard(overrides: Record<string, any> = {}) {
  const defaults = {
    subIssues: mockSubIssues,
  }
  return mount(RelationSubIssuesCard, {
    props: { ...defaults, ...overrides },
  })
}

describe('RelationSubIssuesCard', () => {
  it('renders column headers in the table', () => {
    const wrapper = mountCard()
    const text = wrapper.text()
    expect(text).toContain('subIssue.title')
  })

  it('shows completion count 1/2', () => {
    const wrapper = mountCard()
    expect(wrapper.text()).toContain('1/2')
  })

  it('renders each sub-issue as a row', () => {
    const wrapper = mountCard()
    const text = wrapper.text()
    expect(text).toContain('Email validation')
    expect(text).toContain('Password checker')
  })

  it('shows dropdown menu when + Add button is clicked', async () => {
    const wrapper = mountCard()
    await wrapper.find('[data-test="add-subissue"]').trigger('click')
    expect(wrapper.find('[data-test="add-existing-subissue"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="quick-create-subissue"]').exists()).toBe(true)
  })

  it('emits "add-existing" when select existing is clicked', async () => {
    const wrapper = mountCard()
    await wrapper.find('[data-test="add-subissue"]').trigger('click')
    await wrapper.find('[data-test="add-existing-subissue"]').trigger('click')
    expect(wrapper.emitted('add-existing')).toBeTruthy()
  })

  it('shows inline input when quick create is clicked', async () => {
    const wrapper = mountCard()
    await wrapper.find('[data-test="add-subissue"]').trigger('click')
    await wrapper.find('[data-test="quick-create-subissue"]').trigger('click')
    expect(wrapper.find('input[type="text"]').exists()).toBe(true)
  })

  it('emits "quick-create" with name when Enter is pressed in quick input', async () => {
    const wrapper = mountCard()
    await wrapper.find('[data-test="add-subissue"]').trigger('click')
    await wrapper.find('[data-test="quick-create-subissue"]').trigger('click')
    const input = wrapper.find('input[type="text"]')
    await input.setValue('New sub-issue')
    await input.trigger('keydown.enter')
    expect(wrapper.emitted('quick-create')).toBeTruthy()
    expect(wrapper.emitted('quick-create')![0]).toEqual(['New sub-issue'])
  })

  it('emits "navigate" with issue ID when title is clicked', async () => {
    const wrapper = mountCard()
    await wrapper.find('.subissue-clickable').trigger('click')
    expect(wrapper.emitted('navigate')).toBeTruthy()
    expect(wrapper.emitted('navigate')![0]).toEqual([43])
  })

  it('emits "toggle" when checkbox is clicked', async () => {
    const wrapper = mountCard()
    const checkbox = wrapper.find('input[type="checkbox"]')
    await checkbox.trigger('change')
    expect(wrapper.emitted('toggle')).toBeTruthy()
  })

  it('shows empty state when no sub-issues', () => {
    const wrapper = mountCard({ subIssues: [] })
    expect(wrapper.text()).toContain('subIssue.noSubIssues')
    expect(wrapper.text()).toContain('subIssue.emptyHint')
  })

  it('shows sub-item count 0 in header when list is empty', () => {
    const wrapper = mountCard({ subIssues: [] })
    const badge = wrapper.find('[data-test="subissue-count-badge"]')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toBe('0')
  })

  it('empty state shows quick-create and add-existing CTAs', () => {
    const wrapper = mountCard({ subIssues: [] })
    expect(wrapper.find('[data-test="empty-quick-create-subissue"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="empty-add-existing-subissue"]').exists()).toBe(true)
  })

  it('hides add dropdown when both quick-create and link-existing are denied', () => {
    const wrapper = mountCard({ subIssues: [], canQuickCreate: false, canLinkExisting: false })
    expect(wrapper.find('[data-test="add-subissue"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="empty-quick-create-subissue"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="empty-add-existing-subissue"]').exists()).toBe(false)
  })

  it('shows hierarchy error banner and dismiss emits', async () => {
    const wrapper = mountCard({
      subIssues: [],
      hierarchyError: 'Maximum hierarchy depth (6 levels) exceeded',
    })
    const banner = wrapper.find('[data-test="subissue-hierarchy-error"]')
    expect(banner.exists()).toBe(true)
    expect(banner.text()).toContain('Maximum hierarchy depth')
    await wrapper.find('button[aria-label="common.close"]').trigger('click')
    expect(wrapper.emitted('dismiss-hierarchy-error')).toBeTruthy()
  })
})
