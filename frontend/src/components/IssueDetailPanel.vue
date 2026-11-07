<template>
  <Teleport to="body">
    <div v-if="visible" class="fixed inset-0 z-50 flex">
      <!-- Backdrop -->
      <div class="absolute inset-0 bg-black/30" @click="close"></div>

      <!-- Slide-in panel - wide enough for tabs + sidebar -->
      <div class="relative ml-auto w-[90vw] max-w-5xl bg-white shadow-2xl flex flex-col h-full overflow-hidden animate-slide-in">
        <!-- Loading -->
        <div v-if="loading" class="flex items-center justify-center h-full">
          <svg class="animate-spin h-8 w-8 text-indigo-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
          </svg>
        </div>

        <template v-else-if="issue">
          <!-- Header bar -->
          <div class="flex items-center gap-2 px-4 py-2 border-b border-gray-200 shrink-0 bg-gray-50/50">
            <button @click="close" class="p-1 text-gray-400 hover:text-gray-600 hover:bg-gray-200 rounded transition-colors shrink-0">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
            </button>
            <span class="text-[11px] text-gray-400 font-mono shrink-0">{{ projectIdentifier }}-{{ issue.sequence_id }}</span>
            <span
              v-if="issue.issue_type"
              class="px-1.5 py-0.5 rounded text-[10px] font-medium shrink-0"
              :style="{ backgroundColor: (issue.issue_type.color || '#e5e7eb') + '20', color: issue.issue_type.color || '#6b7280' }"
            >{{ issue.issue_type.name }}</span>
            <input
              :value="issue.name"
              class="flex-1 text-sm font-semibold text-gray-800 bg-transparent border-0 outline-none focus:bg-white focus:border focus:border-indigo-300 focus:rounded focus:px-1.5 focus:py-0.5 min-w-0"
              @blur="(e: FocusEvent) => handleTitleChange((e.target as HTMLInputElement).value)"
              @keydown.enter="(e: KeyboardEvent) => { (e.target as HTMLInputElement).blur() }"
            />
            <span v-if="saving" class="text-[10px] text-indigo-500 animate-pulse shrink-0">{{ t('issue.saving') }}</span>
            <button
              type="button"
              class="p-1 rounded transition-colors shrink-0"
              :class="isWatching ? 'text-indigo-500 hover:bg-indigo-50' : 'text-gray-400 hover:text-gray-600 hover:bg-gray-100'"
              :title="isWatching ? t('issue.unwatch') : t('issue.watch')"
              @click="handleToggleWatch"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/></svg>
            </button>
            <button
              type="button"
              class="p-1 text-gray-400 hover:text-indigo-500 hover:bg-indigo-50 rounded transition-colors shrink-0"
              :title="t('issue.copyLink')"
              @click="copyPanelLink"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"/></svg>
            </button>
            <button
              v-if="can('issue:edit')"
              type="button"
              data-test="merge-into-btn"
              class="px-2 py-0.5 text-[11px] font-medium text-gray-600 border border-gray-200 rounded hover:bg-gray-100 transition-colors shrink-0"
              @click="openMergeDialog"
            >
              {{ t('issue.mergeInto') }}
            </button>
            <button
              v-if="can('issue:delete')"
              type="button"
              class="p-1 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded transition-colors shrink-0"
              :title="t('common.delete')"
              @click="handleDelete"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
            </button>
            <a
              :href="`/workspace/${workspaceSlug}/project/${projectId}/issues/${issue.id}`"
              class="p-1 text-gray-400 hover:text-indigo-500 rounded transition-colors shrink-0"
              :title="t('issue.openFullPage') || 'Open full page'"
              @click.prevent="navigateTo(issue.id)"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/></svg>
            </a>
          </div>

          <ApprovalPendingBanner
            :approval="activeApproval"
            :current-user-id="currentUserId"
            @decide="onDecideApproval"
            @cancel="onCancelApproval"
          />

          <!-- Body: tabs + sidebar -->
          <div class="flex-1 flex overflow-hidden">
            <!-- Main content area -->
            <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
              <!-- Tab buttons -->
              <div class="flex border-b border-gray-200 shrink-0 px-4">
                <button
                  v-for="tab in tabs"
                  :key="tab.key"
                  @click="activeTab = tab.key"
                  :class="activeTab === tab.key ? 'border-indigo-500 text-indigo-600' : 'border-transparent text-gray-500'"
                  class="px-3 py-2 text-xs font-medium border-b-2 -mb-[1px] transition-colors"
                >
                  {{ tab.label }}
                  <span v-if="tab.count !== undefined && tab.count > 0" class="ml-1 text-[10px] bg-gray-100 text-gray-500 px-1 py-0.5 rounded-full">{{ tab.count }}</span>
                </button>
              </div>

              <!-- Tab panels -->
              <div class="flex-1 overflow-y-auto px-4 py-3">
                <IssueTabDetails
                  v-if="activeTab === 'details'"
                  :issue-id="issue.id"
                  :issue="issue"
                  :workspace-id="workspaceId"
                  :project-id="projectId"
                  :issue-type-id="issue.issue_type?.id"
                  :members="projectMembers"
                  :show-title="false"
                  :comments-refresh-key="commentsRefreshKey"
                  @update:title="(v: string) => quickUpdate('name', v)"
                  @update:description="(v: string) => quickUpdate('description_html', v)"
                />
                <IssueTabRelations
                  v-else-if="activeTab === 'relations'"
                  ref="relationsTabRef"
                  :issue-id="issue.id"
                  :project-id="projectId"
                  :workspace-id="workspaceId"
                  :slug="workspaceSlug"
                  :states="stateOptions"
                  :parent="issue.parent"
                  :sub-issues="issue.sub_issues || []"
                  :issue-types="issueTypeOptions"
                  @navigate="navigateTo"
                  @refresh="reloadIssue"
                />
                <IssueTabAttachments
                  v-else-if="activeTab === 'attachments'"
                  :issue-id="issue.id"
                  :project-id="projectId"
                  @attachment-count-change="
                    (c) => {
                      attachmentCountOverride = c
                      if (issue) issue.attachment_count = c
                    }
                  "
                />
                <IssueTabTimeTracking
                  v-else-if="activeTab === 'timetrack'"
                  :issue-id="issue.id"
                />
                <IssueTabActivity
                  v-else-if="activeTab === 'activity'"
                  :issue-id="issue.id"
                  :workspace-id="workspaceId"
                />
                <IssueTabAI
                  v-else-if="activeTab === 'ai'"
                  :issue-id="issue.id"
                  :project-id="projectId"
                  :issue="issue"
                  :labels="projectLabels"
                  @open-copilot="showAICopilot = true"
                  @issue-updated="(updated: any) => { Object.assign(issue, updated); emit('refresh') }"
                />
                <ChatPanel
                  v-else-if="activeTab === 'chat'"
                  :issue-id="issue.id"
                  :workspace-id="workspaceId"
                  :current-user-id="currentUserId"
                  :mention-candidates="chatMentionCandidates"
                />
                <IssueGitPanel
                  v-else-if="activeTab === 'git'"
                  :workspace-id="workspaceId"
                  :issue-id="issue.id"
                />
              </div>
            </div>

            <!-- Right sidebar -->
            <div class="shrink-0 border-l border-gray-200 overflow-y-auto px-3 py-3">
              <IssuePropertySidebar
                v-if="issue"
                :issue="issue"
                :states="stateOptions"
                :members="projectMembers"
                :cycles="cycleOptions"
                :modules="moduleOptions"
                :releases="releaseOptions"
                :custom-fields="customFieldEntries"
                :visible-field-ids="sidebarVisibleFieldIds"
                :workspace-id="workspaceId"
                :agent-dispatching="agentDispatching"
                :auto-triage-pending="autoTriagePending"
                :agent-assigning="agentAssigning"
                :agent-status="agentStatus"
                :labels="projectLabels"
                :relation-summary="effectiveRelationSummary"
                @update:state="handleStateChange"
                @update:priority="(p: any) => quickUpdate('priority', p)"
                @update:assignee="quickUpdateAssignee"
                @update:cycle="quickUpdateCycle"
                @update:module="quickUpdateModule"
                @update:release="quickUpdateRelease"
                @update:start-date="(d: any) => quickUpdate('start_date', d + 'T00:00:00Z')"
                @update:target-date="(d: any) => quickUpdate('target_date', d + 'T00:00:00Z')"
                @update:labels="handleLabelsUpdate"
                @update:custom-field="updateCustomField"
                @assign-agent="assignAgent"
                @unassign-agent="unassignAgent"
                @dispatch-agent="dispatchAgent"
              />
            </div>
          </div>
        </template>

        <!-- Approval Submit Dialog -->
        <ApprovalSubmitDialog
          :show="showSubmitDialog"
          :issue-id="issue?.id || 0"
          :transition-id="submitDialogData.transitionId"
          :from-state-name="submitDialogData.fromStateName"
          :approve-state-name="submitDialogData.approveStateName"
          :approver-names="submitDialogData.approverNames"
          :workflow-name="submitDialogData.workflowName"
          @close="showSubmitDialog = false"
          @submitted="onApprovalSubmitted"
        />
        <ApprovalDecisionDialog
          :show="showDecisionDialog"
          :approval-id="decisionDialogData?.approvalId || 0"
          :decision="decisionDialogData?.decision || 'approved'"
          @close="showDecisionDialog = false"
          @decided="onApprovalDecided"
        />
      </div>

      <AICopilot
        v-if="issue"
        :visible="showAICopilot"
        :project-id="projectId"
        :workspace-id="workspaceId"
        :project-name="projectIdentifier"
        :issue-id="issue.id"
        :issue-label="`${projectIdentifier}-${issue.sequence_id} ${issue.name || ''}`.trim()"
        view="issue_detail"
        @close="showAICopilot = false"
      />

      <MergeIntoIssueDialog
        v-if="issue"
        :visible="showMergeDialog"
        :project-id="projectId"
        :source-issue-id="issue.id"
        :source-label="mergeSourceLabel"
        :project-identifier="projectIdentifier"
        :initial-target-id="mergePrefillTargetId"
        @close="showMergeDialog = false"
        @merged="onMergeSuccess"
      />
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, computed, onBeforeUnmount } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import issueApi from '@/api/issue'
import * as issueTypeApi from '@/api/issue-type'
import * as stateApi from '@/api/project-settings'
import * as cycleApi from '@/api/cycle'
import * as moduleApi from '@/api/module'
import { releaseApi } from '@/api/release'
import api from '@/api'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { useProjectPermissions } from '@/composables/usePermission'
import { agentApi } from '@/api/agent'
import { issueAgentApi, type AgentStatus } from '@/api/issue-agent'
import { buildDispatchTask } from '@/utils/agentDispatchTask'
import { useAutoTriageWatch } from '@/composables/useAutoTriageWatch'
import { getIssueCustomFieldsWithDefinitions, updateIssueCustomFieldValue } from '@/api/custom-field'
import relationApi from '@/api/relation'
import IssuePropertySidebar from '@/components/IssuePropertySidebar.vue'
import IssueTabDetails from '@/components/IssueTabDetails.vue'
import IssueTabRelations from '@/components/IssueTabRelations.vue'
import IssueTabAttachments from '@/components/IssueTabAttachments.vue'
import IssueTabTimeTracking from '@/components/IssueTabTimeTracking.vue'
import IssueTabActivity from '@/components/IssueTabActivity.vue'
import IssueTabAI from '@/components/IssueTabAI.vue'
import IssueGitPanel from '@/components/IssueGitPanel.vue'
import ChatPanel from '@/components/chat/ChatPanel.vue'
import AICopilot from '@/components/AICopilot.vue'
import type { Agent } from '@/types/agent'
import ApprovalSubmitDialog from '@/components/ApprovalSubmitDialog.vue'
import ApprovalDecisionDialog from '@/components/ApprovalDecisionDialog.vue'
import ApprovalPendingBanner from '@/components/ApprovalPendingBanner.vue'
import MergeIntoIssueDialog from '@/components/MergeIntoIssueDialog.vue'
import approvalApi, { type ApprovalResponse } from '@/api/approval'
import type { FieldVisibilityEvaluateRequest } from '@/api/conditional-field'
import { useCustomFieldVisibility } from '@/composables/useCustomFieldVisibility'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const toast = useToast()
const { confirm } = useConfirm()

