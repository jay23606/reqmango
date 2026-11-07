import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

const { mockListRelations, mockDeleteRelation, mockCreateRelation, mockListRelationTypes } = vi.hoisted(() => ({
  mockListRelations: vi.fn().mockResolvedValue([]),
  mockDeleteRelation: vi.fn().mockResolvedValue(undefined),
  mockCreateRelation: vi.fn().mockResolvedValue({}),
  mockListRelationTypes: vi.fn().mockResolvedValue([]),
}))

vi.mock('@/api/relation', () => ({
  listIssueRelations: (...args: any[]) => mockListRelations(...args),
  deleteIssueRelation: (...args: any[]) => mockDeleteRelation(...args),
  createIssueRelation: (...args: any[]) => mockCreateRelation(...args),
  listRelationTypes: (...args: any[]) => mockListRelationTypes(...args),
}))

vi.mock('@/composables/useI18n', () => ({
  useI18n: () => ({ t: (k: string) => k }),
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/composables/usePermission', () => ({
  useProjectPermissions: () => ({
    can: (code: string) => ['issue:create', 'issue:edit'].includes(code),
  }),
}))

vi.mock('@/components/IssuePickerDialog.vue', () => ({
  default: {
    template: '<div v-if="visible" data-test="picker-dialog" class="picker-mock"><slot/></div>',
    props: ['visible', 'projectId', 'excludeId', 'title'],
    emits: ['close', 'select'],
  },
}))

import IssueTabRelations from '@/components/IssueTabRelations.vue'

const mockRelations = [
  {
    id: 101,
    direction: 'outbound',
    relation_type_id: 1,
    relation_type: { id: 1, name: 'blocks', outward_name: 'blocks' },
    related_issue_id: 55,
    inward_name: 'blocked by',
    outward_name: 'blocks',
    relation_name: 'blocks',
    related_name: 'OAuth2 setup',
    related_issue: {
      id: 55, sequence_id: 55, name: 'OAuth2 setup',
      state_name: 'Todo', state_group: 'todo', priority: 'high',
      assignees: [], target_date: null,
      issue_type: { id: 2, name: 'Feature', color: '#10b981' },
    },
  },
  {
    id: 102,
    direction: 'inbound',
    relation_type_id: 2,
    relation_type: { id: 2, name: 'relates_to', outward_name: 'relates to' },
    related_issue_id: 60,
    inward_name: 'relates to',
    outward_name: 'related from',
    relation_name: 'relates_to',
    related_name: 'Session timeout',
    related_issue: {
      id: 60, sequence_id: 60, name: 'Session timeout',
      state_name: 'Done', state_group: 'done', priority: 'medium',
      assignees: [{ id: 1, display_name: 'Alice' }], target_date: null,
      issue_type: { id: 1, name: 'Task', color: '#6366f1' },
    },
  },
  {
    id: 103,
    direction: 'outbound',
    relation_type_id: 1,
    relation_type: { id: 1, name: 'blocks', outward_name: 'blocks' },
    related_issue_id: 66,
    inward_name: 'blocked by',
    outward_name: 'blocks',
    relation_name: 'blocks',
    related_name: 'Another task',
    related_issue: {
      id: 66, sequence_id: 66, name: 'Another task',
      state_name: 'In Progress', state_group: 'in_progress', priority: 'urgent',
      assignees: [], target_date: null,
      issue_type: null,
    },
  },
]

const defaultProps = {
  issueId: 123,
  projectId: 1,
  workspaceId: 1,
  slug: 'test-workspace',
  states: [
    { id: 1, name: 'Todo', group: 'todo' },
    { id: 2, name: 'In Progress', group: 'in_progress' },
    { id: 3, name: 'Done', group: 'done' },
  ],
  parent: null,
  subIssues: [],
  issueTypes: [{ id: 2, name: 'Feature', color: '#10b981' }],
}

function mountComponent(overrides: Record<string, any> = {}) {
  return mount(IssueTabRelations, { props: { ...defaultProps, ...overrides } })
}

