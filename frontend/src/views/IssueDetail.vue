<template>
  <div class="issue-detail-page min-h-screen bg-white">
    <IssueDetailHeader
      :issue
      :project-identifier="projectIdentifier"
      :is-watching="isWatching"
      @back="goBack"
      @delete="deleteIssue"
      @merge="openMergeDialog"
      @toggle-watch="handleToggleWatch"
      @update:title="handleTitleSave"
    />

    <MergeIntoIssueDialog
      :visible="showMergeDialog"
      :project-id="projectId"
      :source-issue-id="issueId"
      :source-label="mergeSourceLabel"
      :project-identifier="projectIdentifier"
      :initial-target-id="mergePrefillTargetId"
      @close="closeMergeDialog"
      @merged="onMergeSuccess"
    />

    <div class="max-w-6xl mx-auto px-6 pt-4">
      <ApprovalPendingBanner
        :approval="activeApproval"
        :current-user-id="currentUserId"
        @decide="onDecideApproval"
        @cancel="onCancelApproval"
      />
    </div>
    <ApprovalSubmitDialog
      :show="showSubmitDialog"
      :issue-id="issueId"
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

    <div class="max-w-6xl mx-auto px-6 py-6">
      <div class="flex gap-8">
        <!-- Main content: tab buttons + tab panels -->
        <div class="flex-1 min-w-0">
          <!-- 5 tab buttons -->
          <div class="flex border-b-2 border-gray-200 mb-4">
            <button
              v-for="tab in tabs"
              :key="tab.key"
              data-test="tab-btn"
              @click="activeTab = tab.key"
              :class="activeTab === tab.key
                ? 'border-indigo-500 text-indigo-600'
                : 'border-transparent text-gray-500'"
              class="px-4 py-2 text-sm font-medium border-b-2 -mb-0.5 transition-colors"
            >
              {{ tab.label }}
              <span v-if="tab.count !== undefined && tab.count > 0" class="ml-1.5 text-xs bg-gray-100 text-gray-500 px-1.5 py-0.5 rounded-full">{{ tab.count }}</span>
            </button>
          </div>

          <!-- Tab panels -->
          <IssueTabDetails
            v-if="activeTab === 'details'"
            v-bind="detailProps"
            @update:description="handleDescriptionSave"
            @navigate="navigateToIssue"
            @suggestionsApplied="onSuggestionsApplied"
          />
          <IssueTabRelations
            v-else-if="activeTab === 'relations'"
            ref="relationsTabRef"
            :issue-id="issueId"
            :project-id="projectId"
            :workspace-id="workspaceId"
            :slug="(route.params.slug as string)"
            :states
            :parent="issue?.parent"
            :sub-issues="issue?.sub_issues || []"
            :issue-types="issueTypes"
            :current-issue-type="issue?.issue_type"
            @navigate="navigateToIssue"
            @refresh="handleRelationsRefresh"
          />
          <IssueTabAttachments
            v-else-if="activeTab === 'attachments'"
            :issue-id="issueId"
            :project-id="projectId"
            @attachment-count-change="
              (c) => {
                attachmentCountOverride = c
                if (issue) issue.attachment_count = c
              }
            "
          />
          <IssueGitPanel
            v-else-if="activeTab === 'git'"
            :workspace-id="workspaceId"
            :issue-id="issueId"
          />
          <IssueTabTimeTracking
            v-else-if="activeTab === 'timetrack'"
            :issue-id="issueId"
          />
          <IssueTabActivity
            v-else-if="activeTab === 'activity'"
            :issue-id="issueId"
            :workspace-id="workspaceId"
          />
          <IssueTabAI
            v-else-if="activeTab === 'ai'"
            :issue-id="issueId"
            :project-id="projectId"
            :issue="issue"
            :labels="projectLabels"
            @open-copilot="showAICopilot = true"
            @issue-updated="(updated: any) => (issue = updated)"
          />
          <!-- Chat Tab (lazy: only mounts when active) -->
          <ChatPanel
            v-else-if="activeTab === 'chat'"
            :issue-id="issueId"
            :workspace-id="workspaceId"
            :current-user-id="currentUserId"
            :mention-candidates="chatMentionCandidates"
          />
        </div>

        <!-- Right sidebar -->
        <IssuePropertySidebar
          v-if="issue"
          :issue
          :states
          :members="projectMembers"
          :cycles
          :modules
          :releases
          :custom-fields="customFieldEntries"
          :visible-field-ids="sidebarVisibleFieldIds"
          :workspace-id="workspaceId"
          :agent-dispatching="agentDispatching"
          :auto-triage-pending="autoTriagePending"
          :agent-assigning="agentAssigning"
          :agent-status="agentStatus"
          :labels="projectLabels"
          :relation-summary="relationSidebarSummary"
          @update:state="(id: any) => handleStateChange(id)"
          @update:priority="(p: any) => instantUpdate('priority', p)"
          @update:assignee="instantUpdateAssignee"
          @update:cycle="handleCycleUpdate"
          @update:module="handleModuleUpdate"
          @update:release="handleReleaseUpdate"
          @update:estimate="handleEstimateUpdate"
          @update:start-date="(d: any) => instantUpdate('start_date', d + 'T00:00:00Z')"
          @update:target-date="(d: any) => instantUpdate('target_date', d + 'T00:00:00Z')"
          @update:labels="handleLabelsUpdate"
          @update:custom-field="updateCustomField"
          @assign-agent="(id: string) => assignAgent(id)"
          @unassign-agent="unassignAgent"
          @dispatch-agent="(id: string) => dispatchAgent(id)"
        />
      </div>
    </div>

    <!-- AI Copilot Side Panel -->
    <AICopilot
      :visible="showAICopilot"
      :project-id="projectId"
      :workspace-id="workspaceId"
      :project-name="projectIdentifier"
      :issue-id="issueId"
      :issue-label="issueCopilotLabel"
      view="issue_detail"
      @close="showAICopilot = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast'