const workspaceSlug = computed(() => (route.params as any).slug as string || '')

const props = defineProps<{
  issueId: number | null
  visible: boolean
  workspaceId: number
  projectId: number
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'delete', issue: any): void
  (e: 'refresh'): void
}>()

const { can } = useProjectPermissions(() => props.projectId)

const issue = ref<any>(null)
/** Local sync for attachments tab count vs list in AttachmentManager */
const attachmentCountOverride = ref<number | null>(null)
const loading = ref(false)
const activeTab = ref('details')

const stateOptions = ref<any[]>([])
const cycleOptions = ref<any[]>([])
const moduleOptions = ref<any[]>([])
const releaseOptions = ref<{ id: number; name: string; version: string }[]>([])
const issueTypeOptions = ref<any[]>([])
const projectMembers = ref<any[]>([])
const projectIdentifier = ref('')
const customFieldEntries = ref<Array<{ field: any; value: string | null }>>([])
const panelWorkspaceId = computed(() => props.workspaceId)
const { visibleFieldIds: sidebarVisibleFieldIds, scheduleEvaluate, evaluateNow } = useCustomFieldVisibility(panelWorkspaceId)
const relationsTabRef = ref<InstanceType<typeof IssueTabRelations> | null>(null)
const agentDispatching = ref(false)
const commentsRefreshKey = ref(0)
const watchIssueId = computed(() => issue.value?.id ?? props.issueId)
const watchWorkspaceId = computed(() => props.workspaceId)
const { pending: autoTriagePending } = useAutoTriageWatch({
  workspaceId: watchWorkspaceId,
  issueId: watchIssueId,
  onRefreshComments: () => { commentsRefreshKey.value++ },
  onDone: (ok) => {
    if (ok) toast.success(t('issue.autoTriageDone'))
    else toast.info(t('issue.autoTriageTimeout'))
  },
})
const agentAssigning = ref(false)
const agentStatus = ref<AgentStatus | null>(null)
const workspaceAgents = ref<Agent[]>([])
const showAICopilot = ref(false)
const showMergeDialog = ref(false)
const mergePrefillTargetId = ref<number | null>(null)

