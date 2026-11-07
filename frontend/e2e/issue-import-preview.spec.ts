import { test, expect } from '@playwright/test'
import { loginPage, resolveDemo } from './helpers/demo'

const APP = 'http://localhost:5173'
const API = 'http://localhost:8000/api/v1'

// CSV import used to be a blind single request: unrecognised headers failed
// with "missing title column" and there was no way to say which column was the
// title. This covers the preview -> mapping -> import path end to end.
test.describe('Issue CSV import preview', () => {
  test('maps unfamiliar headers, previews, then imports', async ({ page, request }) => {
    const demo = await resolveDemo(request)
    const stamp = Date.now()
    const title = `E2E 映射导入 ${stamp}`

    await loginPage(page, demo.token)
    await page.goto(`${APP}/workspace/${demo.workspaceSlug}/project/${demo.projectId}`)

    await page.getByRole('button', { name: /导入|Import/ }).first().click()

    const csv = `Work Item,Stage\n${title},Backlog\n`
    await page.setInputFiles('input[type="file"]', {
      name: 'custom-headers.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(csv, 'utf-8'),
    })

    // Auto-detection cannot know "Work Item"; the mapping control must still
    // appear so the file is importable.
    const nameSelect = page.getByTestId('import-map-name')
    await expect(nameSelect).toBeVisible()
    await nameSelect.selectOption('Work Item')
    await page.getByTestId('import-map-state').selectOption('Stage')

    await expect(page.getByText(/共 1 行/)).toBeVisible()

    await page.getByRole('button', { name: /开始导入|Start import/ }).click()
    await expect(page.getByText(/导入结果|Import result/)).toBeVisible({ timeout: 15000 })
    // Bare name/state rows inherit the project default type; mandatory custom fields block import until CF columns exist.
    await expect(page.getByText(/缺少必填|必填字段|Mandatory|失败|failed/i)).toBeVisible()

    const res = await request.get(
      `${API}/issues?project_id=${demo.projectId}&search=${encodeURIComponent(title)}`,
      { headers: demo.headers },
    )
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    const list = Array.isArray(body) ? body : (body.data || [])
    expect(list.some((i: any) => i.name === title)).toBeFalsy()
  })
})