import { useRoute, useRouter } from 'vue-router'
import * as issueApi from '@/api/issue'
import * as stateApi from '@/api/project-settings'
import * as cycleApi from '@/api/cycle'
import * as moduleApi from '@/api/module'
import { setIssueCycle, removeIssueCycle } from '@/api/issue'
import * as issueTypeApi from '@/api/issue-type'
import { releaseApi } from '@/api/release'
import projectApi from '@/api/project'
import api from '@/api'
import { agentApi } from '@/api/agent'
import { issueAgentApi, type AgentStatus } from '@/api/issue-agent'
import type { Agent } from '@/types/agent'
import { buildDispatchTask } from '@/utils/agentDispatchTask'
import { useAutoTriageWatch } from '@/composables/useAutoTriageWatch'
import { useConfirm } from '@/composables/useConfirm'
import { getIssueCustomFieldsWithDefinitions, updateIssueCustomFieldValue } from '@/api/custom-field'
import relationApi from '@/api/relation'
import IssueDetailHeader from '@/components/IssueDetailHeader.vue'
import MergeIntoIssueDialog from '@/components/MergeIntoIssueDialog.vue'
import IssuePropertySidebar from '@/components/IssuePropertySidebar.vue'
import IssueTabDetails from '@/components/IssueTabDetails.vue'
import IssueTabRelations from '@/components/IssueTabRelations.vue'
import IssueTabAttachments from '@/components/IssueTabAttachments.vue'
import IssueTabTimeTracking from '@/components/IssueTabTimeTracking.vue'
import IssueTabActivity from '@/components/IssueTabActivity.vue'
import IssueGitPanel from '@/components/IssueGitPanel.vue'
import IssueTabAI from '@/components/IssueTabAI.vue'
import ChatPanel from '@/components/chat/ChatPanel.vue'
import ApprovalSubmitDialog from '@/components/ApprovalSubmitDialog.vue'
import ApprovalDecisionDialog from '@/components/ApprovalDecisionDialog.vue'
import ApprovalPendingBanner from '@/components/ApprovalPendingBanner.vue'
import AICopilot from '@/components/AICopilot.vue'
import approvalApi, { type ApprovalResponse } from '@/api/approval'
import type { FieldVisibilityEvaluateRequest } from '@/api/conditional-field'
import { useCustomFieldVisibility } from '@/composables/useCustomFieldVisibility'
import { useProjectPermissions } from '@/composables/usePermission'
// Route params
const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { confirm } = useConfirm()
const toast = useToast()
const issueId = parseInt(route.params.issueId as string, 10) || 0
const projectId = ref(0)
const workspaceId = ref(0)
const projectIdentifier = ref('')
const projectLabels = ref<Array<{ id: number; name: string; color: string }>>([])

useProjectPermissions(projectId)
/** Syncs tab badge with AttachmentManager list without refetching the whole issue. */
const attachmentCountOverride = ref<number | null>(null)