const mergeSourceLabel = computed(() => {
  if (!issue.value) return ''
  const prefix = projectIdentifier.value ? `${projectIdentifier.value}-` : '#'
  return `${prefix}${issue.value.sequence_id} ${issue.value.name || ''}`.trim()
})
const saving = ref(false)
const isWatching = ref(false)
const projectLabels = ref<Array<{ id: number; name: string; color: string }>>([])
const relationSidebarSummary = ref<{
  total: number
  outbound: number
  inbound: number
  byType: Record<string, { outbound: number; inbound: number }>
} | null>(null)
const showSubmitDialog = ref(false)
const submitDialogData = ref<{ transitionId: number; fromStateName: string; approveStateName: string; approverNames: string[]; workflowName?: string }>({
  transitionId: 0, fromStateName: '', approveStateName: '', approverNames: []
})
const activeApproval = ref<ApprovalResponse | null>(null)
const currentUserId = parseInt(localStorage.getItem('user_id') || '0', 10)
const showDecisionDialog = ref(false)
const decisionDialogData = ref<{ approvalId: number; decision: 'approved' | 'rejected' } | null>(null)

async function handleTitleChange(newTitle: string) {
  if (!issue.value || newTitle === issue.value.name) return
  saving.value = true
  try {
    const updated = await issueApi.updateIssue(issue.value.id, { name: newTitle })
    if (issue.value) Object.assign(issue.value, updated)
  } catch (e: any) {
    toast.error(e?.response?.data?.message || t('issue.saveFailed'))
  } finally { saving.value = false }
}

