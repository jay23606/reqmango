import type { APIRequestContext, APIResponse } from '@playwright/test'

const API = 'http://localhost:8000/api/v1'
const ADMIN = { email: 'admin@reqmango.com', password: 'demo1234' }

export type DemoContext = {
  token: string
  headers: Record<string, string>
  workspaceId: number
  workspaceSlug: string
  projectId: number
  projectIdentifier: string
}

type IssueCreateInput = Record<string, unknown>

/** Resolve the seeded reqmango-dev / CORE project (IDs vary across DBs). */
export async function resolveDemo(request: APIRequestContext): Promise<DemoContext> {
  const login = await request.post(`${API}/auth/login`, { data: ADMIN })
  if (!login.ok()) {
    throw new Error(`login failed: ${login.status()} ${await login.text()}`)
  }
  const { access_token: token } = await login.json()
  const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }

  const wsRes = await request.get(`${API}/workspaces`, { headers })
  const wsBody = await wsRes.json()
  const workspaces = Array.isArray(wsBody) ? wsBody : (wsBody.data || [])
  const workspace = workspaces.find((w: any) => w.slug === 'reqmango-dev') || workspaces[0]
  if (!workspace?.id) throw new Error('no workspace found for e2e')

  const projRes = await request.get(`${API}/projects?workspace_id=${workspace.id}`, { headers })
  const projBody = await projRes.json()
  const projects = Array.isArray(projBody) ? projBody : (projBody.data || [])
  const project = projects.find((p: any) => p.identifier === 'CORE') || projects[0]
  if (!project?.id) throw new Error('no project found for e2e')

  return {
    token,
    headers,
    workspaceId: workspace.id,
    workspaceSlug: workspace.slug,
    projectId: project.id,
    projectIdentifier: project.identifier || 'CORE',
  }
}

export async function loginPage(page: any, token: string) {
  await page.goto('http://localhost:5173/login')
  await page.evaluate((t: string) => localStorage.setItem('token', t), token)
}

async function resolveProjectDefaultTypeId(
  request: APIRequestContext,
  headers: Record<string, string>,
  projectId: number,
  explicitTypeId?: number,
): Promise<number | undefined> {
  if (explicitTypeId && explicitTypeId > 0) return explicitTypeId
  const typesRes = await request.get(`${API}/projects/${projectId}/issue-types`, { headers })
  if (!typesRes.ok()) return undefined
  const typesBody = await typesRes.json()
  const types = Array.isArray(typesBody) ? typesBody : (typesBody.data || [])
  const picked =
    types.find((t: any) => t.is_default) ||
    types.find((t: any) => t.name === 'Feature') ||
    types.find((t: any) => t.name === 'Epic') ||
    types[0]
  return picked?.id
}

async function requiredCustomFieldValues(
  request: APIRequestContext,
  headers: Record<string, string>,
  typeId: number,
): Promise<Record<number, string | number>> {
  const fieldsRes = await request.get(`${API}/issue-types/${typeId}/fields`, { headers })
  if (!fieldsRes.ok()) return {}
  const fieldsBody = await fieldsRes.json()
  const fields = Array.isArray(fieldsBody) ? fieldsBody : (fieldsBody.data || [])
  const cfValues: Record<number, string | number> = {}
  for (const f of fields) {
    if (!f.is_required) continue
    const fieldId = f.field_id ?? f.fieldId
    const fieldType = f.field_type ?? f.fieldType
    if (fieldType === 'dropdown' && f.options?.length) {
      cfValues[fieldId] = f.options[0].value
    } else if (fieldType === 'number') {
      cfValues[fieldId] = '1'
    } else if (fieldType === 'boolean') {
      cfValues[fieldId] = 'true'
    } else if (fieldType === 'date') {
      cfValues[fieldId] = '2026-01-01'
    } else {
      cfValues[fieldId] = 'e2e'
    }
  }
  return cfValues
}

/**
 * Merges default type + mandatory custom-field values for seeded/demo workspaces.
 * Skips type defaults when parent_id is set (sub-issues may omit type per SRS).
 */
export async function buildIssueCreateBody(
  request: APIRequestContext,
  headers: Record<string, string>,
  projectId: number,
  data: IssueCreateInput = {},
): Promise<IssueCreateInput> {
  const hasParent = data.parent_id != null && Number(data.parent_id) > 0
  if (hasParent) return { ...data }

  const explicitTypeId =
    typeof data.type_id === 'number' ? data.type_id : data.type_id != null ? Number(data.type_id) : undefined
  const typeId = await resolveProjectDefaultTypeId(request, headers, projectId, explicitTypeId)
  if (!typeId) return { ...data }

  const existing = (data.custom_field_values as Record<number, unknown> | undefined) ?? {}
  const required = await requiredCustomFieldValues(request, headers, typeId)
  return {
    ...data,
    type_id: explicitTypeId ?? typeId,
    custom_field_values: { ...required, ...existing },
  }
}

export async function createDemoIssue(
  request: APIRequestContext,
  demo: DemoContext,
  data: IssueCreateInput = {},
): Promise<APIResponse> {
  const body = await buildIssueCreateBody(request, demo.headers, demo.projectId, data)
  return request.post(
    `${API}/issues?project_id=${demo.projectId}&workspace_id=${demo.workspaceId}`,
    { headers: demo.headers, data: body },
  )
}

export async function createProjectIssue(
  request: APIRequestContext,
  projectId: number,
  workspaceId: number,
  headers: Record<string, string>,
  data: IssueCreateInput = {},
): Promise<APIResponse> {
  const body = await buildIssueCreateBody(request, headers, projectId, data)
  return request.post(`${API}/issues?project_id=${projectId}&workspace_id=${workspaceId}`, { headers, data: body })
}
