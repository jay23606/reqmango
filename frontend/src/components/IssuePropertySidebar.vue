<template>
  <div class="bg-white rounded-lg border border-gray-200 w-[280px] shrink-0 overflow-hidden">
    <div class="px-3 py-2.5 border-b border-gray-100">
      <h3 class="text-xs font-semibold text-gray-600 uppercase tracking-wide">{{ t('issue.properties') }}</h3>
    </div>

    <div class="px-3 py-2 space-y-0">
      <div v-if="isLocked" class="mb-2 px-2.5 py-2 bg-amber-50 border border-amber-200 rounded text-xs text-amber-800">
        <div class="font-medium">{{ t('approvals.pending') }}</div>
        <div class="text-amber-700/80 mt-0.5">{{ t('approvals.pendingLockHint') }}</div>
      </div>

      <!-- Relations summary -->
      <div v-if="relationSummary && relationSummary.total > 0" class="pb-2 mb-1 border-b border-gray-100">
        <div class="flex items-center gap-2 mb-1.5">
          <span class="text-xs font-medium text-gray-600">{{ t('issue.relations') }} ({{ relationSummary.total }})</span>
        </div>
        <div class="space-y-0.5">
          <template v-for="(counts, typeName) in relationSummary.byType" :key="typeName">
            <div v-if="counts.outbound > 0 || counts.inbound > 0" class="flex items-center justify-between text-[11px]">
              <span class="text-gray-500 truncate">{{ typeName }}</span>
              <span class="text-gray-600 shrink-0">
                <span v-if="counts.outbound > 0" class="text-blue-500">→{{ counts.outbound }}</span>
                <span v-if="counts.outbound > 0 && counts.inbound > 0" class="text-gray-300">·</span>
                <span v-if="counts.inbound > 0" class="text-amber-500">←{{ counts.inbound }}</span>
              </span>
            </div>
          </template>
        </div>
      </div>

      <!-- Status & people -->
      <p class="pt-1 pb-1 text-[10px] font-semibold text-gray-400 uppercase tracking-wide">{{ t('issue.state') }} / {{ t('issue.assignee') }}</p>

      <div class="prop-row">
        <label class="prop-label">{{ t('issue.state') }}</label>
        <div class="prop-control">
          <div class="flex items-center gap-1.5 w-full">
            <PropertyPicker
              class="flex-1"
              data-testid="picker-state"
              :model-value="issue.state_id"
              :options="stateOptions"
              :disabled="isLocked"
              @update:model-value="(v) => v !== null && emit('update:state', Number(v))"
            />
            <span v-if="isLocked" class="px-1.5 py-0.5 bg-amber-100 text-amber-700 rounded text-[10px] font-medium whitespace-nowrap">{{ t('approvals.pending') }}</span>
          </div>
        </div>
      </div>

      <div class="prop-row">
        <label class="prop-label">{{ t('issue.priority') }}</label>
        <div class="prop-control">
          <PropertyPicker
            data-testid="picker-priority"
            :model-value="issue.priority"
            :options="priorityOptions"
            :disabled="isLocked"
            @update:model-value="(v) => v !== null && emit('update:priority', String(v))"
          />
        </div>
      </div>

      <div class="prop-row">
        <label class="prop-label">{{ t('issue.assignee') }}</label>
        <div class="prop-control">
          <PropertyPicker
            data-testid="picker-assignee"
            :model-value="issue.assignees?.[0]?.id ?? null"
            :options="memberOptions"
            :disabled="isLocked"
            :placeholder="t('issue.unassigned')"
            :clear-label="t('issue.unassigned')"
            clearable
            @update:model-value="(v) => emit('update:assignee', toId(v))"
          />
        </div>
      </div>

      <!-- Planning -->
      <p class="pt-2 pb-1 text-[10px] font-semibold text-gray-400 uppercase tracking-wide border-t border-gray-100 mt-1">{{ t('issue.cycle') }}</p>

      <div class="prop-row">
        <label class="prop-label">{{ t('issue.cycle') }}</label>
        <div class="prop-control">
          <PropertyPicker
            data-testid="picker-cycle"
            :model-value="issue.cycle_id ?? null"
            :options="cycleOptions"
            :disabled="isLocked"
            clearable
            @update:model-value="(v) => emit('update:cycle', toId(v))"
          />
        </div>
      </div>

      <div class="prop-row">
        <label class="prop-label">{{ t('issue.module') }}</label>
        <div class="prop-control">
          <PropertyPicker
            data-testid="picker-module"
            :model-value="issue.module_ids?.[0] ?? null"
            :options="moduleOptions"
            :disabled="isLocked"
            clearable
            @update:model-value="(v) => emit('update:module', toId(v))"
          />
        </div>
      </div>

      <div class="prop-row">
        <label class="prop-label">{{ t('issue.release') }}</label>
        <div class="prop-control">
          <PropertyPicker
            data-testid="picker-release"
            :model-value="issue.release_id ?? null"
            :options="releaseOptions"
            :disabled="isLocked"
            clearable
            @update:model-value="(v) => emit('update:release', toId(v))"
          />
        </div>
      </div>

      <!-- Dates & labels -->
      <p class="pt-2 pb-1 text-[10px] font-semibold text-gray-400 uppercase tracking-wide border-t border-gray-100 mt-1">{{ t('issue.startDate') }}</p>

      <div class="prop-row">
        <label class="prop-label">{{ t('issue.startDate') }}</label>
        <div class="prop-control">
          <input type="date" class="prop-input" :value="issue.start_date?.split('T')[0] ?? ''" :disabled="isLocked" @input="emitStartDateUpdate" />
        </div>
      </div>

      <div class="prop-row">
        <label class="prop-label">{{ t('issue.targetDate') }}</label>
        <div class="prop-control">
          <input type="date" class="prop-input" :value="issue.target_date?.split('T')[0] ?? ''" :disabled="isLocked" @input="emitTargetDateUpdate" />
        </div>
      </div>

      <div class="prop-row items-start py-2" :class="{ 'pointer-events-none opacity-60': isLocked }">
        <label class="prop-label pt-1">{{ t('issue.labels') }}</label>
        <div class="prop-control">
          <LabelSelector
            :labels="labels"
            :model-value="issue.labels || issue.label_ids || []"
            @change="(ids: number[]) => $emit('update:labels', ids)"
          />
        </div>
      </div>

      <!-- Agent -->
      <div class="pt-2 mt-1 border-t border-gray-100" :class="{ 'pointer-events-none opacity-60': isLocked }">
        <div class="prop-row items-start">
          <label class="prop-label pt-1">{{ t('agent.title') }}</label>
          <div class="prop-control space-y-1.5">
            <AgentSelector v-model="localAgentId" :workspace-id="workspaceId" />
            <div v-if="agentStatus?.agent_id" class="flex items-center gap-1.5 text-xs text-gray-500">
              <span class="font-medium text-violet-700">{{ agentStatus.agent_name }}</span>
              <span v-if="agentStatus.task_status" class="px-1.5 py-0.5 rounded bg-gray-100 text-gray-600">{{ agentStatus.task_status }}</span>
            </div>
            <button
              type="button"
              @click="emitAssign"
              :disabled="!hasAgentSelected || agentAssigning"
              class="w-full px-2 py-1 text-xs font-medium rounded-md bg-violet-500 hover:bg-violet-600 text-white disabled:opacity-50 transition-colors"
            >
              {{ agentAssigning ? t('agent.assigning') : t('agent.assignAgent') }}
            </button>
            <button
              v-if="hasAgentSelected"
              type="button"
              @click="$emit('dispatch-agent', localAgentId)"
              :disabled="agentDispatching"
              class="w-full px-2 py-1 text-xs font-medium rounded-md border border-gray-300 text-gray-600 hover:bg-gray-50 disabled:opacity-50 transition-colors"
            >
              {{ agentDispatching ? t('agent.dispatching') : t('agent.dispatchAgent') }}
            </button>
            <p class="text-[10px] text-gray-400 leading-snug">{{ t('agent.sidebarHint') }}</p>
            <p
              v-if="autoTriagePending"
              data-test="auto-triage-pending"
              class="text-[11px] text-violet-700 bg-violet-50 border border-violet-100 rounded-md px-2 py-1.5 leading-snug"
            >
              {{ t('issue.autoTriageRunning') }}
            </p>
          </div>
        </div>
        <div v-if="agentActivities.length" class="mt-2 space-y-1.5">
          <p class="text-[10px] font-semibold text-gray-400 uppercase tracking-wide">{{ t('agent.issueActivityTitle') }}</p>
          <div
            v-for="act in agentActivities"
            :key="act.id"
            class="rounded-md border border-gray-100 bg-gray-50 px-2 py-1.5 text-xs"
            :title="activityPreview(act.result_summary) || t('agent.activityNoSummary')"
          >
            <div class="flex items-center justify-between gap-2">
              <span class="font-medium text-violet-700 truncate">{{ act.agent_name }}</span>
              <span class="text-[10px] text-gray-400 shrink-0">{{ formatActivityTime(act.executed_at) }}</span>
            </div>
            <p class="text-gray-600 line-clamp-2 mt-0.5">{{ activityPreview(act.result_summary) || t('agent.activityNoSummary') }}</p>
          </div>
        </div>
      </div>

      <!-- Custom fields -->
      <template v-if="visibleCustomFields.length">
        <p class="pt-2 pb-1 text-[10px] font-semibold text-gray-400 uppercase tracking-wide border-t border-gray-100 mt-1">{{ t('issue.customFields') }}</p>
        <div
          v-for="cf in visibleCustomFields"
          :key="cf.field.id"
          class="prop-row"
          :class="{ 'pointer-events-none opacity-60': isLocked && cf.field.field_type !== 'boolean' }"
        >
          <label class="prop-label">
            {{ cf.field.name }}
            <span v-if="isFieldRequiredVisible(cf)" class="text-red-500">*</span>
          </label>
          <div class="prop-control">
            <input
              v-if="cf.field.field_type === 'text' || cf.field.field_type === 'url'"
              type="text"
              class="prop-input"
              :value="cf.value ?? ''"
              :disabled="isLocked"
              @change="(e: Event) => attemptCustomFieldUpdate(cf, (e.target as HTMLInputElement).value)"
            />
            <input
              v-else-if="cf.field.field_type === 'number'"
              type="number"
              class="prop-input"
              :value="cf.value ?? ''"
              :disabled="isLocked"
              @change="(e: Event) => attemptCustomFieldUpdate(cf, (e.target as HTMLInputElement).value)"
            />
            <input
              v-else-if="cf.field.field_type === 'date'"
              type="date"
              class="prop-input"
              :value="cf.value ?? ''"
              :disabled="isLocked"
              @change="(e: Event) => attemptCustomFieldUpdate(cf, (e.target as HTMLInputElement).value)"
            />
            <label v-else-if="cf.field.field_type === 'boolean'" class="flex items-center gap-2" :class="isLocked ? 'cursor-not-allowed' : 'cursor-pointer'">
              <input
                type="checkbox"
                class="w-4 h-4 rounded border-gray-300 text-indigo-600"
                :checked="cf.value === 'true'"
                :disabled="isLocked"
                @change="(e: Event) => attemptCustomFieldUpdate(cf, (e.target as HTMLInputElement).checked ? 'true' : 'false')"
              />
              <span class="text-sm text-gray-700">{{ cf.value === 'true' ? t('customField.yes') : t('customField.no') }}</span>
            </label>
            <select
              v-else-if="cf.field.field_type === 'dropdown'"
              class="prop-input"
              :value="cf.value ?? ''"
              :disabled="isLocked"
              @change="(e: Event) => attemptCustomFieldUpdate(cf, (e.target as HTMLSelectElement).value)"
            >
              <option value=""></option>
              <option v-for="opt in cf.field.options" :key="opt.id" :value="opt.value">{{ opt.value }}</option>
            </select>
            <select
              v-else-if="cf.field.field_type === 'member'"
              class="prop-input"
              :value="memberSelectValue(cf.value)"
              :disabled="isLocked"
              @change="(e: Event) => {
                const v = (e.target as HTMLSelectElement).value
                attemptCustomFieldUpdate(cf, v === '' ? '' : JSON.stringify([Number(v)]))
              }"
            >
              <option value=""></option>
              <option v-for="m in members" :key="m.id" :value="m.id">{{ m.display_name }}</option>
            </select>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed, nextTick } from 'vue'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast'