const relationSidebarSummaryFromTab = computed(() => {
  return relationsTabRef.value?.relationSummary ?? null
})
const effectiveRelationSummary = computed(() => relationSidebarSummary.value || relationSidebarSummaryFromTab.value)

const tabs = computed(() => [
  { key: 'details', label: t('issue.tabDetails'), count: undefined },
  { key: 'relations', label: t('issue.tabRelations'), count: effectiveRelationSummary.value?.total ?? undefined },
  {
    key: 'attachments',
    label: t('issue.tabAttachments'),
    count: (() => {
      const n = attachmentCountOverride.value ?? issue.value?.attachment_count
      return n != null && n > 0 ? n : undefined
    })(),
  },
  { key: 'activity', label: t('issue.tabActivity'), count: undefined },
  { key: 'ai', label: 'AI', count: undefined },
  { key: 'chat', label: t('issue.tabChat'), count: undefined },
  { key: 'git', label: t('gitIntegration.title'), count: undefined },
  { key: 'timetrack', label: t('issue.tabTimetrack'), count: undefined },
])

const chatMentionCandidates = computed(() => [
  ...projectMembers.value.map((m: any) => ({
    id: m.id,
    name: m.display_name || m.username || `User #${m.id}`,
    type: 'user' as const,
  })),
  ...workspaceAgents.value
    .filter((a) => a.status === 'active')
    .map((a) => ({ id: a.id, name: a.name, type: 'agent' as const })),
])

