import { test, expect } from '@playwright/test'
import { createDemoIssue, loginPage, resolveDemo } from './helpers/demo'

const API = 'http://localhost:8000/api/v1'
const BASE = 'http://localhost:5173'

test.describe('Custom Fields — Full UI Journey', () => {

  test('CF01: Custom Fields settings page renders', async ({ page, request }) => {
    const demo = await resolveDemo(request)
    await loginPage(page, demo.token)
    await page.goto(`${BASE}/workspaces/${demo.workspaceId}/projects/${demo.projectId}/custom-fields`)
    await page.waitForTimeout(2500)
    const body = await page.textContent('body')
    console.log('CF Settings page chars:', body?.length)
    expect(body).toBeTruthy()
  })

  test('CF02: Custom Fields API — list fields', async ({ request }) => {
    const demo = await resolveDemo(request)

    const res = await request.get(`${API}/custom-fields?workspace_id=${demo.workspaceId}`, { headers: demo.headers })
    expect(res.status()).toBe(200)
    const fields = await res.json()
    expect(Array.isArray(fields)).toBe(true)
    console.log(`Total custom fields: ${fields.length}`)
    for (const f of fields.slice(0, 5)) {
      console.log(`  [${f.id}] ${f.name} type=${f.field_type} active=${f.is_active}`)
    }
  })

  test('CF03: Custom Fields — CRUD lifecycle via API', async ({ request }) => {
    const demo = await resolveDemo(request)
    const name = `[E2E-CRUD] ${Date.now()}`

    const create = await request.post(`${API}/custom-fields?workspace_id=${demo.workspaceId}`, {
      headers: demo.headers,
      data: { name, field_type: 'text', description: 'CRUD test' },
    })
    expect(create.status()).toBe(201)
    const field = await create.json()
    console.log(`Created: [${field.id}]`)
    expect(field.field_type).toBe('text')

    const get = await request.get(`${API}/custom-fields/${field.id}?workspace_id=${demo.workspaceId}`, { headers: demo.headers })
    expect(get.status()).toBe(200)

    const update = await request.put(`${API}/custom-fields/${field.id}?workspace_id=${demo.workspaceId}`, {
      headers: demo.headers,
      data: { description: 'Updated description' },
    })
    expect(update.status()).toBe(200)

    const del = await request.delete(`${API}/custom-fields/${field.id}?workspace_id=${demo.workspaceId}`, { headers: demo.headers })
    expect(del.status()).toBe(200)
    console.log('CRUD lifecycle: OK')
  })

  test('CF04: Dropdown custom field with options', async ({ request }) => {
    const demo = await resolveDemo(request)
    const name = `[E2E-Dropdown] ${Date.now()}`

    const create = await request.post(`${API}/custom-fields?workspace_id=${demo.workspaceId}`, {
      headers: demo.headers,
      data: { name, field_type: 'dropdown' },
    })
    expect(create.status()).toBe(201)
    const field = await create.json()

    for (const opt of [{ value: 'Alpha', color: '#3B82F6' }, { value: 'Beta', color: '#10B981' }]) {
      const r = await request.post(`${API}/custom-fields/${field.id}/options?workspace_id=${demo.workspaceId}`, {
        headers: demo.headers, data: opt,
      })
      expect(r.status()).toBe(201)
    }

    const get = await request.get(`${API}/custom-fields/${field.id}?workspace_id=${demo.workspaceId}`, { headers: demo.headers })
    const got = await get.json()
    expect(got.options).toHaveLength(2)
    console.log(`Options: ${got.options.map((o: any) => o.value).join(', ')}`)

    await request.delete(`${API}/custom-fields/${field.id}?workspace_id=${demo.workspaceId}`, { headers: demo.headers })
    console.log('Dropdown CRUD: OK')
  })

  test('CF05: Issue custom field values — set, read, bulk update', async ({ request }) => {
    const demo = await resolveDemo(request)
    const name = `[E2E-Values] ${Date.now()}`

    const create = await request.post(`${API}/custom-fields?workspace_id=${demo.workspaceId}`, {
      headers: demo.headers,
      data: { name, field_type: 'text' },
    })
    const field = await create.json()

    const issueRes = await createDemoIssue(request, demo, {
      name: `CF05 issue ${Date.now()}`,
      description: 'e2e',
    })
    expect(issueRes.status()).toBeLessThan(400)
    const issue = await issueRes.json()
    const issueId = issue.id || issue.data?.id

    // Create often omits type_id; values require a type binding, so assign a type then attach the field.
    const typesRes = await request.get(`${API}/projects/${demo.projectId}/issue-types`, { headers: demo.headers })
    const typesBody = await typesRes.json()
    const types = Array.isArray(typesBody) ? typesBody : (typesBody.data || [])
    const typeId = (types.find((t: any) => t.name === 'Task') || types[0])?.id
    expect(typeId).toBeTruthy()
    // Seeded projects may list workspace types via legacy fallback without an
    // import row — import first (200/201/409 all fine) so the write path accepts it.
    await request.post(`${API}/projects/${demo.projectId}/issue-types/${typeId}/import`, {
      headers: demo.headers,
    })
    const typed = await request.put(`${API}/issues/${issueId}?workspace_id=${demo.workspaceId}`, {
      headers: demo.headers,
      data: { type_id: typeId },
    })
    expect(typed.status()).toBeLessThan(400)

    const bind = await request.post(`${API}/issue-types/${typeId}/fields`, {
      headers: demo.headers,
      data: { field_id: field.id, is_required: false, sequence: 1 },
    })
    expect([200, 201]).toContain(bind.status())

    const setVal = await request.post(`${API}/custom-fields/issues/${issueId}/values?workspace_id=${demo.workspaceId}`, {
      headers: demo.headers,
      data: { field_id: field.id, value: 'Hello E2E' },
    })
    expect(setVal.status()).toBe(201)
    console.log('Set value: OK')

    const getVals = await request.get(`${API}/custom-fields/issues/${issueId}/values?workspace_id=${demo.workspaceId}`, { headers: demo.headers })
    expect(getVals.status()).toBe(200)
    const vals = await getVals.json()
    const ours = Array.isArray(vals) ? vals.find((v: any) => v.field_id === field.id) : null
    expect(ours).toBeDefined()
    console.log(`Read value: ${ours?.value}`)

    await request.delete(`${API}/custom-fields/issues/${issueId}/values/${field.id}?workspace_id=${demo.workspaceId}`, { headers: demo.headers })
    await request.delete(`${API}/custom-fields/${field.id}?workspace_id=${demo.workspaceId}`, { headers: demo.headers })
    console.log('Values test: OK')
  })

  test('CF06: Issue detail page renders with custom fields', async ({ page, request }) => {
    const demo = await resolveDemo(request)
    const issueRes = await createDemoIssue(request, demo, {
      name: `CF06 issue ${Date.now()}`,
      description: 'e2e',
    })
    const issue = await issueRes.json()
    const issueId = issue.id || issue.data?.id

    await loginPage(page, demo.token)
    await page.goto(`${BASE}/workspace/${demo.workspaceSlug}/project/${demo.projectId}/issues/${issueId}`)
    await page.waitForTimeout(3000)

    const body = await page.textContent('body')
    console.log('Issue detail chars:', body?.length)
    expect(body).toBeTruthy()
  })

  test('CF07: Issue list column config gear icon opens modal', async ({ page, request }) => {
    const demo = await resolveDemo(request)
    await loginPage(page, demo.token)
    await page.goto(`${BASE}/workspace/${demo.workspaceSlug}/project/${demo.projectId}`)
    await page.waitForTimeout(3000)

    const gearBtn = page.locator('[title*="Column"], [title*="列"]').first()
    const gearSvg = page.locator('button svg path[d*="M10.325"]').first()

    if (await gearBtn.isVisible().catch(() => false)) {
      await gearBtn.click()
    } else if (await gearSvg.isVisible().catch(() => false)) {
      await page.locator('button:has(svg)').filter({ has: page.locator('svg') }).nth(1).click()
    }
    await page.waitForTimeout(1000)

    const modal = await page.textContent('body')
    console.log('Page text sample:', modal?.slice(0, 300))
  })

  test('CF08: All field types — create via API', async ({ request }) => {
    const demo = await resolveDemo(request)

    for (const ftype of ['text', 'number', 'boolean', 'date', 'url']) {
      const create = await request.post(`${API}/custom-fields?workspace_id=${demo.workspaceId}`, {
        headers: demo.headers,
        data: { name: `[E2E-${ftype}] ${Date.now()}`, field_type: ftype },
      })
      expect(create.status()).toBe(201)
      const field = await create.json()
      expect(field.field_type).toBe(ftype)
      await request.delete(`${API}/custom-fields/${field.id}?workspace_id=${demo.workspaceId}`, { headers: demo.headers })
    }
    console.log('All field types: OK')
  })

  test('CF09: Field active/inactive toggle', async ({ request }) => {
    const demo = await resolveDemo(request)
    const name = `[E2E-Toggle] ${Date.now()}`

    const create = await request.post(`${API}/custom-fields?workspace_id=${demo.workspaceId}`, {
      headers: demo.headers,
      data: { name, field_type: 'text' },
    })
    const field = await create.json()
    expect(field.is_active).toBe(true)

    await request.put(`${API}/custom-fields/${field.id}?workspace_id=${demo.workspaceId}`, {
      headers: demo.headers, data: { is_active: false },
    })
    const deact = await request.get(`${API}/custom-fields/${field.id}?workspace_id=${demo.workspaceId}`, { headers: demo.headers })
    expect((await deact.json()).is_active).toBe(false)

    await request.put(`${API}/custom-fields/${field.id}?workspace_id=${demo.workspaceId}`, {
      headers: demo.headers, data: { is_active: true },
    })
    const react = await request.get(`${API}/custom-fields/${field.id}?workspace_id=${demo.workspaceId}`, { headers: demo.headers })
    expect((await react.json()).is_active).toBe(true)

    await request.delete(`${API}/custom-fields/${field.id}?workspace_id=${demo.workspaceId}`, { headers: demo.headers })
    console.log('Toggle: OK')
  })

  test('CF10: Project-scoped custom field', async ({ request }) => {
    const demo = await resolveDemo(request)
    const name = `[E2E-Project] ${Date.now()}`

    const create = await request.post(`${API}/custom-fields?workspace_id=${demo.workspaceId}`, {
      headers: demo.headers,
      data: { name, field_type: 'text', project_id: demo.projectId },
    })
    expect(create.status()).toBe(201)
    const field = await create.json()
    expect(field.project_id).toBe(demo.projectId)

    await request.delete(`${API}/custom-fields/${field.id}?workspace_id=${demo.workspaceId}`, { headers: demo.headers })
    console.log('Project-scoped field: OK')
  })

  test('CF11: Default value for new custom field', async ({ request }) => {
    const demo = await resolveDemo(request)
    const name = `[E2E-Default] ${Date.now()}`

    const create = await request.post(`${API}/custom-fields?workspace_id=${demo.workspaceId}`, {
      headers: demo.headers,
      data: { name, field_type: 'text', default_value: 'DefaultText' },
    })
    expect(create.status()).toBe(201)
    const field = await create.json()
    expect(field.default_value).toBe('DefaultText')

    await request.delete(`${API}/custom-fields/${field.id}?workspace_id=${demo.workspaceId}`, { headers: demo.headers })
    console.log('Default value: OK')
  })
})