import { isEmptyIssueCustomFieldValue } from '@/utils/issue-custom-field-empty'
import AgentSelector from '@/components/AgentSelector.vue'
import LabelSelector from '@/components/LabelSelector.vue'
import PropertyPicker, { type PickerOption } from '@/components/PropertyPicker.vue'
import type { AgentStatus } from '@/api/issue-agent'
import { agentApi } from '@/api/agent'
import type { AgentActivity } from '@/types/agent'

interface StateOption { id: number; name: string }
interface MemberOption { id: number; display_name: string }
interface CycleOption { id: number; name: string }
interface ModuleOption { id: number; name: string }
interface CustomFieldEntry {
  field: {
    id: number
    name: string
    field_type: string
    is_required?: boolean
    options?: Array<{ id: number; value: string; color?: string }>
  }
  value: string | null
}

const props = defineProps<{
  issue: any
  states: StateOption[]
  members: MemberOption[]
  cycles: CycleOption[]
  modules: ModuleOption[]
  releases: Array<{ id: number; name: string; version: string }>
  customFields: CustomFieldEntry[]
  workspaceId: number
  /** When non-null, only these custom field IDs are shown (from conditional-field evaluate). */
  visibleFieldIds?: Set<number> | null
  agentDispatching?: boolean
  autoTriagePending?: boolean
  agentAssigning?: boolean
  agentStatus?: AgentStatus | null
  labels?: Array<{ id: number; name: string; color: string }>
  relationSummary?: {
    total: number
    outbound: number
    inbound: number
    byType: Record<string, { outbound: number; inbound: number }>
  } | null
}>()