function navigateTo(issueId: number) {
  const slug = workspaceSlug.value
  if (slug) {
    router.push(`/workspace/${slug}/project/${props.projectId}/issues/${issueId}`)
  }
}

// ---- Data loading ----
watch(() => [props.issueId, props.visible] as const, async ([id, vis]) => {
  if (id && vis) {
    loading.value = true
    activeTab.value = 'details'
    showAICopilot.value = false
    try {
      const result = await issueApi.getIssue(id)
      issue.value = result
      attachmentCountOverride.value = null
      projectIdentifier.value = result.project?.identifier || ''
      await Promise.all([
        loadStates(),
        loadCycles(),
        loadModules(),
        loadReleases(),
        loadIssueTypes(),
        loadMembers(),
        loadCustomFields(),
        loadLabels(),
        loadAgentStatus(id as number),
        loadRelationSummary(id as number),
        loadWatchers(id as number),
        loadWorkspaceAgents(),
      ])
      await loadActiveApproval()
    } catch (e) {
      console.error('Failed to load issue:', e)
      issue.value = null
      attachmentCountOverride.value = null
    } finally {
      loading.value = false
    }
  } else if (!vis) {
    issue.value = null
    attachmentCountOverride.value = null
    activeApproval.value = null
    agentStatus.value = null
    showAICopilot.value = false
  }
}, { immediate: true })

function closeCopilotOnEscape(e: KeyboardEvent) {
  if (e.key !== 'Escape') return
  e.preventDefault()
  e.stopImmediatePropagation()
  showAICopilot.value = false
}

watch(showAICopilot, (open) => {
  if (open) window.addEventListener('keydown', closeCopilotOnEscape, true)
  else window.removeEventListener('keydown', closeCopilotOnEscape, true)
})
onBeforeUnmount(() => window.removeEventListener('keydown', closeCopilotOnEscape, true))

async function loadWorkspaceAgents() {
  if (!props.workspaceId) return
  try {
    workspaceAgents.value = await agentApi.list(props.workspaceId)
  } catch {
    workspaceAgents.value = []
  }
}

async function loadAgentStatus(id: number) {
  try {
    const res = await issueAgentApi.getStatus(id)
    agentStatus.value = res.data
  } catch {
    agentStatus.value = null
  }
}

async function loadStates() {
  try {
    const raw = await stateApi.listStates(props.projectId)
    stateOptions.value = (raw || []).filter((s: any) => {
      if (s?.is_active === false) return false
      if (/^E2E\s+Test/i.test(String(s?.name || ''))) return false
      return true
    })
  } catch { /* */ }
}

async function loadRelationSummary(id: number) {
  try {
    const relations = await relationApi.listIssueRelations(id, 'both')
    const list = Array.isArray(relations) ? relations : []
    const byType: Record<string, { outbound: number; inbound: number }> = {}
    let outbound = 0
    let inbound = 0
    for (const rel of list) {
      const typeName = rel.relation_type?.name || rel.relation_name || ''
      if (!byType[typeName]) byType[typeName] = { outbound: 0, inbound: 0 }
      if (rel.direction === 'outbound') {
        byType[typeName].outbound++
        outbound++
      } else {
        byType[typeName].inbound++
        inbound++
      }
    }
    relationSidebarSummary.value = { total: list.length, outbound, inbound, byType }
  } catch {
    relationSidebarSummary.value = null
  }
}

async function loadWatchers(id: number) {
  try {
    const { watchers } = await issueApi.listWatchers(id)
    isWatching.value = (watchers || []).includes(currentUserId)
  } catch {
    isWatching.value = false
  }
}
async function loadCycles() {
  try { const d = await cycleApi.listCycles(props.projectId); cycleOptions.value = d?.items || d || [] } catch { /* */ }
}
async function loadModules() {
  try { moduleOptions.value = await moduleApi.listModules(props.projectId, props.workspaceId) } catch { /* */ }
}
async function loadReleases() {
  try { releaseOptions.value = await releaseApi.list(props.projectId) } catch { /* */ }
}
async function loadIssueTypes() {
  try { issueTypeOptions.value = await issueTypeApi.getIssueTypes(props.workspaceId, props.projectId) } catch { /* */ }
}
async function loadMembers() {
  try {
    const d = await api.get(`/projects/${props.projectId}/members`)
    projectMembers.value = (d.data || []).map((m: any) => m.user || m)
  } catch { /* */ }
}
function issueLabelIdsForVisibility(): number[] {
  const i = issue.value
  if (!i) return []
  if (Array.isArray(i.label_ids) && i.label_ids.length) {
    return i.label_ids.map((x: any) => Number(x)).filter((n: number) => n > 0)
  }
  if (Array.isArray(i.labels) && i.labels.length && typeof i.labels[0] === 'number') {
    return (i.labels as number[]).filter((n) => n > 0)
  }
  if (Array.isArray(i.label_details) && i.label_details.length) {
    return i.label_details.map((l: { id: number }) => l.id).filter((id: number) => id > 0)
  }
  return []
}

