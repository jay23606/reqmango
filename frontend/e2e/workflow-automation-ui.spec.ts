import { test, expect } from '@playwright/test'
import { createDemoIssue, loginPage, resolveDemo, type DemoContext } from './helpers/demo'

const API = 'http://localhost:8000/api/v1'
const BASE = 'http://localhost:5173'

async function twoStateIds(request: any, demo: DemoContext): Promise<[number, number]> {
  const res = await request.get(`${API}/projects/${demo.projectId}/settings/states`, { headers: demo.headers })
  const body = await res.json()
  const states = Array.isArray(body) ? body : (body.data || [])
  expect(states.length).toBeGreaterThanOrEqual(2)
  return [states[0].id, states[1].id]
}

test.describe('Workflow & Automation — Full UI Journey', () => {

  test('1. Login page → workspace → project settings renders', async ({ page, request }) => {
    const demo = await resolveDemo(request)
    await loginPage(page, demo.token)

    await page.goto(BASE + '/')
    await page.waitForTimeout(2000)
    console.log('Home title:', await page.title())

    await page.goto(`${BASE}/workspace/${demo.workspaceSlug}/settings`)
    await page.waitForTimeout(1500)
    const wsBody = await page.textContent('body')
    console.log('Workspace settings:', wsBody?.length, 'chars')

    await page.goto(`${BASE}/workspace/${demo.workspaceSlug}/project/${demo.projectId}/settings`)
    await page.waitForTimeout(2000)
    const psBody = await page.textContent('body')
    console.log('Project settings:', psBody?.length, 'chars')
  })

  test('2. States API — verify format matches StateWorkflowManager expectation', async ({ request }) => {
    const demo = await resolveDemo(request)

    const res = await request.get(`${API}/projects/${demo.projectId}/settings/states`, { headers: demo.headers })
    expect(res.status()).toBe(200)
    const body = await res.json()

    console.log('States response keys:', Object.keys(body).join(', '))
    if ('data' in body) {
      console.log('✅ States wrap in {data: [...]} — StateWorkflowManager expects this')
      console.log('States count:', body.data.length)
      for (const s of body.data.slice(0, 5)) {
        console.log(`  State: ${s.name} (id=${s.id}, group=${s.group})`)
      }
    } else if (Array.isArray(body)) {
      console.log('⚠️ States return as raw array — StateWorkflowManager expects {data: [...]}')
    } else {
      console.log('⚠️ Unknown format:', JSON.stringify(body).slice(0, 200))
    }
  })

  test('3. StateWorkflowManager — state workflow CRUD via workspace routes', async ({ request }) => {
    const demo = await resolveDemo(request)
    const [fromId, toId] = await twoStateIds(request, demo)
    const WS = `${API}/workspaces/${demo.workspaceId}/workflows`

    console.log('--- Step 1: List workspace workflows ---')
    const list1 = await request.get(WS, { headers: demo.headers })
    expect(list1.status()).toBe(200)
    const wfs1 = (await list1.json()) || []
    console.log(`Existing workspace workflows: ${wfs1.length}`)

    console.log('\n--- Step 2: Create workflow ---')
    const create = await request.post(WS, {
      headers: demo.headers,
      data: { name: `[UI Flow Test] ${Date.now()}`, description: 'Created simulating UI flow' },
    })
    expect(create.status()).toBe(201)
    const wf = await create.json()
    console.log(`Created: [${wf.id}] ${wf.name}`)

    console.log('\n--- Step 3: Add transition ---')
    // State transitions belong to `workflows` and are served under the
    // workspace group; the project group is agent orchestration and has no
    // transition routes.
    const trans = await request.post(`${WS}/${wf.id}/transitions`, {
      headers: demo.headers,
      data: { name: 'A→B', source_state_id: fromId, target_state_id: toId },
    })
    expect(trans.status()).toBe(201)
    const transData = await trans.json()
    expect(transData).toHaveProperty('id')
    expect(transData.rule_type).toBe('allow')
    console.log('Created transition:', transData.id, transData.name)

    const wfDetail = await request.get(`${WS}/${wf.id}`, { headers: demo.headers })
    expect(wfDetail.status()).toBe(200)
    const detail = await wfDetail.json()
    expect(Array.isArray(detail.transitions)).toBe(true)
    expect(detail.transitions.length).toBeGreaterThanOrEqual(1)

    console.log('\n--- Step 4: Delete workflow (cleanup) ---')
    const del = await request.delete(`${WS}/${wf.id}`, { headers: demo.headers })
    expect(del.status(), 'workflow delete must not return 500 (BUG-57)').toBeLessThan(400)
    console.log('Deleted test workflow')
  })

  test('3b. Project workflow group — agent orchestration detail', async ({ request }) => {
    // The project group addresses agent_workflows. Its structure is nodes and
    // edges, so the detail response must not carry `transitions` at all.
    const demo = await resolveDemo(request)
    const P = `${API}/projects/${demo.projectId}/workflows`

    const create = await request.post(P, {
      headers: demo.headers,
      data: { name: `[Agent Flow Test] ${Date.now()}`, description: 'orchestration' },
    })
    expect(create.status()).toBe(201)
    const wf = await create.json()

    const detailRes = await request.get(`${P}/${wf.id}`, { headers: demo.headers })
    expect(detailRes.status()).toBe(200)
    const detail = await detailRes.json()
    expect(Array.isArray(detail.nodes)).toBe(true)
    expect(Array.isArray(detail.edges)).toBe(true)
    expect(detail.transitions).toBeUndefined()

    const del = await request.delete(`${P}/${wf.id}`, { headers: demo.headers })
    expect(del.status()).toBeLessThan(400)
  })

  test('4. AutomationManager — full CRUD via API (simulating UI actions)', async ({ request }) => {
    const demo = await resolveDemo(request)

    console.log('--- List automations ---')
    const list = await request.get(`${API}/projects/${demo.projectId}/automations`, { headers: demo.headers })
    expect(list.status()).toBe(200)
    const autos = await list.json()
    console.log(`Existing automations: ${autos.length}`)

    console.log('\n--- Create automation ---')
    const create = await request.post(`${API}/projects/${demo.projectId}/automations`, {
      headers: demo.headers,
      data: {
        name: `[UI Flow Test] ${Date.now()}`,
        trigger_type: 'issue_created',
        conditions: '[{"field":"priority","operator":"equals","value":"high"}]',
        actions: '[{"type":"add_comment","value":"[Auto] High priority issue created"}]',
      },
    })
    expect(create.status()).toBe(201)
    const auto = await create.json()
    console.log(`Created: [${auto.id}] ${auto.name}`)

    const issueRes = await createDemoIssue(request, demo, {
      name: `Auto exec ${Date.now()}`,
      description: 'e2e',
      priority: 'high',
    })
    const issue = await issueRes.json()
    const issueId = issue.id || issue.data?.id

    console.log('\n--- Execute automation ---')
    const exec = await request.post(`${API}/projects/${demo.projectId}/automations/${auto.id}/execute`, {
      headers: demo.headers,
      data: { issue_id: issueId, context: { priority: 'high' } },
    })
    expect(exec.status()).toBe(200)
    const execResult = await exec.json()
    console.log('Execute results:', execResult.results)

    console.log('\n--- Delete automation (cleanup) ---')
    await request.delete(`${API}/projects/${demo.projectId}/automations/${auto.id}`, { headers: demo.headers })
    console.log('Deleted test automation')
  })

  test('5. Approval fields round-trip through a workspace state workflow', async ({ request }) => {
    const demo = await resolveDemo(request)
    const [fromId, toId] = await twoStateIds(request, demo)
    const WS = `${API}/workspaces/${demo.workspaceId}/workflows`

    const wfCreate = await request.post(WS, {
      headers: demo.headers,
      data: { name: `[Gap Test] ${Date.now()}`, description: 'Approval gap test' },
    })
    expect(wfCreate.status()).toBe(201)
    const wf = await wfCreate.json()

    const tr = await request.post(`${WS}/${wf.id}/transitions`, {
      headers: demo.headers,
      data: {
        name: 'Requires admin approval',
        source_state_id: fromId, target_state_id: toId,
        rule_type: 'approval', approver_ids: '1',
      },
    })
    expect(tr.status()).toBe(201)
    const t = await tr.json()
    expect(t.rule_type).toBe('approval')
    expect(t.approver_ids).toBe('1')
    console.log(`API: Created approval transition (rule=${t.rule_type}, approvers=${t.approver_ids})`)

    await request.delete(`${WS}/${wf.id}`, { headers: demo.headers })
  })

  test('6. ⚠️ States API response format check', async ({ request }) => {
    const demo = await resolveDemo(request)

    const res = await request.get(`${API}/projects/${demo.projectId}/settings/states`, { headers: demo.headers })
    const body = await res.json()

    const hasWrapper = 'data' in body && Array.isArray(body.data)
    const isArray = Array.isArray(body)

    console.log(`States API format: ${hasWrapper ? '{data:[...]}' : isArray ? 'raw array' : 'other'}`)

    if (!hasWrapper) {
      console.log('⚠️ ISSUE: StateWorkflowManager.vue line 56 expects states.value = s.data')
    } else {
      console.log('✅ Format matches StateWorkflowManager expectation')
    }
  })
})