// Reactive state
const issue = ref<any>(null)
const issueForm = ref({ name: '', description: '' })
const states = ref<any[]>([])
const cycles = ref<any[]>([])
const modules = ref<any[]>([])
const releases = ref<{ id: number; name: string; version: string }[]>([])
const projectMembers = ref<any[]>([])
const issueTypes = ref<any[]>([])
const activeTab = ref('details')
const isWatching = ref(false)
const customFieldEntries = ref<Array<{ field: any; value: string | null }>>([])
const { visibleFieldIds: sidebarVisibleFieldIds, scheduleEvaluate, evaluateNow } = useCustomFieldVisibility(workspaceId)
const relationsTabRef = ref<InstanceType<typeof IssueTabRelations> | null>(null)
const agentDispatching = ref(false)
const commentsRefreshKey = ref(0)
const watchIssueId = computed(() => issue.value?.id ?? issueId)
const { pending: autoTriagePending } = useAutoTriageWatch({
  workspaceId,
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
const activeApproval = ref<ApprovalResponse | null>(null)
const issueCopilotLabel = computed(() => {
  if (!issue.value) return ''
  const seq = issue.value.sequence_id
  const name = issue.value.name || ''
  const prefix = projectIdentifier.value ? `${projectIdentifier.value}-` : '#'
  return seq ? `${prefix}${seq} ${name}`.trim() : name
})
const currentUserId = computed(() => {
  // Prefer localStorage 'user_id' (matches existing convention in this file),
  // then fall back to the 'user' JSON blob, then 0.
  const uid = parseInt(localStorage.getItem('user_id') || '0', 10)
  if (uid) return uid
  try {
    const raw = localStorage.getItem('user')
    if (raw) return JSON.parse(raw).id || 0
  } catch {}
  return 0
})

const chatMentionCandidates = computed(() => {
  const users = (projectMembers.value || []).map((m: any) => ({
    id: m.id,
    name: m.display_name || m.username || `User #${m.id}`,
    type: 'user' as const,
  }))
  const agents = workspaceAgents.value
    .filter((a) => a.status === 'active')
    .map((a) => ({
      id: a.id,
      name: a.name,
      type: 'agent' as const,
    }))
  return [...users, ...agents]
})
const showSubmitDialog = ref(false)
const showDecisionDialog = ref(false)
const submitDialogData = ref<{ transitionId: number; fromStateName: string; approveStateName: string; approverNames: string[]; workflowName?: string }>({
  transitionId: 0, fromStateName: '', approveStateName: '', approverNames: []
})
const decisionDialogData = ref<{ approvalId: number; decision: 'approved' | 'rejected' } | null>(null)

// Relation summary for sidebar (loaded independently of Relations tab)
const relationSidebarSummary = ref<{
  total: number
  outbound: number
  inbound: number
  byType: Record<string, { outbound: number; inbound: number }>
} | null>(null)

async function loadRelationSummary() {
  try {
    const relations = await relationApi.listIssueRelations(issueId, 'both')
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

// Tabs definition
const tabs = computed(() => {
  const relationCount = relationSidebarSummary.value?.total ?? 0
  const subCount = Array.isArray(issue.value?.sub_issues) ? issue.value.sub_issues.length : 0
  const relationsTabTotal = relationCount + subCount
  return [
  { key: 'details', label: t('issue.tabDetails'), count: undefined },
  { key: 'relations', label: t('issue.tabRelations'), count: relationsTabTotal > 0 ? relationsTabTotal : undefined },
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
  ]
})

// Props for IssueTabDetails
const detailProps = computed(() => ({
  issueId,
  issue: issue.value,
  workspaceId: workspaceId.value,
  projectId: projectId.value,
  issueTypeId: issue.value?.issue_type?.id,
  members: projectMembers.value,
  commentsRefreshKey: commentsRefreshKey.value,
}))

// Data loading
onMounted(async () => {
  if (!issueId) return
  try {
    const issueData = await issueApi.getIssue(issueId)
    issue.value = issueData
    attachmentCountOverride.value = null
    if (issueData.workspace_id) workspaceId.value = issueData.workspace_id
    if (issueData.project_id) projectId.value = issueData.project_id
    projectIdentifier.value = issueData.project?.identifier || ''
    issueForm.value = {
      name: issueData.name || '',
      description: issueData.description_html || '',
    }

    // Load auxiliary data
    await Promise.all([
      loadStates(),
      loadCycles(),
      loadModules(),
      loadReleases(),
      loadProjectMembers(),
      loadIssueTypes(),
      loadCustomFields(),
      loadLabels(),
      loadWatchers(),
      loadAgentStatus(),
      loadWorkspaceAgents(),
      loadRelationSummary(),
    ])
    // Load active approval if pending
    await loadActiveApproval()
    maybeOpenMergeFromQuery()
  } catch (error) {
    console.error('Failed to load issue:', error)
  }

  const handleKeydown = (e: KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'j') {
      e.preventDefault()
      showAICopilot.value = !showAICopilot.value
    }
  }
  document.addEventListener('keydown', handleKeydown)
  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeydown)
  })
})