function buildSidebarVisibilityBody(): FieldVisibilityEvaluateRequest | null {
  if (!props.workspaceId || !issue.value) return null
  const sid = Number(issue.value.state_id)
  if (!sid || sid <= 0) return null
  const typeId = issue.value.issue_type?.id ?? issue.value.issue_type_id ?? issue.value.type_id
  const pri = String(issue.value.priority ?? 'none').trim().toLowerCase() || 'none'
  let assignee_ids: number[] = []
  if (Array.isArray(issue.value.assignee_ids) && issue.value.assignee_ids.length > 0) {
    assignee_ids = issue.value.assignee_ids.map((x: any) => Number(x)).filter((n: number) => n > 0)
  } else if (issue.value.assignees?.[0]?.id) {
    assignee_ids = [Number(issue.value.assignees[0].id)]
  }
  return {
    issue_type_id: typeId ? Number(typeId) : undefined,
    state_id: sid,
    priority: pri,
    assignee_ids,
    label_ids: issueLabelIdsForVisibility(),
  }
}

async function loadCustomFields() {
  try {
    const resp = await getIssueCustomFieldsWithDefinitions(props.issueId!)
    if (resp?.fields) {
      customFieldEntries.value = resp.fields.map((item: any) => ({
        field: {
          id: item.id,
          name: item.name,
          field_type: item.field_type,
          options: item.options || [],
          is_required: !!item.is_required,
        },
        value: item.value ?? null,
      }))
    } else {
      customFieldEntries.value = []
    }
    await evaluateNow(buildSidebarVisibilityBody)
  } catch { /* */ }
}

watch(
  () => issue.value?.issue_type?.id ?? issue.value?.issue_type_id,
  (next, prev) => {
    if (next === prev || prev === undefined) return
    loadCustomFields()
  },
)

watch(
  () => [
    issue.value?.state_id,
    issue.value?.priority,
    issue.value?.assignee_ids,
    issue.value?.assignees,
    issue.value?.label_ids,
    issue.value?.labels,
    issue.value?.issue_type?.id,
    issue.value?.issue_type_id,
    props.workspaceId,
  ],
  () => {
    scheduleEvaluate(buildSidebarVisibilityBody)
  },
  { deep: true }
)

async function loadActiveApproval() {
  if (!issue.value) return
  if (issue.value.approval_status === 'pending' && issue.value.active_approval_id) {
    try {
      activeApproval.value = await approvalApi.get(issue.value.active_approval_id)
    } catch (e) {
      console.error('Failed to load active approval:', e)
      activeApproval.value = null
    }
  } else {
    activeApproval.value = null
  }
}

