/**
 * Issue API - 工作项 API 调用模块
 */
import api from './index'
import type {
  IssueCreate,
  IssueUpdate,
  IssueResponse,
  IssueSearchResult,
  IssueActivity,
  IssueStatistics,
  IssuePriority
} from '@/types/issue'

// ==================== Issue CRUD ====================

/**
 * 创建工作项
 */
export async function createIssue(
  projectId: number,
  workspaceId: number,
  data: IssueCreate
): Promise<IssueResponse> {
  const response = await api.post(
    `/issues?project_id=${projectId}&workspace_id=${workspaceId}`,
    data
  )
  return response.data
}

/**
 * 列出项目的工作项（带分页信息）
 */
export interface IssueListResult {
  items: IssueResponse[]
  total: number
}

export async function listIssues(
  projectId: number,
  workspaceId: number,
  filters?: {
    state_id?: number
    priority?: IssuePriority
    assignee_id?: number
    parent_id?: number
    cycle_id?: number
    module_id?: number
    search?: string
    is_draft?: boolean
    issue_type_id?: number
    cf_field_id?: number
    cf_value?: string
    cf_and?: string
    rql?: string
    sort_by?: string
    sort_dir?: string
    sort_config?: string  // JSON array of {field, dir}, for multi-sort
    limit?: number
    offset?: number
  }
): Promise<IssueListResult> {
  const params = new URLSearchParams()
  params.append('project_id', projectId.toString())
  params.append('workspace_id', workspaceId.toString())
  
  if (filters) {
    if (filters.state_id) params.append('state_id', filters.state_id.toString())
    if (filters.priority) params.append('priority', filters.priority)
    if (filters.assignee_id) params.append('assignee_id', filters.assignee_id.toString())
    if (filters.rql) params.append('rql', filters.rql)
    if (filters.sort_by) params.append('sort_by', filters.sort_by)
    if (filters.sort_dir) params.append('sort_dir', filters.sort_dir)
    if (filters.sort_config) params.append('sort_config', filters.sort_config)
    if (filters.parent_id) params.append('parent_id', filters.parent_id.toString())
    if (filters.cycle_id) params.append('cycle_id', filters.cycle_id.toString())
    if (filters.module_id) params.append('module_id', filters.module_id.toString())
    if (filters.search) params.append('search', filters.search)
    if (filters.is_draft !== undefined) params.append('is_draft', filters.is_draft.toString())
    if (filters.issue_type_id) params.append('issue_type_id', filters.issue_type_id.toString())
    if (filters.cf_field_id) params.append('cf_field_id', filters.cf_field_id.toString())
    if (filters.cf_value) params.append('cf_value', filters.cf_value)
    if (filters.cf_and) params.append('cf_and', filters.cf_and)
    if (filters.limit) params.append('limit', filters.limit.toString())
    if (filters.offset) params.append('offset', filters.offset.toString())
  }
  
  const response = await api.get(`/issues?${params.toString()}`)
  const total = parseInt(response.headers['x-total-count'] || '0', 10)
  return { items: response.data || [], total }
}

/**
 * 获取工作项详情
 */
export async function getIssue(issueId: number): Promise<IssueResponse> {
  const response = await api.get(`/issues/${issueId}`)
  return response.data
}

/**
 * 更新工作项
 */
export async function updateIssue(
  issueId: number,
  data: IssueUpdate
): Promise<IssueResponse> {
  const response = await api.put(`/issues/${issueId}`, data)
  return response.data
}

/**
 * 删除工作项
 */
export async function deleteIssue(issueId: number): Promise<void> {
  await api.delete(`/issues/${issueId}`)
}

/**
 * 归档工作项
 */
export async function archiveIssue(issueId: number): Promise<void> {
  await api.post(`/issues/${issueId}/archive`)
}

/**
 * 恢复工作项
 */
export async function restoreIssue(issueId: number): Promise<IssueResponse> {
  const response = await api.post(`/issues/${issueId}/restore`)
  return response.data
}