const isLocked = computed(() => props.issue?.approval_status === 'pending')

const visibleCustomFields = computed(() => {
  const set = props.visibleFieldIds
  if (set === undefined || set === null) return props.customFields || []
  return (props.customFields || []).filter((cf) => set.has(cf.field.id))
})

function isFieldRequiredVisible(cf: CustomFieldEntry): boolean {
  if (!cf.field.is_required) return false
  const set = props.visibleFieldIds
  if (set === undefined || set === null) return true
  return set.has(cf.field.id)
}

const visibleStates = computed(() =>
  (props.states || []).filter((s: any) => {
    if (s?.is_active === false) return false
    if (/^E2E\s+Test/i.test(String(s?.name || ''))) return false
    return true
  })
)

const emit = defineEmits<{
  (e: 'update:state', stateId: number): void
  (e: 'update:priority', priority: string): void
  (e: 'update:assignee', userId: number | null): void
  (e: 'update:cycle', cycleId: number | null): void
  (e: 'update:module', moduleId: number | null): void
  (e: 'update:release', releaseId: number | null): void
  (e: 'update:startDate', date: string): void
  (e: 'update:targetDate', date: string): void
  (e: 'update:customField', fieldId: number, value: string): void
  (e: 'dispatch-agent', agentId: string): void
  (e: 'assign-agent', agentId: string): void
  (e: 'unassign-agent'): void
  (e: 'update:labels', labelIds: number[]): void
}>()