async function loadAgentStatus() {
  try {
    const res = await issueAgentApi.getStatus(issueId)
    agentStatus.value = res.data
  } catch (error) {
    console.error('Failed to load agent status:', error)
    agentStatus.value = null
  }
}

async function loadWorkspaceAgents() {
  if (!workspaceId.value) return
  try {
    workspaceAgents.value = await agentApi.list(workspaceId.value)
  } catch (error) {
    console.error('Failed to load agents:', error)
    workspaceAgents.value = []
  }
}

async function loadStates() {
  try {
    const data = await stateApi.listStates(projectId.value)
    const list = Array.isArray(data) ? data : (data as any)?.data || []
    // Hide inactive / E2E junk states from product UI (CORE-03)
    states.value = list.filter((s: any) => {
      if (s?.is_active === false || s?.is_deleted) return false
      const name = String(s?.name || '')
      if (/^E2E\s+Test/i.test(name)) return false
      return true
    })
  } catch (error) {
    console.error('Failed to load states:', error)
  }
}

async function loadCycles() {
  try {
    const data = await cycleApi.listCycles(projectId.value)
    cycles.value = data.items || data
  } catch (error) {
    console.error('Failed to load cycles:', error)
  }
}

async function loadModules() {
  try {
    const data = await moduleApi.listModules(projectId.value, workspaceId.value)
    modules.value = data
  } catch (error) {
    console.error('Failed to load modules:', error)
  }
}

async function loadReleases() {
  try {
    const data = await releaseApi.list(projectId.value)
    releases.value = data
  } catch (error) {
    console.error('Failed to load releases:', error)
  }
}

async function loadIssueTypes() {
  try {
    const types = await issueTypeApi.getIssueTypes(workspaceId.value, projectId.value)
    issueTypes.value = types
  } catch (error) {
    console.error('Failed to load issue types:', error)
  }
}