// ==================== Issue Activities ====================

/**
 * 获取工作项活动历史
 */
export async function getIssueActivities(
  issueId: number,
  limit?: number,
  offset?: number
): Promise<IssueActivity[]> {
  const params = new URLSearchParams()
  if (limit) params.append('limit', limit.toString())
  if (offset) params.append('offset', offset.toString())
  
  const response = await api.get(`/issues/${issueId}/activities?${params.toString()}`)
  return response.data
}

// ==================== Issue Statistics ====================

/**
 * 获取项目工作项统计
 */
export async function getIssueStatistics(projectId: number): Promise<IssueStatistics> {
  const response = await api.get(`/issues/statistics?project_id=${projectId}`)
  return response.data
}

// ==================== Issue Search ====================

/**
 * 搜索工作项
 */
export async function searchIssues(
  workspaceId: number,
  query: string,
  projectId?: number,
  limit?: number
): Promise<IssueSearchResult[]> {
  const params = new URLSearchParams()
  params.append('workspace_id', workspaceId.toString())
  params.append('query', query)
  
  if (projectId) params.append('project_id', projectId.toString())
  if (limit) params.append('limit', limit.toString())
  
  const response = await api.get(`/issues/search?${params.toString()}`)
  return response.data
}

/**
 * 获取搜索建议
 */
export async function suggestIssues(
  projectId: number,
  query: string,
  limit?: number
): Promise<IssueSearchResult[]> {
  const params = new URLSearchParams()
  params.append('project_id', projectId.toString())
  params.append('query', query)
  
  if (limit) params.append('limit', limit.toString())
  
  const response = await api.get(`/issues/suggest?${params.toString()}`)
  return response.data
}

// ==================== Bulk Operations ====================

/**
 * 批量更新工作项
 */
export async function bulkUpdateIssues(
  projectId: number,
  issueIds: number[],
  data: IssueUpdate
): Promise<IssueResponse[]> {
  const response = await api.post(
    `/issues/bulk/update?project_id=${projectId}`,
    { issue_ids: issueIds, ...data }
  )
  return response.data
}

/**
 * 批量删除工作项
 */
export async function bulkDeleteIssues(issueIds: number[]): Promise<void> {
  await api.post('/issues/bulk/delete', { issue_ids: issueIds })
}

export interface ImportError {
  row: number
  title: string
  message: string
}

export interface ImportResult {
  success_count: number
  fail_count: number
  errors: ImportError[]
  imported_ids: number[]
  dry_run?: boolean
  total_rows?: number
  headers?: string[]
  mapping?: Record<string, string>
  warnings?: ImportWarning[]
}

export interface ImportWarning {
  row: number
  field: string
  message: string
}

/** Canonical CSV fields the importer accepts, in display order. */
export const IMPORT_FIELDS = [
  'name',
  'description',
  'priority',
  'state',
  'type',
  'assignees',
  'labels',
  'start_date',
  'target_date',
  'parent_title',
  'module',
  'cycle',
  'estimate',
] as const

export type ImportField = (typeof IMPORT_FIELDS)[number]

export interface ImportOptions {
  /** Validate only; the server writes nothing. */
  dryRun?: boolean
  /** Canonical field -> CSV header. */
  mapping?: Record<string, string>
}

function importQuery(projectId: number, workspaceId: number, opts?: ImportOptions): string {
  const params = new URLSearchParams({
    project_id: String(projectId),
    workspace_id: String(workspaceId),
  })
  if (opts?.dryRun) params.set('dry_run', 'true')
  return params.toString()
}

export interface ImportIssueItem {
  name: string
  description?: string
  priority?: string
  state_name?: string
  type_name?: string
  assignee_emails?: string[]
  label_names?: string[]
  start_date?: string
  target_date?: string
  parent_title?: string
}

/**
 * JSON 批量导入工作项
 */
export async function importIssuesJSON(
  projectId: number,
  workspaceId: number,
  items: ImportIssueItem[],
  opts?: ImportOptions
): Promise<ImportResult> {
  const response = await api.post(
    `/issues/import/json?${importQuery(projectId, workspaceId, opts)}`,
    items
  )
  return response.data
}