// ---- State change with approval handling ----
async function handleStateChange(newStateId: number) {
  if (!issue.value) return
  if (issue.value.approval_status === 'pending') {
    toast.error(t('approvals.stateDisabledHint'))
    return
  }
  try {
    const updated = await issueApi.updateIssue(issue.value.id, { state_id: newStateId })
    if (issue.value) Object.assign(issue.value, updated)
    emit('refresh')
  } catch (e: any) {
    if (e?.response?.status === 409 && e?.response?.data?.message === 'approval_required') {
      const transitionId = e.response.data.transition_id
      const sourceStateId = e.response.data.source_state_id
      const targetStateId = e.response.data.target_state_id
      const workflowName = e.response.data.workflow_name
      const sourceStateName = e.response.data.source_state_name
      const targetStateName = e.response.data.target_state_name
      try {
        const workspaceId = issue.value.workspace_id
        let transition: any = null
        const workflowId = e.response.data.workflow_id
        if (workspaceId && workflowId) {
          try {
            const trRes = await api.get(`/workspaces/${workspaceId}/workflows/${workflowId}/transitions`)
            const list = Array.isArray(trRes.data) ? trRes.data : (trRes.data?.data || [])
            transition = list.find((tr: any) => tr.id === transitionId) || null
          } catch { /* fall through */ }
        }
        if (!transition && workspaceId) {
          const wfRes = await api.get(`/workspaces/${workspaceId}/workflows`)
          const workflows = Array.isArray(wfRes.data) ? wfRes.data : (wfRes.data?.data || [])
          for (const w of workflows) {
            const found = (w.transitions || []).find((tr: any) => tr.id === transitionId)
            if (found) { transition = found; break }
          }
        }
        let approverNames: string[] = []
        const rawApprovers = transition?.approver_ids
        if (rawApprovers) {
          try {
            const ids: number[] = typeof rawApprovers === 'string' ? JSON.parse(rawApprovers) : rawApprovers
            approverNames = ids.map(id => {
              const m = projectMembers.value.find((m: any) => m.user_id === id || m.id === id)
              return m?.user?.display_name || m?.display_name || `#${id}`
            })
          } catch { /* ignore parse errors */ }
        }
        submitDialogData.value = {
          transitionId,
          fromStateName: sourceStateName || `#${sourceStateId}`,
          approveStateName: targetStateName || `#${targetStateId}`,
          approverNames,
          workflowName,
        }
        showSubmitDialog.value = true
      } catch (fetchErr) {
        console.error('Failed to fetch transition details:', fetchErr)
        toast.error(t('issue.saveFailed'))
      }
    } else {
      console.error('Failed to update state:', e)
      toast.error(e?.response?.data?.message || t('issue.saveFailed'))
    }
  }
}

async function onApprovalSubmitted() {
  showSubmitDialog.value = false
  await reloadIssue()
  await loadActiveApproval()
}

function onDecideApproval(approval: ApprovalResponse, decision: 'approved' | 'rejected') {
  decisionDialogData.value = { approvalId: approval.id, decision }
  showDecisionDialog.value = true
}

async function onApprovalDecided() {
  showDecisionDialog.value = false
  decisionDialogData.value = null
  await reloadIssue()
  await loadActiveApproval()
}

async function onCancelApproval(approval: ApprovalResponse) {
  if (!(await confirm(t('approvals.cancelApproval')))) return
  try {
    await approvalApi.cancel(approval.id)
    await reloadIssue()
    await loadActiveApproval()
  } catch (e: any) {
    toast.error(e?.response?.data?.message || t('issue.saveFailed'))
  }
}

// ---- Quick updates (same pattern as IssueDetail.vue) ----
async function quickUpdate(field: string, value: any) {
  try {
    const updated = await issueApi.updateIssue(issue.value.id, { [field]: value })
    if (issue.value) Object.assign(issue.value, updated)
    emit('refresh')
  } catch (e: any) {
    toast.error(e?.response?.data?.message || t('issue.saveFailed'))
  }
}

async function quickUpdateAssignee(userId: any) {
  try {
    const assigneeIds = userId ? [Number(userId)] : []
    const updated = await issueApi.updateIssue(issue.value.id, { assignee_ids: assigneeIds })
    if (issue.value) Object.assign(issue.value, updated)
    emit('refresh')
  } catch (e: any) { toast.error(t('issue.saveFailed')) }
}

async function quickUpdateCycle(cycleId: number | null) {
  try {
    if (cycleId) {
      await issueApi.setIssueCycle(issue.value.id, cycleId)
    } else {
      await issueApi.removeIssueCycle(issue.value.id)
    }
    await reloadIssue()
  } catch (e: any) { toast.error(t('issue.saveFailed')) }
}

async function quickUpdateModule(moduleId: number | null) {
  await quickUpdate('module_ids', moduleId ? [moduleId] : [])
}

async function quickUpdateRelease(releaseId: number | null) {
  await quickUpdate('release_id', releaseId ?? 0)
}

async function updateCustomField(fieldId: number, value: string) {
  try {
    await updateIssueCustomFieldValue(issue.value.id, fieldId, { value })
    const entry = customFieldEntries.value.find(e => e.field.id === fieldId)
    if (entry) entry.value = value
  } catch { toast.error(t('issue.saveFailed')) }
}

async function loadLabels() {
  try {
    const resp = await api.get(`/projects/${props.projectId}/settings/labels`)
    projectLabels.value = Array.isArray(resp.data) ? resp.data : (resp.data?.items || [])
  } catch { /* */ }
}

