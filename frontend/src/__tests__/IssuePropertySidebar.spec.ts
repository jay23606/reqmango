import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import IssuePropertySidebar from '@/components/IssuePropertySidebar.vue'

const { mockActivity } = vi.hoisted(() => ({ mockActivity: vi.fn() }))

vi.mock('@/api/agent', () => ({
  agentApi: { listWorkspaceActivity: (...args: any[]) => mockActivity(...args) },
}))
vi.mock('@/composables/useI18n', () => ({
  useI18n: () => ({ t: (k: string) => k }),
}))
const mockToastWarning = vi.fn()
vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ success: vi.fn(), error: vi.fn(), warning: (...a: unknown[]) => mockToastWarning(...a) }),
}))

const mockIssue = {
  id: 42, state_id: 1, priority: 'high',
  assignees: [{ id: 1, display_name: 'Alice' }],
  cycle_id: 5, start_date: '2026-07-01', target_date: '2026-07-10',
}

const mockStates = [
  { id: 1, name: 'To Do' },
  { id: 2, name: 'In Progress' },
]
const mockMembers = [
  { id: 1, display_name: 'Alice' },
  { id: 2, display_name: 'Bob' },
]
const mockCycles = [
  { id: 5, name: 'Sprint 5' },
  { id: 6, name: 'Sprint 6' },
]
const mockModules = [
  { id: 1, name: 'Module A' },
  { id: 2, name: 'Module B' },
]

const mountOptions = {
  props: {
    issue: mockIssue,
    states: mockStates,
    members: mockMembers,
    cycles: mockCycles,
    modules: mockModules,
    releases: [],
    customFields: [],
    workspaceId: 1,
  },
  global: {
    stubs: {
      AgentSelector: true,
      LabelSelector: true,
      teleport: true,
    },
  },
}

describe('IssuePropertySidebar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockActivity.mockResolvedValue([])
    mockToastWarning.mockClear()
  })

  it('renders state, priority and assignee pickers with current values', () => {
    const wrapper = mount(IssuePropertySidebar, mountOptions)
    expect(wrapper.text()).toContain('issue.state')
    expect(wrapper.find('[data-testid="picker-state"]').text()).toContain('To Do')
    expect(wrapper.find('[data-testid="picker-priority"]').text()).toContain('issue.priorityHigh')
    expect(wrapper.find('[data-testid="picker-assignee"]').text()).toContain('Alice')
    expect(wrapper.find('[data-testid="picker-cycle"]').text()).toContain('Sprint 5')
  })

  async function pickOption(wrapper: ReturnType<typeof mount>, testId: string, label: string) {
    await wrapper.find(`[data-testid="${testId}"] button`).trigger('click')
    const option = wrapper.findAll('[role="option"]').find((o) => o.text().includes(label))
    await option!.trigger('mousedown')
  }

  it('emits update:state when a state is picked', async () => {
    const wrapper = mount(IssuePropertySidebar, mountOptions)
    await pickOption(wrapper, 'picker-state', 'In Progress')
    expect(wrapper.emitted('update:state')![0]).toEqual([2])
  })

  it('emits update:priority when a priority is picked', async () => {
    const wrapper = mount(IssuePropertySidebar, mountOptions)
    await pickOption(wrapper, 'picker-priority', 'issue.priorityUrgent')
    expect(wrapper.emitted('update:priority')![0]).toEqual(['urgent'])
  })

  it('emits update:assignee null when cleared', async () => {
    const wrapper = mount(IssuePropertySidebar, mountOptions)
    await pickOption(wrapper, 'picker-assignee', 'issue.unassigned')
    expect(wrapper.emitted('update:assignee')![0]).toEqual([null])
  })

  it('disables pickers while approval is pending', () => {
    const wrapper = mount(IssuePropertySidebar, {
      ...mountOptions,
      props: { ...mountOptions.props, issue: { ...mockIssue, approval_status: 'pending' } },
    })
    expect(wrapper.find('[data-testid="picker-state"] button').attributes('disabled')).toBeDefined()
    expect(wrapper.find('[data-testid="picker-priority"] button').attributes('disabled')).toBeDefined()
  })

  it('renders start date and target date inputs', () => {
    const wrapper = mount(IssuePropertySidebar, mountOptions)
    const dateInputs = wrapper.findAll('input[type="date"]')
    expect(dateInputs.length).toBe(2)
    expect(dateInputs[0].element.value).toBe('2026-07-01')
    expect(dateInputs[1].element.value).toBe('2026-07-10')
  })

  it('emits update:startDate on date change', () => {
    const wrapper = mount(IssuePropertySidebar, mountOptions)
    const dateInputs = wrapper.findAll('input[type="date"]')
    dateInputs[0].setValue('2026-08-01')
    expect(wrapper.emitted('update:startDate')).toBeTruthy()
    expect(wrapper.emitted('update:startDate')![0]).toEqual(['2026-08-01'])
  })

  it('shows required marker for mandatory custom fields', () => {
    const wrapper = mount(IssuePropertySidebar, {
      ...mountOptions,
      props: {
        ...mountOptions.props,
        customFields: [
          {
            field: {
              id: 9,
              name: 'Repro',
              field_type: 'text',
              is_required: true,
            },
            value: 'filled',
          },
        ],
      },
    })
    const label = wrapper.findAll('label.prop-label').find((l) => l.text().includes('Repro'))
    expect(label?.text()).toContain('*')
  })

  it('blocks clearing a required custom field and toasts', async () => {
    const wrapper = mount(IssuePropertySidebar, {
      ...mountOptions,
      props: {
        ...mountOptions.props,
        customFields: [
          {
            field: {
              id: 9,
              name: 'Repro',
              field_type: 'text',
              is_required: true,
            },
            value: 'steps',
          },
        ],
      },
    })
    const el = wrapper.find('input.prop-input[type="text"]').element as HTMLInputElement
    el.value = ''
    await el.dispatchEvent(new Event('change', { bubbles: true }))
    expect(mockToastWarning).toHaveBeenCalled()
    expect(wrapper.emitted('update:customField')).toBeUndefined()
  })

})
