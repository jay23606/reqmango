/**
 * Issue Type Types - 工作项类型定义
 */

export interface IssueType {
  id: number
  name: string
  color: string
  icon: string
  description?: string
  level?: number
  parent_type_id?: number
  allowed_child_type_ids?: number[]
  is_default: boolean
  sequence: number
  is_active: boolean
  project_id?: number
  workspace_id: number
  created_at: string
  updated_at: string
  fields?: IssueTypeField[]
  is_inherited?: boolean  // true = structurally a workspace-level type (project_id IS NULL)
  is_imported?: boolean  // true = project has explicitly imported this workspace type (workspace-type Import model)
}

export interface IssueTypeField {
  id?: number
  field_id: number
  is_required: boolean
  sequence: number
  // 包含 CustomField 的完整信息
  name?: string
  field_type?: string
  description?: string
  is_required_field?: boolean
  is_readonly?: boolean
  is_unique?: boolean
  is_active?: boolean
  options?: Array<{ id: number; field_id: number; value: string; color: string; sequence: number }>
}

export interface IssueTypeCreate {
  name: string
  color?: string
  icon?: string
  description?: string
  level?: number
  parent_type_id?: number
  allowed_child_type_ids?: number[]
  is_default?: boolean
  sequence?: number
}

export interface IssueTypeUpdate {
  name?: string
  color?: string
  icon?: string
  description?: string
  level?: number
  parent_type_id?: number
  allowed_child_type_ids?: number[]
  is_default?: boolean
  sequence?: number
}

export interface IssueTypeFieldCreate {
  field_id: number
  is_required?: boolean
  sequence?: number
}

export interface IssueTypeFieldUpdate {
  is_required?: boolean
  sequence?: number
}

// IssueType 图标选项
export const ISSUE_TYPE_ICONS = [
  'circle',
  'square',
  'bug',
  'task',
  'check-square',
  'bookmark',
  'flag',
  'star',
  'heart',
  'zap',
  'layers',
  'box',
  'database',
  'file',
  'code',
  'terminal',
  'settings',
  'users',
  'calendar',
  'clock'
]

// IssueType 颜色选项
export const ISSUE_TYPE_COLORS = [
  '#EF4444', // 红
  '#F97316', // 橙
  '#F59E0B', // 黄
  '#84CC16', // 绿
  '#10B981', // 青
  '#06B6D4', // 蓝绿
  '#0EA5E9', // 蓝
  '#6366F1', // 靛蓝
  '#8B5CF6', // 紫
  '#D946EF', // 品红
  '#EC4899', // 粉
  '#6B7280', // 灰
]

// 获取图标显示名称
export function getIconName(icon: string): string {
  const names: Record<string, string> = {
    'circle': '圆形',
    'square': '方形',
    'bug': 'Bug',
    'task': '任务',
    'check-square': '复选框',
    'bookmark': '书签',
    'flag': '旗帜',
    'star': '星标',
    'heart': '爱心',
    'zap': '闪电',
    'layers': '层级',
    'box': '箱子',
    'database': '数据库',
    'file': '文件',
    'code': '代码',
    'terminal': '终端',
    'settings': '设置',
    'users': '用户',
    'calendar': '日历',
    'clock': '时钟'
  }
  return names[icon] || icon
}