async function handleLabelsUpdate(labelIds: number[]) {
  await quickUpdate('label_ids', labelIds)
}

async function assignAgent(agentSelectorId: string) {
  if (!agentSelectorId || !agentSelectorId.startsWith('agent:') || !issue.value) return
  const agentId = parseInt(agentSelectorId.replace('agent:', ''), 10)
  if (!agentId) return
  agentAssigning.value = true
  try {
    await issueAgentApi.assign(issue.value.id, { agent_id: agentId })
    await loadAgentStatus(issue.value.id)
    toast.success(t('agent.assignSuccess'))
  } catch (e: any) {
    toast.error(e?.response?.data?.message || e?.message || t('common.error'))
  } finally {
    agentAssigning.value = false
  }
}

async function unassignAgent() {
  if (!issue.value) return
  agentAssigning.value = true
  try {
    await issueAgentApi.unassign(issue.value.id)
    await loadAgentStatus(issue.value.id)
    toast.success(t('agent.unassignSuccess'))
  } catch (e: any) {
    toast.error(e?.response?.data?.message || e?.message || t('common.error'))
  } finally {
    agentAssigning.value = false
  }
}

// An agent run takes minutes. The request is fire-and-forget so the UI never looks
// frozen: the thread shows the agent working, and the result lands as a comment.
const DISPATCH_BUTTON_GUARD_MS = 1500

async function dispatchAgent(agentSelectorId: string) {
  if (!agentSelectorId || !agentSelectorId.startsWith('agent:')) return
  const agentId = parseInt(agentSelectorId.replace('agent:', ''))
  if (!agentId || agentDispatching.value || !issue.value) return
  agentDispatching.value = true
  setTimeout(() => { agentDispatching.value = false }, DISPATCH_BUTTON_GUARD_MS)
  const agents = await agentApi.list(props.workspaceId).catch(() => [])
  const agent = agents.find(a => a.id === agentId)
  const task = buildDispatchTask(
    agent,
    { seq: String(issue.value.sequence_id ?? ''), name: issue.value.name || '' },
    t,
  )
  agentApi
    .dispatch(props.workspaceId, agentId, {
      task,
      issue_id: issue.value.id,
      project_id: props.projectId,
    })
    .then(() => {
      if (issue.value) loadAgentStatus(issue.value.id)
    })
    .catch((e: any) => {
      toast.error(e?.response?.data?.message || e?.message || 'Failed to dispatch agent')
    })
  toast.success(t('agent.dispatch'))
  commentsRefreshKey.value++
}

async function reloadIssue() {
  if (!issue.value) return
  try {
    const data = await issueApi.getIssue(issue.value.id)
    Object.assign(issue.value, data)
    await loadRelationSummary(issue.value.id)
    await relationsTabRef.value?.loadLinkedPages?.()
  } catch { /* */ }
}

async function copyPanelLink() {
  if (!issue.value) return
  const url = `${window.location.origin}/workspace/${workspaceSlug.value}/project/${props.projectId}/issues/${issue.value.id}`
  try {
    await navigator.clipboard.writeText(url)
    toast.success(t('issue.copied'))
  } catch {
    toast.error(t('issue.copyFailed'))
  }
}

async function handleToggleWatch() {
  if (!issue.value) return
  try {
    if (isWatching.value) {
      await issueApi.removeWatcher(issue.value.id)
      isWatching.value = false
    } else {
      await issueApi.addWatcher(issue.value.id)
      isWatching.value = true
    }
  } catch (e: any) {
    toast.error(e?.response?.data?.message || t('issue.saveFailed'))
  }
}

function openMergeDialog() {
  mergePrefillTargetId.value = null
  showMergeDialog.value = true
}

function onMergeSuccess(targetId: number) {
  showMergeDialog.value = false
  mergePrefillTargetId.value = null
  toast.success(t('issue.mergeIntoSuccess'))
  close()
  navigateTo(targetId)
}

async function handleDelete() {
  if (!issue.value) return
  if (!(await confirm(t('issueDetail.confirmDelete', { 0: issue.value.name })))) return
  emit('delete', issue.value)
  close()
}

function close() {
  emit('close')
}
</script>

<style scoped>
@keyframes slideIn {
  from { transform: translateX(100%); }
  to { transform: translateX(0); }
}
.animate-slide-in {
  animation: slideIn 0.25s ease-out;
}
</style>
