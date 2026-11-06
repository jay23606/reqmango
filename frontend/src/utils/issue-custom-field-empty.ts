/**
 * Whether an issue custom field value should be treated as "unset" for required validation
 * (aligned with SRS: empty string, null, missing; boolean false and number 0 count as set).
 */
export function isEmptyIssueCustomFieldValue(fieldType: string, raw: string | null | undefined): boolean {
  if (fieldType === 'boolean') return false

  if (raw === null || raw === undefined) return true

  if (fieldType === 'member') {
    const trimmed = raw.trim()
    if (trimmed === '') return true
    try {
      const parsed = JSON.parse(trimmed)
      if (!Array.isArray(parsed) || parsed.length === 0) return true
      const first = parsed[0]
      return first === null || first === undefined || first === ''
    } catch {
      return true
    }
  }

  if (fieldType === 'number') {
    const s = raw.trim()
    return s === ''
  }

  // text, url, date, dropdown
  return raw.trim() === ''
}