/**
 * CSV 批量导入工作项
 */
export async function importIssuesCSV(
  projectId: number,
  workspaceId: number,
  file: File,
  opts?: ImportOptions
): Promise<ImportResult> {
  const formData = new FormData()
  formData.append('file', file)
  if (opts?.mapping && Object.keys(opts.mapping).length > 0) {
    formData.append('mapping', JSON.stringify(opts.mapping))
  }
  const response = await api.post(
    `/issues/import/csv?${importQuery(projectId, workspaceId, opts)}`,
    formData,
    {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    }
  )
  return response.data
}

/**
 * 只校验不落库：返回表头、自动识别的字段映射与逐行结果。
 */
export function previewImportCSV(
  projectId: number,
  workspaceId: number,
  file: File,
  mapping?: Record<string, string>
): Promise<ImportResult> {
  return importIssuesCSV(projectId, workspaceId, file, { dryRun: true, mapping })
}

/**
 * 只校验不落库（JSON）。
 */
export function previewImportJSON(
  projectId: number,
  workspaceId: number,
  items: ImportIssueItem[]
): Promise<ImportResult> {
  return importIssuesJSON(projectId, workspaceId, items, { dryRun: true })
}

/**
 * 下载CSV导入模板
 */
export async function downloadImportTemplate(): Promise<void> {
  const response = await api.get('/issues/import/template', {
    responseType: 'blob'
  })
  const url = window.URL.createObjectURL(new Blob([response.data]))
  const link = document.createElement('a')
  link.href = url
  link.setAttribute('download', 'issue_import_template.csv')
  document.body.appendChild(link)
  link.click()
  link.remove()
  window.URL.revokeObjectURL(url)
}

// ==================== Assignee Management ====================

/**
 * 添加工作项负责人
 */
export async function addIssueAssignee(
  issueId: number,
  userId: number
): Promise<{ issue_id: number; user_id: number; action: string }> {
  const response = await api.post(`/issues/${issueId}/assignees?user_id=${userId}`)
  return response.data
}

/**
 * 移除工作项负责人
 */
export async function removeIssueAssignee(
  issueId: number,
  userId: number
): Promise<{ issue_id: number; user_id: number; action: string }> {
  const response = await api.delete(`/issues/${issueId}/assignees/${userId}`)
  return response.data
}

// ==================== Label Management ====================

/**
 * 添加工作项标签
 */
export async function addIssueLabel(
  issueId: number,
  labelId: number
): Promise<{ issue_id: number; label_id: number; action: string }> {
  const response = await api.post(`/issues/${issueId}/labels?label_id=${labelId}`)
  return response.data
}

/**
 * 移除工作项标签
 */
export async function removeIssueLabel(
  issueId: number,
  labelId: number
): Promise<{ issue_id: number; label_id: number; action: string }> {
  const response = await api.delete(`/issues/${issueId}/labels/${labelId}`)
  return response.data
}

// ==================== Cycle Management ====================

/**
 * 设置工作项周期
 */
export async function setIssueCycle(
  issueId: number,
  cycleId: number
): Promise<{ issue_id: number; cycle_id: number; action: string }> {
  const response = await api.post(`/issues/${issueId}/cycle?cycle_id=${cycleId}`)
  return response.data
}

/**
 * 移除工作项周期
 */
export async function removeIssueCycle(
  issueId: number
): Promise<{ issue_id: number; cycle_id: null; action: string }> {
  const response = await api.delete(`/issues/${issueId}/cycle`)
  return response.data
}

// ==================== Bulk Advanced Operations ====================

/**
 * 批量转换工作项类型
 */
export async function bulkConvertIssueType(
  projectId: number,
  issueIds: number[],
  issueTypeId: number
): Promise<any> {
  const res = await api.post(`/issues/bulk/convert-type?project_id=${projectId}`, {
    issue_ids: issueIds,
    issue_type_id: issueTypeId,
  })
  return res.data
}
/**
 * 批量复制工作项到另一个项目
 */
