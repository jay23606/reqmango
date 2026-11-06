import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import IssueTabAttachments from '@/components/IssueTabAttachments.vue'

vi.mock('@/composables/useI18n', () => ({
  useI18n: () => ({ t: (k: string) => k }),
}))

vi.mock('@/composables/usePermission', () => ({
  useProjectPermissions: () => ({ can: (code: string) => code === 'issue:edit' }),
}))

vi.mock('@/components/AttachmentManager.vue', () => ({
  default: {
    template: '<div class="mock-attach">Attachments</div>',
    props: ['issueId', 'projectId'],
  },
}))

const defaultProps = {
  issueId: 1,
  projectId: 1,
}

function mountComponent(overrides: Record<string, any> = {}) {
  return mount(IssueTabAttachments, {
    props: { ...defaultProps, ...overrides },
  })
}

describe('IssueTabAttachments', () => {
  it('renders AttachmentManager', () => {
    const wrapper = mountComponent()
    expect(wrapper.text()).toContain('Attachments')
  })

  it('passes props correctly', () => {
    const wrapper = mountComponent({ issueId: 42, projectId: 7 })
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.find('.mock-attach').exists()).toBe(true)
  })
})
