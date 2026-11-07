/**
 * Issue API 单元测试 - Mock axios 验证端点 URL 和参数
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { IssuePriority } from '@/types/issue'

const mockGet = vi.fn()
const mockPost = vi.fn()
const mockPut = vi.fn()
const mockDelete = vi.fn()

vi.mock('./index', () => ({
  default: {
    get: (...args: any[]) => mockGet(...args),
    post: (...args: any[]) => mockPost(...args),
    put: (...args: any[]) => mockPut(...args),
    delete: (...args: any[]) => mockDelete(...args),
  },
}))

import {
  createIssue, listIssues, getIssue, updateIssue, deleteIssue,
  archiveIssue, restoreIssue, getIssueActivities, getIssueStatistics,
  searchIssues, suggestIssues,
  bulkUpdateIssues, bulkDeleteIssues, importIssuesJSON, exportIssues,
  addIssueAssignee, removeIssueAssignee,
  addIssueLabel, removeIssueLabel,
  setIssueCycle, removeIssueCycle,
  listIssuePages, addIssuePage, removeIssuePage,
  bulkConvertIssueType, bulkCopyIssues, bulkMoveIssues, mergeIssues,
  getFlowMetrics, generateAIComment,
  listTreeIssues, getIssueChildren, checkDuplicates,
  issueApi,
} from './issue'

beforeEach(() => {
  vi.clearAllMocks()
  vi.stubGlobal('URL', {
    createObjectURL: vi.fn(() => 'blob:test-url'),
    revokeObjectURL: vi.fn()
  })
})

// ==================== CRUD ====================

describe('issue CRUD API', () => {
  it('createIssue should POST with project/workspace', async () => {
    mockPost.mockResolvedValue({ data: { id: 1, name: 'Test' } })
    const result = await createIssue(1, 2, { name: 'Test', project_id: 1 } as any)
    expect(mockPost).toHaveBeenCalledWith('/issues?project_id=1&workspace_id=2', { name: 'Test', project_id: 1 })
    expect(result.id).toBe(1)
  })

  it('listIssues should GET with params', async () => {
    mockGet.mockResolvedValue({ data: [], headers: {} })
    const result = await listIssues(1, 2)
    expect(result.items).toEqual([])
    expect(result.total).toBe(0)
  })

  it('listIssues should include filter params', async () => {
    mockGet.mockResolvedValue({ data: [{ id: 1 }], headers: { 'x-total-count': '1' } })
    const result = await listIssues(1, 2, { state_id: 3, priority: IssuePriority.HIGH, rql: 'state=open' })
    const url = mockGet.mock.calls[0][0]
    expect(url).toContain('project_id=1')
    expect(url).toContain('workspace_id=2')
    expect(url).toContain('state_id=3')
    expect(url).toContain('priority=high')
    expect(url).toContain('rql=state%3Dopen')
    expect(result.total).toBe(1)
  })

  it('getIssue should GET by id', async () => {
    mockGet.mockResolvedValue({ data: { id: 42, name: 'Bug' } })
    const result = await getIssue(42)
    expect(mockGet).toHaveBeenCalledWith('/issues/42')
    expect(result.id).toBe(42)
  })

  it('updateIssue should PUT by id', async () => {
    mockPut.mockResolvedValue({ data: { id: 42, name: 'Updated' } })
    const result = await updateIssue(42, { name: 'Updated' } as any)
    expect(mockPut).toHaveBeenCalledWith('/issues/42', { name: 'Updated' })
    expect(result.name).toBe('Updated')
  })

  it('deleteIssue should DELETE by id', async () => {
    mockDelete.mockResolvedValue({})
    await deleteIssue(42)
    expect(mockDelete).toHaveBeenCalledWith('/issues/42')
  })

  it('archiveIssue should POST to archive', async () => {
    mockPost.mockResolvedValue({})
    await archiveIssue(42)
    expect(mockPost).toHaveBeenCalledWith('/issues/42/archive')
  })

  it('restoreIssue should POST to restore', async () => {
    mockPost.mockResolvedValue({ data: { id: 42 } })
    const result = await restoreIssue(42)
    expect(mockPost).toHaveBeenCalledWith('/issues/42/restore')
    expect(result.id).toBe(42)
  })
})

// ==================== Activities ====================

describe('issue activities API', () => {
  it('getIssueActivities should GET activities', async () => {
    mockGet.mockResolvedValue({ data: [{ id: 1, field: 'status' }] })
    const result = await getIssueActivities(42)
    expect(mockGet).toHaveBeenCalledWith('/issues/42/activities?')
    expect(result).toHaveLength(1)
  })

  it('getIssueActivities should pass pagination', async () => {
    mockGet.mockResolvedValue({ data: [] })
    await getIssueActivities(42, 10, 20)
    const url = mockGet.mock.calls[0][0]
    expect(url).toContain('limit=10')
    expect(url).toContain('offset=20')
  })
})

// ==================== Statistics ====================

describe('issue statistics API', () => {
  it('getIssueStatistics should GET stats', async () => {
    mockGet.mockResolvedValue({ data: { total: 100, open: 50 } })
    const result = await getIssueStatistics(1)
    expect(mockGet).toHaveBeenCalledWith('/issues/statistics?project_id=1')
    expect(result.total).toBe(100)
  })
})

// ==================== Search ====================

describe('issue search API', () => {
  it('searchIssues should GET search', async () => {
    mockGet.mockResolvedValue({ data: [{ id: 1, name: 'Found' }] })
    const result = await searchIssues(1, 'bug', 2, 10)
    const url = mockGet.mock.calls[0][0]
    expect(url).toContain('workspace_id=1')
    expect(url).toContain('query=bug')
    expect(url).toContain('project_id=2')
    expect(url).toContain('limit=10')
    expect(result).toHaveLength(1)
  })

  it('searchIssues without optional params', async () => {
    mockGet.mockResolvedValue({ data: [] })
    await searchIssues(1, 'test')
    const url = mockGet.mock.calls[0][0]
    expect(url).not.toContain('project_id')
    expect(url).not.toContain('limit')
  })

  it('suggestIssues should GET suggest', async () => {
    mockGet.mockResolvedValue({ data: [{ id: 1 }] })
    const result = await suggestIssues(1, 'tes', 5)
    const url = mockGet.mock.calls[0][0]
    expect(url).toContain('project_id=1')
    expect(url).toContain('query=tes')
    expect(url).toContain('limit=5')
    expect(result).toHaveLength(1)
  })
})

// ==================== Bulk Operations ====================

describe('bulk operations API', () => {
  it('bulkUpdateIssues should POST bulk update', async () => {
    mockPost.mockResolvedValue({ data: [{ id: 1 }, { id: 2 }] })
    const result = await bulkUpdateIssues(1, [1, 2], { priority: 'low' } as any)
    expect(mockPost).toHaveBeenCalledWith('/issues/bulk/update?project_id=1', {
      issue_ids: [1, 2], priority: 'low'
    })
    expect(result).toHaveLength(2)
  })

  it('bulkDeleteIssues should POST bulk delete', async () => {
    mockPost.mockResolvedValue({})
    await bulkDeleteIssues([1, 2, 3])
    expect(mockPost).toHaveBeenCalledWith('/issues/bulk/delete', { issue_ids: [1, 2, 3] })
  })

  it('importIssuesJSON should POST import', async () => {
    const items = [{ name: 'Task 1' }]
    mockPost.mockResolvedValue({ data: { success_count: 1, fail_count: 0, errors: [], imported_ids: [1] } })
    const result = await importIssuesJSON(1, 2, items)
    expect(mockPost).toHaveBeenCalledWith('/issues/import/json?project_id=1&workspace_id=2', items)
    expect(result.success_count).toBe(1)
  })

  it('previewImportCSV should dry-run with the mapping', async () => {
    mockPost.mockResolvedValue({ data: { success_count: 1, fail_count: 0, errors: [], imported_ids: [], dry_run: true } })
    const file = new File(['title\nA\n'], 'issues.csv', { type: 'text/csv' })
    const { previewImportCSV } = await import('./issue')
    await previewImportCSV(1, 2, file, { name: 'title' })

    const [url, body] = mockPost.mock.calls[0]
    expect(url).toBe('/issues/import/csv?project_id=1&workspace_id=2&dry_run=true')
    expect((body as FormData).get('mapping')).toBe(JSON.stringify({ name: 'title' }))
  })

  it('previewImportJSON should dry-run', async () => {
    mockPost.mockResolvedValue({ data: { success_count: 1, fail_count: 0, errors: [], imported_ids: [], dry_run: true } })
    const { previewImportJSON } = await import('./issue')
    await previewImportJSON(1, 2, [{ name: 'A' }])
    expect(mockPost).toHaveBeenCalledWith('/issues/import/json?project_id=1&workspace_id=2&dry_run=true', [{ name: 'A' }])
  })

  it('bulkConvertIssueType should POST convert', async () => {
    mockPost.mockResolvedValue({ data: { message: 'ok' } })
    await bulkConvertIssueType(1, [1, 2], 5)
    expect(mockPost).toHaveBeenCalledWith('/issues/bulk/convert-type?project_id=1', {
      issue_ids: [1, 2], issue_type_id: 5
    })
  })

  it('bulkCopyIssues should POST copy', async () => {
    mockPost.mockResolvedValue({ data: [] })
    await bulkCopyIssues(1, [1, 2], 3)
    expect(mockPost).toHaveBeenCalledWith('/issues/bulk/copy?project_id=1', {
      issue_ids: [1, 2], target_project_id: 3
    })
  })

  it('bulkMoveIssues should POST move', async () => {
    mockPost.mockResolvedValue({ data: [] })
    await bulkMoveIssues(1, [1], 3)
    expect(mockPost).toHaveBeenCalledWith('/issues/bulk/move?project_id=1', {
      issue_ids: [1], target_project_id: 3
    })
  })

  it('mergeIssues should POST merge', async () => {
    mockPost.mockResolvedValue({ data: { id: 2 } })
    await mergeIssues(2, [1])
    expect(mockPost).toHaveBeenCalledWith('/issues/merge', {
      target_issue_id: 2,
      source_issue_ids: [1],
      keep_source_comments: false,
      keep_source_labels: true,
      keep_source_assignees: true,
    })
  })
})

// ==================== Assignee/Label/Cycle ====================

describe('assignee management API', () => {
  it('addIssueAssignee should POST', async () => {
    mockPost.mockResolvedValue({ data: { issue_id: 42, user_id: 7, action: 'added' } })
    const result = await addIssueAssignee(42, 7)
    expect(mockPost).toHaveBeenCalledWith('/issues/42/assignees?user_id=7')
    expect(result.user_id).toBe(7)
  })

  it('removeIssueAssignee should DELETE', async () => {
    mockDelete.mockResolvedValue({ data: { issue_id: 42, user_id: 7, action: 'removed' } })
    const result = await removeIssueAssignee(42, 7)
    expect(mockDelete).toHaveBeenCalledWith('/issues/42/assignees/7')
    expect(result.action).toBe('removed')
  })
})

describe('label management API', () => {
  it('addIssueLabel should POST', async () => {
    mockPost.mockResolvedValue({ data: { issue_id: 42, label_id: 3, action: 'added' } })
    const result = await addIssueLabel(42, 3)
    expect(mockPost).toHaveBeenCalledWith('/issues/42/labels?label_id=3')
    expect(result.label_id).toBe(3)
  })

  it('removeIssueLabel should DELETE', async () => {
    mockDelete.mockResolvedValue({ data: { issue_id: 42, label_id: 3, action: 'removed' } })
    const result = await removeIssueLabel(42, 3)
    expect(mockDelete).toHaveBeenCalledWith('/issues/42/labels/3')
    expect(result.action).toBe('removed')
  })
})

describe('cycle management API', () => {
  it('setIssueCycle should POST', async () => {
    mockPost.mockResolvedValue({ data: { issue_id: 42, cycle_id: 5, action: 'set' } })
    const result = await setIssueCycle(42, 5)
    expect(mockPost).toHaveBeenCalledWith('/issues/42/cycle?cycle_id=5')
    expect(result.cycle_id).toBe(5)
  })

  it('removeIssueCycle should DELETE', async () => {
    mockDelete.mockResolvedValue({ data: { issue_id: 42, cycle_id: null, action: 'removed' } })
    const result = await removeIssueCycle(42)
    expect(mockDelete).toHaveBeenCalledWith('/issues/42/cycle')
    expect(result.cycle_id).toBeNull()
  })
})

// ==================== Pages ====================

describe('issue pages API', () => {
  it('listIssuePages should GET', async () => {
    mockGet.mockResolvedValue({ data: [{ id: 1 }] })
    const result = await listIssuePages(42)
    expect(mockGet).toHaveBeenCalledWith('/issues/42/pages')
    expect(result).toHaveLength(1)
  })

  it('addIssuePage should POST', async () => {
    mockPost.mockResolvedValue({ data: { issue_id: 42, page_id: 10 } })
    await addIssuePage(42, 10)
    expect(mockPost).toHaveBeenCalledWith('/issues/42/pages?page_id=10')
  })

  it('removeIssuePage should DELETE', async () => {
    mockDelete.mockResolvedValue({ data: {} })
    await removeIssuePage(42, 10)
    expect(mockDelete).toHaveBeenCalledWith('/issues/42/pages?page_id=10')
  })
})

// ==================== Export ====================

describe('export issues API', () => {
  it('exportIssues should GET with blob response', async () => {
    mockGet.mockResolvedValue({ data: new Blob() })
    await exportIssues(1, 'csv')
    expect(mockGet).toHaveBeenCalledWith('/issues/export?project_id=1&format=csv', { responseType: 'blob' })
  })

  it('exportIssues should default to csv', async () => {
    mockGet.mockResolvedValue({ data: new Blob() })
    await exportIssues(1)
    expect(mockGet).toHaveBeenCalledWith('/issues/export?project_id=1&format=csv', { responseType: 'blob' })
  })
})

// ==================== Tree View ====================

describe('tree view API', () => {
  it('listTreeIssues should GET tree', async () => {
    mockGet.mockResolvedValue({ data: [], headers: {} })
    const result = await listTreeIssues(1, 100)
    expect(mockGet).toHaveBeenCalledWith('/issues/tree?project_id=1&workspace_id=100')
    expect(result.items).toEqual([])
    expect(result.total).toBe(0)
  })

  it('listTreeIssues with filters', async () => {
    mockGet.mockResolvedValue({ data: [{ id: 1, children: [] }], headers: { 'x-total-count': '1' } })
    const result = await listTreeIssues(1, 100, { state_id: 3, rql: 'priority=high' })
    const url = mockGet.mock.calls[0][0]
    expect(url).toContain('state_id=3')
    expect(url).toContain('rql=priority%3Dhigh')
    expect(result.total).toBe(1)
  })

  it('getIssueChildren should GET children', async () => {
    mockGet.mockResolvedValue({ data: [{ id: 2 }] })
    const result = await getIssueChildren(42)
    expect(mockGet).toHaveBeenCalledWith('/issues/42/children')
    expect(result).toHaveLength(1)
  })

  it('checkDuplicates should POST to duplicate-check', async () => {
    mockPost.mockResolvedValue({
      data: { duplicates: [{ id: 9, sequence_id: 3, name: 'Fix login', priority: 'high' }] },
    })
    const result = await checkDuplicates(1, { name: 'Fix login bug', description: 'mobile' })
    expect(mockPost).toHaveBeenCalledWith(
      '/projects/1/issues/duplicate-check?limit=5',
      { name: 'Fix login bug', description: 'mobile' }
    )
    expect(result.duplicates).toHaveLength(1)
    expect(result.duplicates[0].id).toBe(9)
  })
})

// ==================== Flow Metrics & AI ====================

describe('flow metrics & AI API', () => {
  it('getFlowMetrics should GET flow metrics', async () => {
    mockGet.mockResolvedValue({ data: { avg_cycle_time: 3.5 } })
    const result = await getFlowMetrics(1)
    expect(mockGet).toHaveBeenCalledWith('/projects/1/issues/flow-metrics')
    expect(result.avg_cycle_time).toBe(3.5)
  })

  it('generateAIComment should POST', async () => {
    mockPost.mockResolvedValue({ data: { comment: 'AI review' } })
    const result = await generateAIComment(42, { description_html: '<p>test</p>' })
    expect(mockPost).toHaveBeenCalledWith('/issues/42/ai-comment', { description_html: '<p>test</p>' })
    expect(result.comment).toBe('AI review')
  })
})

// ==================== issueApi export ====================

describe('issueApi export', () => {
  it('should export all methods', () => {
    const methods = [
      'createIssue', 'listIssues', 'getIssue', 'updateIssue', 'deleteIssue',
      'archiveIssue', 'restoreIssue', 'getIssueActivities', 'getIssueStatistics',
      'searchIssues', 'downloadImportTemplate', 'exportIssues',
      'listTreeIssues', 'getIssueChildren', 'checkDuplicates',
      'bulkUpdateIssues', 'bulkDeleteIssues', 'importIssuesJSON', 'importIssuesCSV',
      'addIssueAssignee', 'removeIssueAssignee',
      'addIssueLabel', 'removeIssueLabel',
      'setIssueCycle', 'removeIssueCycle',
      'listIssuePages', 'addIssuePage', 'removeIssuePage',
      'bulkConvertIssueType', 'bulkCopyIssues', 'bulkMoveIssues', 'mergeIssues',
      'getFlowMetrics', 'generateAIComment',
    ]
    for (const m of methods) {
      expect(issueApi).toHaveProperty(m)
      expect(typeof (issueApi as any)[m]).toBe('function')
    }
  })
})
