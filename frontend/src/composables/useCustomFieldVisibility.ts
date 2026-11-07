/**
 * Debounced POST /conditional-fields/evaluate for create/detail UIs.
 * visibleFieldIds: null = not evaluated or error — show all linked fields (safe default).
 */
import { ref, onBeforeUnmount, type Ref } from 'vue'
import {
  conditionalFieldApi,
  type FieldVisibilityEvaluateRequest,
} from '@/api/conditional-field'

export function useCustomFieldVisibility(workspaceId: Ref<number>, debounceMs = 250) {
  const visibleFieldIds = ref<Set<number> | null>(null)
  let timer: ReturnType<typeof setTimeout> | null = null

  async function evaluateWithBody(body: FieldVisibilityEvaluateRequest | null) {
    if (!workspaceId.value) {
      visibleFieldIds.value = null
      return
    }
    if (!body) {
      visibleFieldIds.value = null
      return
    }
    try {
      const res = await conditionalFieldApi.evaluateFieldVisibility(workspaceId.value, body)
      visibleFieldIds.value = new Set(res.visible_field_ids)
    } catch (e) {
      console.error('Field visibility evaluation failed:', e)
      visibleFieldIds.value = null
    }
  }

  function scheduleEvaluate(getBody: () => FieldVisibilityEvaluateRequest | null) {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => {
      void evaluateWithBody(getBody())
    }, debounceMs)
  }

  async function evaluateNow(getBody: () => FieldVisibilityEvaluateRequest | null) {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
    await evaluateWithBody(getBody())
  }

  onBeforeUnmount(() => {
    if (timer) clearTimeout(timer)
  })

  return { visibleFieldIds, scheduleEvaluate, evaluateNow }
}