async function loadProjectMembers() {
  try {
    const data = await projectApi.listProjectMembers(projectId.value)
    projectMembers.value = data.map((m: any) => m.user || m)
  } catch (error) {
    console.error('Failed to load project members:', error)
  }
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
  if (!workspaceId.value || !issue.value) return null
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
    const response = await getIssueCustomFieldsWithDefinitions(issueId)
    if (response?.fields) {
      customFieldEntries.value = response.fields.map((item: any) => ({
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
  } catch (error) {
    console.error('Failed to load custom fields:', error)
  }
}

// When issue type changes, field set must follow the new type bindings.
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
    workspaceId.value,
  ],
  () => {
    scheduleEvaluate(buildSidebarVisibilityBody)
  },
  { deep: true }
)

async function loadLabels() {
  try {
    const resp = await api.get(`/projects/${projectId.value}/settings/labels`)
    projectLabels.value = Array.isArray(resp.data) ? resp.data : (resp.data?.items || [])
  } catch { /* */ }
}

async function handleLabelsUpdate(labelIds: number[]) {
  await instantUpdate('label_ids', labelIds)
}

async function deleteIssue() {
  if (!issue.value || !(await confirm(t('issueDetail.confirmDelete', { 0: issue.value.name })))) return
  try {
    await issueApi.deleteIssue(issueId)
    router.back()
  } catch (e: any) {
    toast.error(e?.response?.data?.message || t('issue.saveFailed'))
  }
}

function openMergeDialog() {
  mergePrefillTargetId.value = null
  showMergeDialog.value = true
}

function closeMergeDialog() {
  showMergeDialog.value = false
  mergePrefillTargetId.value = null
  const q = { ...route.query }
  if (q.mergeTargetId) {
    delete q.mergeTargetId
    router.replace({ query: q })
  }
}

function onMergeSuccess(targetId: number) {
  showMergeDialog.value = false
  mergePrefillTargetId.value = null
  toast.success(t('issue.mergeIntoSuccess'))
  navigateToIssue(targetId)
}

function maybeOpenMergeFromQuery() {
  const raw = route.query.mergeTargetId
  const id = typeof raw === 'string' ? parseInt(raw, 10) : 0
  if (!id || !issue.value) return
  mergePrefillTargetId.value = id
  showMergeDialog.value = true
}

async function updateCustomField(fieldId: number, value: string) {
  try {
    await updateIssueCustomFieldValue(issueId, fieldId, { value })
    const entry = customFieldEntries.value.find((e) => e.field.id === fieldId)
    if (entry) entry.value = value
  } catch (error) {
    console.error('Failed to update custom field:', error)
    toast.error(t('issue.saveFailed'))
  }
}

async function handleTitleSave(title: string) {
  try {
    const updated = await issueApi.updateIssue(issueId, { name: title })
    if (issue.value) Object.assign(issue.value, updated)
    issueForm.value.name = title
  } catch (e: any) {
    toast.error(e?.response?.data?.message || t('issue.saveFailed'))
  }
}

let descTimer: ReturnType<typeof setTimeout> | null = null
async function handleDescriptionSave(html: string) {
  issueForm.value.description = html
  if (descTimer) clearTimeout(descTimer)
  descTimer = setTimeout(async () => {
    try {
      const updated = await issueApi.updateIssue(issueId, { description_html: html || undefined })
      if (issue.value) Object.assign(issue.value, updated)
    } catch (e: any) {
      toast.error(e?.response?.data?.message || t('issue.saveFailed'))
    }
  }, 600)
}

// Instant update for sidebar field changes
async function instantUpdate(field: string, value: any) {
  try {
    const updated = await issueApi.updateIssue(issueId, { [field]: value })
    if (issue.value) Object.assign(issue.value, updated)
  } catch (e: any) {
    console.error('Failed to update field:', field, e)
    toast.error(t('issue.saveFailed'))
  }
}

// Load active approval if issue has pending approval
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

// Intercept state change for approval transitions
async function handleStateChange(newStateId: number) {
  if (!issue.value) return
  if (issue.value.approval_status === 'pending') {
    toast.error(t('approvals.stateDisabledHint'))
    return
  }
  try {
    const updated = await issueApi.updateIssue(issueId, { state_id: newStateId })
    if (issue.value) Object.assign(issue.value, updated)
    await loadActiveApproval()
  } catch (e: any) {
    // Check for approval_required response (HTTP 409)
    if (e?.response?.status === 409 && e?.response?.data?.message === 'approval_required') {
      const transitionId = e.response.data.transition_id
      const sourceStateId = e.response.data.source_state_id
      const targetStateId = e.response.data.target_state_id
      const workflowName = e.response.data.workflow_name
      const sourceStateName = e.response.data.source_state_name
      const targetStateName = e.response.data.target_state_name
      // Fetch transition details from workspace (or project) state workflows — not agent_workflows.
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
        // Parse approver IDs and resolve names from projectMembers
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
      toast.error(t('issue.saveFailed'))
    }
  }
}

// Adopting an agent's proposals changed issue fields server-side, so pull the
// fresh issue back in to keep the header and property sidebar in sync.
async function onSuggestionsApplied() {
  try {
    const updated = await issueApi.getIssue(issueId)
    issue.value = updated
  } catch (e) {
    console.error('Failed to reload issue after applying suggestions:', e)
  }
}

async function onApprovalSubmitted() {
  showSubmitDialog.value = false
  // Reload issue to get updated approval_status
  try {
    const updated = await issueApi.getIssue(issueId)
    issue.value = updated
    await loadActiveApproval()
  } catch (e) {
    console.error('Failed to reload issue:', e)
  }
}

function onDecideApproval(approval: ApprovalResponse, decision: 'approved' | 'rejected') {
  decisionDialogData.value = { approvalId: approval.id, decision }
  showDecisionDialog.value = true
}

async function onApprovalDecided() {
  showDecisionDialog.value = false
  try {
    const updated = await issueApi.getIssue(issueId)
    issue.value = updated
    await loadActiveApproval()
  } catch (e) {
    console.error('Failed to reload issue:', e)
  }
}

async function onCancelApproval(approval: ApprovalResponse) {
  if (!(await confirm(t('approvals.cancelApproval')))) return
  try {
    await approvalApi.cancel(approval.id)
    const updated = await issueApi.getIssue(issueId)
    issue.value = updated
    await loadActiveApproval()
  } catch (e: any) {
    toast.error(e?.response?.data?.message || t('issue.saveFailed'))
  }
}

// Instant update for assignee
async function instantUpdateAssignee(userId: any) {
  try {
    const assigneeIds = userId ? [Number(userId)] : []
    const updated = await issueApi.updateIssue(issueId, { assignee_ids: assigneeIds })
    if (issue.value) Object.assign(issue.value, updated)
  } catch (e: any) {
    console.error('Failed to update assignee:', e)
    toast.error(t('issue.saveFailed'))
  }
}

// Cycle update uses dedicated API
async function handleCycleUpdate(cycleId: number | null) {
  try {
    if (cycleId) {
      await setIssueCycle(issueId, cycleId)
    } else {
      await removeIssueCycle(issueId)
    }
    const updated = await issueApi.getIssue(issueId)
    if (issue.value) Object.assign(issue.value, updated)
  } catch (e: any) {
    console.error('Failed to update cycle:', e)
    toast.error(t('issue.saveFailed'))
  }
}

// Module update — backend Update handles module_ids atomically with delete+recreate
async function handleModuleUpdate(moduleId: number | null) {
  await instantUpdate('module_ids', moduleId ? [moduleId] : [])
}

// Release update
async function handleReleaseUpdate(releaseId: number | null) {
  await instantUpdate('release_id', releaseId ?? 0)
}

async function handleEstimateUpdate(estimatePointId: number | null) {
  await instantUpdate('estimate_point_id', estimatePointId ?? 0)
}

// Navigation
function goBack() {
  router.back()
}

function navigateToIssue(id: number) {
  router.push({
    path: `/workspace/${route.params.slug}/project/${projectId.value}/issues/${id}`,
  })
}

async function handleRelationsRefresh() {
  try {
    const data = await issueApi.getIssue(issueId)
    if (issue.value) {
      Object.assign(issue.value, data)
    }
    await loadRelationSummary()
    await relationsTabRef.value?.loadLinkedPages?.()
  } catch (err) {
    console.error('Failed to refresh issue after relations change:', err)
  }
}

async function loadWatchers() {
  try {
    const { watchers } = await issueApi.listWatchers(issueId)
    const currentUserId = parseInt(localStorage.getItem('user_id') || '0', 10)
    isWatching.value = watchers.includes(currentUserId)
  } catch (error) {
    console.error('Failed to load watchers:', error)
  }
}

async function handleToggleWatch() {
  try {
    if (isWatching.value) {
      await issueApi.removeWatcher(issueId)
      isWatching.value = false
      toast.success(t('issue.unwatch'))
    } else {
      await issueApi.addWatcher(issueId)
      isWatching.value = true
      toast.success(t('issue.watch'))
    }
  } catch (error: any) {
    console.error('Failed to toggle watch:', error)
    toast.error(error?.response?.data?.message || t('common.error'))
  }
}

// Agent assign / unassign / dispatch
async function assignAgent(agentSelectorId: string) {
  if (!agentSelectorId || !agentSelectorId.startsWith('agent:')) return
  const agentId = parseInt(agentSelectorId.replace('agent:', ''), 10)
  if (!agentId) return
  agentAssigning.value = true
  try {
    await issueAgentApi.assign(issueId, { agent_id: agentId })
    await loadAgentStatus()
    toast.success(t('agent.assignSuccess'))
  } catch (e: any) {
    toast.error(e?.response?.data?.message || e?.message || t('common.error'))
  } finally {
    agentAssigning.value = false
  }
}

async function unassignAgent() {
  agentAssigning.value = true
  try {
    await issueAgentApi.unassign(issueId)
    await loadAgentStatus()
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
  const agentId = parseInt(agentSelectorId.replace('agent:', ''), 10)
  if (!agentId || agentDispatching.value) return
  agentDispatching.value = true
  setTimeout(() => { agentDispatching.value = false }, DISPATCH_BUTTON_GUARD_MS)
  const agents = await agentApi.list(workspaceId.value).catch(() => [])
  const agent = agents.find(a => a.id === agentId)
  const task = buildDispatchTask(
    agent,
    { seq: String(issue.value?.sequence_id || issueId), name: issue.value?.name || '' },
    t,
  )
  agentApi
    .dispatch(workspaceId.value, agentId, {
      task,
      issue_id: issueId,
      project_id: projectId.value,
    })
    .then(() => loadAgentStatus())
    .catch((e: any) => {
      toast.error(e?.response?.data?.message || e?.message || 'Failed to dispatch agent')
    })
  toast.success(t('agent.dispatch'))
  commentsRefreshKey.value++
}
</script>

<style scoped>
.issue-detail-page {
  min-height: 100vh;
}
</style>
