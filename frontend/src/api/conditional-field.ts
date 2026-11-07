/**
 * Conditional Field API — workspace-scoped rules under /api/v1/conditional-fields
 */
import api from './index'

/** Matches backend request.ConditionalFieldResponse */
export interface ConditionalField {
  id: number
  workspace_id: number
  field_id: number
  condition_type: string
  operator: string
  condition_values: string[]
  is_enabled: boolean
  priority: number
  description: string
  created_at: string
  updated_at: string
}

/** Body for POST /conditional-fields (workspace_id comes from query). */
export interface ConditionalFieldCreate {
  field_id: number
  condition_type: string
  operator: string
  condition_values?: string[]
  is_enabled?: boolean
  priority?: number
  description?: string
}

/** Body for PUT /conditional-fields/:id */
export interface ConditionalFieldUpdate {
  condition_type?: string
  operator?: string
  condition_values?: string[]
  is_enabled?: boolean
  priority?: number
  description?: string
}

/** Body for POST /conditional-fields/evaluate */
export interface FieldVisibilityEvaluateRequest {
  issue_type_id?: number
  state_id: number
  priority: string
  assignee_ids: number[]
  label_ids: number[]
}

/** Response from POST /conditional-fields/evaluate */
export interface FieldVisibilityEvaluateResponse {
  visible_field_ids: number[]
}

function workspaceParams(workspaceId: number): Record<string, string> {
  return { workspace_id: String(workspaceId) }
}

export const conditionalFieldApi = {
  list: async (
    workspaceId: number,
    fieldId?: number
  ): Promise<ConditionalField[]> => {
    const params: Record<string, string> = workspaceParams(workspaceId)
    if (fieldId !== undefined) params.field_id = String(fieldId)
    const res = await api.get<ConditionalField[]>('/conditional-fields', { params })
    return res.data
  },

  get: async (workspaceId: number, id: number): Promise<ConditionalField> => {
    const res = await api.get<ConditionalField>(`/conditional-fields/${id}`, {
      params: workspaceParams(workspaceId)
    })
    return res.data
  },

  create: async (
    workspaceId: number,
    data: ConditionalFieldCreate
  ): Promise<ConditionalField> => {
    const res = await api.post<ConditionalField>('/conditional-fields', data, {
      params: workspaceParams(workspaceId)
    })
    return res.data
  },

  update: async (
    workspaceId: number,
    id: number,
    data: ConditionalFieldUpdate
  ): Promise<ConditionalField> => {
    const res = await api.put<ConditionalField>(`/conditional-fields/${id}`, data, {
      params: workspaceParams(workspaceId)
    })
    return res.data
  },

  remove: async (workspaceId: number, id: number): Promise<void> => {
    await api.delete(`/conditional-fields/${id}`, {
      params: workspaceParams(workspaceId)
    })
  },

  evaluateFieldVisibility: async (
    workspaceId: number,
    body: FieldVisibilityEvaluateRequest
  ): Promise<FieldVisibilityEvaluateResponse> => {
    const res = await api.post<FieldVisibilityEvaluateResponse>(
      '/conditional-fields/evaluate',
      body,
      { params: workspaceParams(workspaceId) }
    )
    return res.data
  }
}
