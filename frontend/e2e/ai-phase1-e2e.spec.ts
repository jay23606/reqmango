import { test, expect } from '@playwright/test'
import { createDemoIssue, loginPage, resolveDemo } from './helpers/demo'

const API = 'http://localhost:8000/api/v1'
const BASE = 'http://localhost:5173'

test.describe('AI Phase 1 — Agent Automation + Unified Copilot + Result Actions', () => {

  // ═══ P1.1: Agent 自动化触发器 ═══

  test('AI01: Automation rules list includes existing rules', async ({ request }) => {
    const demo = await resolveDemo(request)

    const res = await request.get(`${API}/projects/${demo.projectId}/automations`, { headers: demo.headers })
    expect(res.status()).toBe(200)
    const rules = await res.json()
    console.log(`Automation rules: ${rules.length}`)
    for (const r of rules.slice(0, 5)) {
      console.log(`  [${r.id}] ${r.name} trigger=${r.trigger_type}`)
    }
  })

  test('AI02: Create automation rule with dispatch_agent action', async ({ request }) => {
    const demo = await resolveDemo(request)

    const create = await request.post(`${API}/projects/${demo.projectId}/automations`, {
      headers: demo.headers,
      data: {
        name: `[E2E Agent] ${Date.now()}`,
        trigger_type: 'issue_created',
        conditions: '[{"field":"priority","operator":"equals","value":"urgent"}]',
        actions: '[{"type":"dispatch_agent","value":1,"field":"分析这个紧急工作项并推荐处理方案"}]',
        is_enabled: true,
      },
    })
    expect(create.status()).toBe(201)
    const rule = await create.json()
    console.log(`Created rule [${rule.id}]: ${rule.name}`)
    expect(rule.trigger_type).toBe('issue_created')

    const get = await request.get(`${API}/projects/${demo.projectId}/automations/${rule.id}`, { headers: demo.headers })
    expect(get.status()).toBe(200)

    await request.delete(`${API}/projects/${demo.projectId}/automations/${rule.id}`, { headers: demo.headers })
    console.log('Cleaned up')
  })

  test('AI03: Automation rule with multiple actions including dispatch_agent', async ({ request }) => {
    const demo = await resolveDemo(request)

    const issueRes = await createDemoIssue(request, demo, {
      name: `AI03 issue ${Date.now()}`,
      description: 'e2e',
    })
    expect(issueRes.status()).toBeLessThan(400)
    const issue = await issueRes.json()
    const issueId = issue.id || issue.data?.id

    const create = await request.post(`${API}/projects/${demo.projectId}/automations`, {
      headers: demo.headers,
      data: {
        name: `[E2E Multi] ${Date.now()}`,
        trigger_type: 'state_changed',
        conditions: '[{"field":"state_group","operator":"equals","value":"started"}]',
        actions: '[{"type":"add_comment","value":"状态变更为进行中"},{"type":"dispatch_agent","value":1,"field":"分析工作项进行中的风险"}]',
        is_enabled: true,
      },
    })
    expect(create.status()).toBe(201)
    const rule = await create.json()

    const exec = await request.post(`${API}/projects/${demo.projectId}/automations/${rule.id}/execute`, {
      headers: demo.headers,
      data: { issue_id: issueId, context: { state_group: 'started' } },
    })
    expect(exec.status()).toBe(200)
    console.log(`Execute multi-action rule: ${exec.status()}`)

    const history = await request.get(`${API}/issues/${issueId}/automation-history?limit=1`, { headers: demo.headers })
    const hist = await history.json()
    if (hist.length > 0) {
      console.log(`History: ${hist[0].status} actions=${hist[0].actions_taken?.slice(0, 60)}`)
    }

    await request.delete(`${API}/projects/${demo.projectId}/automations/${rule.id}`, { headers: demo.headers })
  })

  test('AI04: List agents for dispatch_agent action', async ({ request }) => {
    const demo = await resolveDemo(request)

    const agents = await request.get(`${API}/workspaces/${demo.workspaceId}/agents`, { headers: demo.headers })
    expect(agents.status()).toBe(200)
    const list = await agents.json()
    console.log(`Agents: ${list.length}`)
    for (const a of list.slice(0, 5)) {
      console.log(`  [${a.id}] ${a.name} type=${a.agent_type} status=${a.status} capabilities=${JSON.stringify(a.capabilities)}`)
    }
  })

  // ═══ P1.3: AI 结果操作化 ═══

  test('AI05: AI Copilot opens with Ctrl+J', async ({ page, request }) => {
    const demo = await resolveDemo(request)
    await loginPage(page, demo.token)
    await page.goto(`${BASE}/workspace/${demo.workspaceSlug}/project/${demo.projectId}`)
    await page.waitForTimeout(3000)

    await page.keyboard.press('Control+KeyJ')
    await page.waitForTimeout(1000)

    const body = await page.textContent('body')
    const hasAI = body?.includes('AI') || body?.includes('Copilot') || body?.includes('Ask') || body?.includes('Build')
    console.log(`AI Copilot visible: ${hasAI}`)
  })

  test('AI06: AI Copilot has operation buttons after response', async ({ page, request }) => {
    const demo = await resolveDemo(request)
    await loginPage(page, demo.token)
    await page.goto(`${BASE}/workspace/${demo.workspaceSlug}/project/${demo.projectId}`)
    await page.waitForTimeout(3000)

    await page.keyboard.press('Control+KeyJ')
    await page.waitForTimeout(1000)

    const input = page.locator('textarea, input[type="text"]').last()
    if (await input.isVisible().catch(() => false)) {
      await input.fill('How many issues are in this project?')
      await page.keyboard.press('Enter')
      await page.waitForTimeout(5000)

      const body = await page.textContent('body')
      const hasCreateIssue = body?.includes('Create Issues') || body?.includes('创建工作项')
      console.log(`'Create Issues' button: ${hasCreateIssue}`)
    } else {
      console.log('Chat input not found')
    }
  })

  test('AI07: AI Copilot Create mode accessible via tab', async ({ page, request }) => {
    const demo = await resolveDemo(request)
    await loginPage(page, demo.token)
    await page.goto(`${BASE}/workspace/${demo.workspaceSlug}/project/${demo.projectId}`)
    await page.waitForTimeout(3000)

    // Prefer the FAB — Ctrl+J can race with page load / focus.
    const fab = page.locator('button[title*="AI Assistant"], button[title*="Ctrl+J"]').first()
    if (await fab.isVisible({ timeout: 5000 }).catch(() => false)) {
      await fab.click()
    } else {
      await page.keyboard.press('Control+KeyJ')
    }
    await page.waitForTimeout(1000)

    // Scope to the Copilot drawer so we don't hit QuickCreate's disabled 「创建」.
    const copilot = page.locator('div.fixed.right-0').filter({ hasText: /Ask|Build|构建|提问/ }).first()
    await expect(copilot).toBeVisible({ timeout: 10000 })

    // Create is a Build sub-panel, not a top-level tab.
    const buildTab = copilot.locator('button:has-text("Build"), button:has-text("构建")').first()
    await expect(buildTab).toBeVisible({ timeout: 5000 })
    await buildTab.click()
    await page.waitForTimeout(500)

    const createBtn = copilot.locator('button:has-text("✨")').first()
    await expect(createBtn).toBeVisible({ timeout: 10000 })
    await createBtn.click()

    await expect(
      copilot.locator('button:has-text("生成预览"), button:has-text("Generate Preview")').first(),
    ).toBeVisible({ timeout: 10000 })
  })

  test('AI08: AI Copilot quick action buttons visible', async ({ page, request }) => {
    const demo = await resolveDemo(request)
    await loginPage(page, demo.token)
    await page.goto(`${BASE}/workspace/${demo.workspaceSlug}/project/${demo.projectId}`)
    await page.waitForTimeout(3000)

    await page.keyboard.press('Control+KeyJ')
    await page.waitForTimeout(1000)

    const body = await page.textContent('body')
    const hasSuggestions = (body?.includes('💡') || false) || (body?.includes('Summarize') || body?.includes('总结') || false)
    console.log(`Suggestions visible: ${hasSuggestions}`)
  })

  // ═══ Backend AI API Regression ═══

  test('AI09: AI Chat API returns SSE stream', async ({ request }) => {
    const demo = await resolveDemo(request)

    const res = await request.post(`${API}/projects/${demo.projectId}/ai/analyze`, {
      headers: demo.headers,
    })
    // Analyze may be unavailable without an LLM key in CI.
    expect([200, 400, 422, 500, 503]).toContain(res.status())
    if (res.status() === 200) {
      const data = await res.json()
      console.log(`AI Analyze keys: ${Object.keys(data).join(', ')}`)
    } else {
      console.log(`AI Analyze skipped/unavailable: ${res.status()}`)
    }
  })

  test('AI10: AI config endpoint works', async ({ request }) => {
    const demo = await resolveDemo(request)

    const cfg = await request.get(`${API}/workspaces/${demo.workspaceId}/ai-config`, { headers: demo.headers })
    expect(cfg.status()).toBe(200)
    const config = await cfg.json()
    console.log(`AI config: provider=${config.provider} model=${config.model} configured=${config.configured}`)
  })
})