const localAgentId = ref('')
const syncingAgent = ref(false)
const hasAgentSelected = computed(() => !!localAgentId.value && localAgentId.value.startsWith('agent:'))

watch(() => props.agentStatus, (status) => {
  syncingAgent.value = true
  localAgentId.value = status?.agent_id ? `agent:${status.agent_id}` : ''
  nextTick(() => { syncingAgent.value = false })
}, { immediate: true })

// Only clear selection unassigns; assign requires the Assign button (no auto-assign on select).
watch(localAgentId, (val, oldVal) => {
  if (syncingAgent.value) return
  if (oldVal && oldVal.startsWith('agent:') && (!val || !val.startsWith('agent:'))) {
    emit('unassign-agent')
  }
})

function emitAssign() {
  if (!hasAgentSelected.value) return
  emit('assign-agent', localAgentId.value)
}

const { t, locale } = useI18n()
const toast = useToast()

const agentActivities = ref<AgentActivity[]>([])

async function loadAgentActivities() {
  const issueId = props.issue?.id
  if (!issueId || !props.workspaceId) {
    agentActivities.value = []
    return
  }
  try {
    agentActivities.value = await agentApi.listWorkspaceActivity(props.workspaceId, { issue_id: issueId, limit: 5 })
  } catch {
    agentActivities.value = []
  }
}

watch(() => [props.issue?.id, props.workspaceId], loadAgentActivities, { immediate: true })
watch(() => props.agentDispatching, (now, before) => {
  if (before && !now) loadAgentActivities()
})
watch(() => props.autoTriagePending, (now, before) => {
  if (before && !now) loadAgentActivities()
})