export async function bulkCopyIssues(
  projectId: number,
  issueIds: number[],
  targetProjectId: number
): Promise<any> {
  const res = await api.post(`/issues/bulk/copy?project_id=${projectId}`, {
    issue_ids: issueIds,
    target_project_id: targetProjectId,
  })
  return res.data
}

/**
 * 批量移动工作项到另一个项目
 */
export async function bulkMoveIssues(
  projectId: number,
  issueIds: number[],
  targetProjectId: number
): Promise<any> {
  const res = await api.post(`/issues/bulk/move?project_id=${projectId}`, {
    issue_ids: issueIds,
    target_project_id: targetProjectId,
  })
  return res.data
}

/**
 * Merge source issue(s) into a target issue (soft merge; same project).
 */
export async function mergeIssues(
  targetIssueId: number,
  sourceIssueIds: number[],
  options?: {
    keepSourceComments?: boolean
    keepSourceLabels?: boolean
    keepSourceAssignees?: boolean
  }
): Promise<IssueResponse> {
  const res = await api.post('/issues/merge', {
    target_issue_id: targetIssueId,
    source_issue_ids: sourceIssueIds,
    keep_source_comments: options?.keepSourceComments ?? false,
    keep_source_labels: options?.keepSourceLabels ?? true,
    keep_source_assignees: options?.keepSourceAssignees ?? true,
  })
  return res.data
}

// ==================== Sub-issues ====================

export async function reorderSubIssues(
  parentIssueId: number,
  issueIds: number[]
): Promise<any> {
  const res = await api.post(`/issues/${parentIssueId}/reorder-sub-issues`, {
    issue_ids: issueIds,
  })
  return res.data
}

// ==================== Watcher ====================

export async function addWatcher(issueId: number): Promise<any> {
  const res = await api.post(`/issues/${issueId}/watch`)
  return res.data
}

export async function removeWatcher(issueId: number): Promise<any> {
  const res = await api.delete(`/issues/${issueId}/watch`)
  return res.data
}

export async function listWatchers(issueId: number): Promise<any> {
  const res = await api.get(`/issues/${issueId}/watchers`)
  return res.data
}

// ==================== Flow Metrics ====================

/**
 * 获取工作项流转指标
 */
export async function getFlowMetrics(projectId: number): Promise<any> {
  const res = await api.get(`/projects/${projectId}/issues/flow-metrics`)
  return res.data
}

export interface AICommentRequest {
  description_html: string
  tone?: 'formal' | 'casual' | 'technical'
}

/**
 * AI 生成工作项评论
 */
export async function generateAIComment(issueId: number, data: AICommentRequest): Promise<any> {
  const res = await api.post(`/issues/${issueId}/ai-comment`, data)
  return res.data
}

// ==================== Unified API Object ====================

export async function listIssuePages(issueId: number): Promise<any[]> {
  const response = await api.get(`/issues/${issueId}/pages`)
  return response.data
}

export async function addIssuePage(issueId: number, pageId: number): Promise<any> {
  const response = await api.post(`/issues/${issueId}/pages?page_id=${pageId}`)
  return response.data
}

export async function removeIssuePage(issueId: number, pageId: number): Promise<any> {
  const response = await api.delete(`/issues/${issueId}/pages?page_id=${pageId}`)
  return response.data
}

// ==================== Export ====================

/**
 * 导出工作项（支持 CSV 和 JSON 格式）
 */
export async function exportIssues(projectId: number, format: 'csv' | 'json' = 'csv'): Promise<void> {
  const response = await api.get(`/issues/export?project_id=${projectId}&format=${format}`, {
    responseType: 'blob'
  })
  const url = window.URL.createObjectURL(new Blob([response.data]))
  const a = document.createElement('a')
  a.href = url
  a.download = `issues.${format}`
  a.click()
}

// ==================== Tree View ====================