describe('IssueTabRelations', () => {
  const mockRelationTypes = [
    { id: 1, name: 'blocks', inward_name: 'blocked by', outward_name: 'blocks' },
    { id: 2, name: 'relates_to', inward_name: 'relates to', outward_name: 'related from' },
  ]

  beforeEach(() => {
    vi.clearAllMocks()
    mockListRelations.mockResolvedValue(mockRelations)
    mockListRelationTypes.mockResolvedValue(mockRelationTypes)
  })

  it('renders global add-relation button', () => {
    const wrapper = mountComponent()
    expect(wrapper.text()).toContain('issue.addRelation')
  })

  it('loads relations and creates one table group per relation type with data', async () => {
    const wrapper = mountComponent()
    await flushPromises()

    expect(mockListRelations).toHaveBeenCalledWith(123, 'both')
    expect(mockListRelationTypes).toHaveBeenCalledWith(1)

    // Should have 2 group tables (type 1 has 2 relations, type 2 has 1)
    const tables = wrapper.findAll('table')
    expect(tables.length).toBe(2)

    // Both relations should be rendered
    expect(wrapper.text()).toContain('OAuth2 setup')
    expect(wrapper.text()).toContain('Session timeout')
    expect(wrapper.text()).toContain('Another task')
  })

  it('shows empty state when no relations', async () => {
    mockListRelations.mockResolvedValue([])
    const wrapper = mountComponent()
    await flushPromises()

    expect(wrapper.text()).toContain('issueKanban.noRelations')
    const tables = wrapper.findAll('table')
    expect(tables.length).toBe(0)
  })

  it('handles API load failure gracefully', async () => {
    mockListRelations.mockRejectedValue(new Error('Network error'))
    const wrapper = mountComponent()
    await flushPromises()

    expect(wrapper.exists()).toBe(true)
    expect(wrapper.text()).toContain('issue.addRelation')
  })

  it('emits navigate when clicking related issue title', async () => {
    const wrapper = mountComponent()
    await flushPromises()

    const issueLinks = wrapper.findAll('.cursor-pointer')
    const relLink = issueLinks.find(el => el.text() === 'OAuth2 setup')
    expect(relLink).toBeTruthy()
    await relLink!.trigger('click')
    expect(wrapper.emitted('navigate')).toBeTruthy()
  })

  it('shows relation type dropdown when add button clicked', async () => {
    const wrapper = mountComponent()
    await flushPromises()

    const addBtn = wrapper.find('[data-test="add-relation-main"]')
    expect(addBtn.exists()).toBe(true)
    await addBtn.trigger('click')

    // Should show relation type names in dropdown
    expect(wrapper.text()).toContain('blocks')
    expect(wrapper.text()).toContain('relates to')
  })

  it('opens picker when relation type selected from dropdown', async () => {
    const wrapper = mountComponent()
    await flushPromises()

    const addBtn = wrapper.find('[data-test="add-relation-main"]')
    await addBtn.trigger('click')

    // Click first type button in dropdown
    const typeButtons = wrapper.findAll('button')
    const blocksBtn = typeButtons.find(btn => btn.text().includes('blocks'))
    expect(blocksBtn).toBeTruthy()
    await blocksBtn!.trigger('click')

    const picker = wrapper.find('[data-test="picker-dialog"]')
    expect(picker.exists()).toBe(true)
  })

  it('calls addRelation API on picker select', async () => {
    const wrapper = mountComponent()
    await flushPromises()

    const addBtn = wrapper.find('[data-test="add-relation-main"]')
    await addBtn.trigger('click')
    const typeButtons = wrapper.findAll('button')
    const blocksBtn = typeButtons.find(btn => btn.text().includes('blocks'))
    await blocksBtn!.trigger('click')

    const picker = wrapper.findComponent({ name: 'IssuePickerDialog' })
    await picker.vm.$emit('select', 42)

    await flushPromises()
    expect(mockCreateRelation).toHaveBeenCalledWith(123, {
      related_issue_id: 42,
      relation_type_id: 1,
    })
  })

  it('calls removeRelation when remove button clicked', async () => {
    const wrapper = mountComponent()
    await flushPromises()

    const removeBtns = wrapper.findAll('[data-test="remove-relation"]')
    expect(removeBtns.length).toBe(3) // 3 relations total
    await removeBtns[0].trigger('click')

    expect(mockDeleteRelation).toHaveBeenCalledWith(101)
  })

  it('groups relations by type - blocks group has 2 items', async () => {
    const wrapper = mountComponent()
    await flushPromises()

    // First table is 'blocks' group with 2 items, second is 'relates_to' with 1
    const tables = wrapper.findAll('table')
    expect(tables.length).toBe(2)

    // Check group headers
    const groupHeaders = wrapper.findAll('[data-test="relation-group-header"]')
    expect(groupHeaders.length).toBe(2)
    expect(groupHeaders[0].text()).toContain('blocks')
    expect(groupHeaders[1].text()).toContain('relates_to')
  })

  it('shows table header row with column labels', async () => {
    const wrapper = mountComponent()
    await flushPromises()

    // Table header should contain direction, type, ID, title, status, priority, assignee
    const thead = wrapper.find('thead')
    expect(thead.exists()).toBe(true)
    const headerText = thead.text()
    expect(headerText).toContain('issue.direction')
    expect(headerText).toContain('issue.relationType')
    expect(headerText).toContain('ID')
    expect(headerText).toContain('issue.title')
    expect(headerText).toContain('issue.type')
    expect(headerText).toContain('issue.status')
    expect(headerText).toContain('issue.priority')
    expect(headerText).toContain('issue.assignee')
  })

  it('direction arrows rendered in table', async () => {
    const wrapper = mountComponent()
    await flushPromises()

    // Outbound items have →, inbound have ←
    const arrows = wrapper.findAll('.text-amber-500, .text-blue-500')
    expect(arrows.length).toBeGreaterThan(0)
  })
})