function formatActivityTime(iso: string): string {
  return new Date(iso).toLocaleString(locale.value, { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

// Agent summaries are markdown (headings, tables). The sidebar only has room for a
// short preview, so flatten the markup instead of dumping raw syntax into two lines.
// Never falls back to task_context: that holds the internal prompt.
function activityPreview(markdown: string | undefined): string {
  if (!markdown) return ''
  return markdown
    .replace(/```[\s\S]*?```/g, ' ')
    .split('\n')
    .filter(line => !/^\s*\|?[\s:|-]*\|[\s:|-]*\|?\s*$/.test(line))
    .map(line => line
      .replace(/^\s*#{1,6}\s*/, '')
      .replace(/^\s*>\s?/, '')
      .replace(/^\s*[-*+]\s+/, '· ')
      .replace(/\|/g, ' · ')
      .trim())
    .filter(Boolean)
    .join('  ')
    .replace(/\*\*(.+?)\*\*/g, '$1')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/\s+/g, ' ')
    .trim()
}

function memberSelectValue(raw: string | null): string | number {
  if (!raw) return ''
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed) && parsed.length > 0) return parsed[0] ?? ''
  } catch { /* ignore */ }
  return ''
}

function attemptCustomFieldUpdate(cf: CustomFieldEntry, value: string) {
  const set = props.visibleFieldIds
  if (set !== undefined && set !== null && !set.has(cf.field.id)) {
    return
  }
  if (isFieldRequiredVisible(cf) && isEmptyIssueCustomFieldValue(cf.field.field_type, value)) {
    toast.warning(t('issue.requiredFieldsMsg', { fields: cf.field.name }))
    return
  }
  emit('update:customField', cf.field.id, value)
}
function toId(value: string | number | null): number | null {
  return value === null ? null : Number(value)
}

const PRIORITY_COLORS: Record<string, string> = {
  urgent: '#ef4444',
  high: '#f97316',
  medium: '#eab308',
  low: '#3b82f6',
  none: '#d1d5db',
}

const stateOptions = computed<PickerOption[]>(() =>
  visibleStates.value.map((s: any) => ({ value: s.id, label: s.name, color: s.color || '#9ca3af' }))
)
const priorityOptions = computed<PickerOption[]>(() => [
  { value: 'urgent', label: t('issue.priorityUrgent'), color: PRIORITY_COLORS.urgent },
  { value: 'high', label: t('issue.priorityHigh'), color: PRIORITY_COLORS.high },
  { value: 'medium', label: t('issue.priorityMedium'), color: PRIORITY_COLORS.medium },
  { value: 'low', label: t('issue.priorityLow'), color: PRIORITY_COLORS.low },
  { value: 'none', label: t('issue.priorityNone'), color: PRIORITY_COLORS.none },
])
const memberOptions = computed<PickerOption[]>(() =>
  (props.members || []).map((m) => ({ value: m.id, label: m.display_name }))
)
const cycleOptions = computed<PickerOption[]>(() =>
  (props.cycles || []).map((c) => ({ value: c.id, label: c.name }))
)
const moduleOptions = computed<PickerOption[]>(() =>
  (props.modules || []).map((m) => ({ value: m.id, label: m.name }))
)
const releaseOptions = computed<PickerOption[]>(() =>
  (props.releases || []).map((r) => ({ value: r.id, label: r.version ? `${r.name} (${r.version})` : r.name }))
)

function emitStartDateUpdate(event: Event) {
  emit('update:startDate', (event.target as HTMLInputElement).value)
}
function emitTargetDateUpdate(event: Event) {
  emit('update:targetDate', (event.target as HTMLInputElement).value)
}
</script>

<style scoped>
.prop-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-height: 2rem;
  padding: 0.2rem 0;
}
.prop-label {
  width: 4.5rem;
  flex-shrink: 0;
  font-size: 0.75rem;
  line-height: 1rem;
  color: #6b7280;
}
.prop-control {
  flex: 1;
  min-width: 0;
}
.prop-input {
  width: 100%;
  padding: 0.25rem 0.375rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.25rem;
  font-size: 0.8125rem;
  line-height: 1.25rem;
  background: #fff;
}
.prop-input:disabled {
  background: #f3f4f6;
  cursor: not-allowed;
}
.prop-input:focus {
  outline: none;
  border-color: #a5b4fc;
  box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.15);
}
</style>