import type {
  TreeIssueResponse
} from '@/types/issue'

export interface TreeListResult {
  items: TreeIssueResponse[]
  total: number
}

/**
 * 获取树形视图根节点（支持分页和筛选）
 */
export async function listTreeIssues(
  projectId: number,
  workspaceId: number,
  filters?: {
    state_id?: number
    priority?: string
    issue_type_id?: number
    search?: string
    rql?: string
    limit?: number
    offset?: number
    sort_by?: string
    sort_dir?: string
    sort_config?: string
  }
): Promise<TreeListResult> {
  const params = new URLSearchParams()
  params.append('project_id', projectId.toString())
  params.append('workspace_id', workspaceId.toString())
  if (filters) {
    if (filters.state_id) params.append('state_id', filters.state_id.toString())
    if (filters.priority) params.append('priority', filters.priority)
    if (filters.issue_type_id) params.append('issue_type_id', filters.issue_type_id.toString())
    if (filters.search) params.append('search', filters.search)
    if (filters.rql) params.append('rql', filters.rql)
    if (filters.limit) params.append('limit', filters.limit.toString())
    if (filters.offset) params.append('offset', filters.offset.toString())
    if (filters.sort_by) params.append('sort_by', filters.sort_by)
    if (filters.sort_dir) params.append('sort_dir', filters.sort_dir)
    if (filters.sort_config) params.append('sort_config', filters.sort_config)
  }
  const response = await api.get(`/issues/tree?${params.toString()}`)
  const total = parseInt(response.headers['x-total-count'] || '0', 10)
  return { items: response.data || [], total }
}

/**
 * 获取子节点（懒加载）
 */
export async function getIssueChildren(issueId: number): Promise<TreeIssueResponse[]> {
  const response = await api.get(`/issues/${issueId}/children`)
  return response.data || []
}

/** Similar issue found by pre-create duplicate check (warn-only). */
export interface DuplicateIssueItem {
  id: number
  sequence_id: number
  name: string
  priority: string
  state_id?: number
  similarity?: number
}

export interface DuplicateCheckResult {
  duplicates: DuplicateIssueItem[]
}

/**
 * Check for similar issues before create (warn-only; never blocks create).
 */
export async function checkDuplicates(
  projectId: number,
  data: { name: string; description?: string },
  limit = 5
): Promise<DuplicateCheckResult> {
  const response = await api.post(
    `/projects/${projectId}/issues/duplicate-check?limit=${limit}`,
    {
      name: data.name,
      description: data.description || '',
    }
  )
  return response.data || { duplicates: [] }
}

export const issueApi = {
  // CRUD
  createIssue,
  listIssues,
  getIssue,
  updateIssue,
  deleteIssue,
  archiveIssue,
  restoreIssue,
  
  // Activities
  getIssueActivities,
  
  // Statistics
  getIssueStatistics,
  
  // Search
  searchIssues,
  
  // Import
	downloadImportTemplate,

	// Export
  exportIssues,
  
  // Tree
  listTreeIssues,
  getIssueChildren,

  // Duplicate check (pre-create warn-only)
  checkDuplicates,
  
  // Bulk
  bulkUpdateIssues,
  bulkDeleteIssues,
  importIssuesJSON,
  importIssuesCSV,
  previewImportCSV,
  previewImportJSON,
  
  // Assignee
  addIssueAssignee,
  removeIssueAssignee,
  
  // Label
  addIssueLabel,
  removeIssueLabel,
  
  // Cycle
  setIssueCycle,
  removeIssueCycle,
  
  // Pages
  listIssuePages,
  addIssuePage,
  removeIssuePage,

  // Bulk Advanced
  bulkConvertIssueType,
  bulkCopyIssues,
  bulkMoveIssues,
  mergeIssues,

  // Flow Metrics
  getFlowMetrics,

  // Sub-issues
  reorderSubIssues,

  // Watcher
  addWatcher,
  removeWatcher,
  listWatchers,

  // AI
  generateAIComment,
}

export default issueApi