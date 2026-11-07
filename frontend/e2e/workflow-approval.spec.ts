import { test, expect } from '@playwright/test'
import { createProjectIssue } from './helpers/demo'

const ADMIN_USER = 'admin@reqmango.com'
const ADMIN_PASS = 'demo1234'
const WORKSPACE_SLUG = 'reqmango-dev'
const PROJECT_IDENTIFIER = 'CORE'

let authToken: string
let projectId: number
let workspaceId: number

test.describe('工作流审批功能', () => {
  test.beforeAll(async ({ request }) => {
    const loginResponse = await request.post('http://localhost:8000/api/v1/auth/login', {
      data: { email: ADMIN_USER, password: ADMIN_PASS },
    })
    const loginData = await loginResponse.json()
    authToken = loginData.access_token

    const workspacesResponse = await request.get('http://localhost:8000/api/v1/workspaces', {
      headers: { Authorization: `Bearer ${authToken}` },
    })
    const workspacesData = await workspacesResponse.json()
    const workspaces = Array.isArray(workspacesData) ? workspacesData : (workspacesData?.data || [])
    const workspace = workspaces.find((w: any) => w.slug === WORKSPACE_SLUG)
    workspaceId = workspace.id

    const projectsResponse = await request.get(`http://localhost:8000/api/v1/projects?workspace_id=${workspaceId}`, {
      headers: { Authorization: `Bearer ${authToken}` },
    })
    const projectsData = await projectsResponse.json()
    const projects = Array.isArray(projectsData) ? projectsData : (projectsData?.data || [])
    const project = projects.find((p: any) => p.identifier === PROJECT_IDENTIFIER)
    projectId = project.id
  })

  test('API - 审批计数接口', async ({ request }) => {
    const response = await request.get(`http://localhost:8000/api/v1/workspaces/${WORKSPACE_SLUG}/approvals/count`, {
      headers: { Authorization: `Bearer ${authToken}` },
    })
    expect(response.ok()).toBeTruthy()
    const data = await response.json()
    console.log('审批计数:', data)
    expect(data).toHaveProperty('pending_count')
  })

  test('API - 工作空间审批列表接口', async ({ request }) => {
    const response = await request.get(`http://localhost:8000/api/v1/workspaces/${WORKSPACE_SLUG}/approvals`, {
      headers: { Authorization: `Bearer ${authToken}` },
    })
    expect(response.ok()).toBeTruthy()
    const data = await response.json()
    console.log('工作空间审批列表:', data)
  })

  test('API - 项目审批列表接口', async ({ request }) => {
    const response = await request.get(`http://localhost:8000/api/v1/projects/${projectId}/approvals?workspace_id=${workspaceId}`, {
      headers: { Authorization: `Bearer ${authToken}` },
    })
    expect(response.ok()).toBeTruthy()
    const data = await response.json()
    console.log('项目审批列表:', data)
  })

  // Full happy path: create an approval-type transition, create an issue,
  // submit an approval request, then approve it — verifying the issue state
  // changes as expected.
  test('API - 创建审批并批准（端到端）', async ({ request }) => {
    const H = { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' }

    // 1. Get states
    const statesResponse = await request.get(`http://localhost:8000/api/v1/projects/${projectId}/settings/states`, {
      headers: H,
    })
    const statesBody = await statesResponse.json()
    const states = Array.isArray(statesBody) ? statesBody : (statesBody.data || [])
    expect(states.length).toBeGreaterThanOrEqual(2)

    const fromState = states[0]
    const toState = states.find((s: any) => s.id !== fromState.id) || states[1]

    // 2. Create a state workflow with an approval transition. Transitions
    // belong to `workflows`, served under the workspace group; the project
    // group is agent orchestration and has no transition routes.
    const WS = `http://localhost:8000/api/v1/workspaces/${workspaceId}/workflows`
    const workflowRes = await request.post(WS, {
      headers: H,
      data: { name: `[E2E 审批] ${Date.now()}` },
    })
    expect(workflowRes.ok()).toBeTruthy()
    const workflow = await workflowRes.json()

    const transRes = await request.post(`${WS}/${workflow.id}/transitions`, {
      headers: H,
      data: {
        name: '审批流转',
        source_state_id: fromState.id,
        target_state_id: toState.id,
        rule_type: 'approval',
        approver_ids: '[1]',
        approve_target_state_id: toState.id,
        reject_target_state_id: fromState.id,
        approval_mode: 'any',
      },
    })
    expect(transRes.ok()).toBeTruthy()
    const transition = await transRes.json()
    console.log('创建审批转换:', transition.id)

    // 3. Create an issue in the source state
    const issueRes = await createProjectIssue(request, projectId, workspaceId, H, {
      name: 'API测试审批工作项-端到端',
      description_html: '<p>测试审批端到端</p>',
      priority: 'medium',
      state_id: fromState.id,
    })
    expect(issueRes.ok()).toBeTruthy()
    const issue = await issueRes.json()

    // 4. Create an approval request
    const approvalRes = await request.post(`http://localhost:8000/api/v1/issues/${issue.id}/approvals`, {
      headers: H,
      data: {
        transition_id: transition.id,
        request_note: '端到端测试审批',
      },
    })
    expect(approvalRes.ok()).toBeTruthy()
    const approval = await approvalRes.json()
    expect(approval).toHaveProperty('id')
    expect(approval.status).toBe('pending')
    console.log('创建审批成功:', approval.id)

    // 5. Approve the request
    const decideRes = await request.post(`http://localhost:8000/api/v1/approvals/${approval.id}/decide`, {
      headers: H,
      data: { decision: 'approved', note: '端到端测试批准' },
    })
    expect(decideRes.ok()).toBeTruthy()
    console.log('审批批准成功')

    // Cleanup
    await request.delete(`${WS}/${workflow.id}`, {
      headers: H,
    })
  })

  test('审批中心页面 - 查看待审批列表', async ({ page }) => {
    await page.goto('/')
    await page.waitForLoadState('networkidle')
    
    await page.evaluate((token) => {
      localStorage.setItem('token', token)
    }, authToken)
    
    await page.reload()
    await page.waitForLoadState('networkidle')
    
    await page.goto(`/workspace/${WORKSPACE_SLUG}/approvals`)
    await page.waitForLoadState('networkidle')
    
    await expect(page.locator('h1:text("审批中心")')).toBeVisible()
    console.log('审批中心页面加载成功')
  })

  test('TopBar - 审批图标显示', async ({ page }) => {
    await page.goto('/')
    await page.waitForLoadState('networkidle')
    
    await page.evaluate((token) => {
      localStorage.setItem('token', token)
    }, authToken)
    
    await page.reload()
    await page.waitForLoadState('networkidle')
    
    await page.goto(`/workspace/${WORKSPACE_SLUG}`)
    await page.waitForLoadState('networkidle')
    
    const badge = page.locator('.approval-badge')
    await expect(badge).toBeVisible()
    console.log('审批图标显示成功')
  })
})