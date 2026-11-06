import { describe, it, expect } from 'vitest'
import { isEmptyIssueCustomFieldValue } from '@/utils/issue-custom-field-empty'

describe('isEmptyIssueCustomFieldValue', () => {
  it('treats boolean as never empty (false is valid)', () => {
    expect(isEmptyIssueCustomFieldValue('boolean', '')).toBe(false)
    expect(isEmptyIssueCustomFieldValue('boolean', 'false')).toBe(false)
  })

  it('text/url/date/dropdown: trim and empty string', () => {
    expect(isEmptyIssueCustomFieldValue('text', '')).toBe(true)
    expect(isEmptyIssueCustomFieldValue('text', '   ')).toBe(true)
    expect(isEmptyIssueCustomFieldValue('text', ' x ')).toBe(false)
    expect(isEmptyIssueCustomFieldValue('url', '')).toBe(true)
    expect(isEmptyIssueCustomFieldValue('date', '')).toBe(true)
    expect(isEmptyIssueCustomFieldValue('dropdown', '')).toBe(true)
  })

  it('number: empty or whitespace only', () => {
    expect(isEmptyIssueCustomFieldValue('number', '')).toBe(true)
    expect(isEmptyIssueCustomFieldValue('number', '0')).toBe(false)
    expect(isEmptyIssueCustomFieldValue('number', '  ')).toBe(true)
  })

  it('member: empty string or empty array', () => {
    expect(isEmptyIssueCustomFieldValue('member', '')).toBe(true)
    expect(isEmptyIssueCustomFieldValue('member', '[]')).toBe(true)
    expect(isEmptyIssueCustomFieldValue('member', '[1]')).toBe(false)
  })
})
