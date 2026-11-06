<template>
  <div class="space-y-4">
    <!-- Sub-issues card -->
    <RelationSubIssuesCard
      :sub-issues="subIssues"
      :issue-types="issueTypes"
      :parent-issue-type="currentIssueType"
      :can-quick-create="canQuickCreateSubIssue"
      :can-link-existing="canLinkExistingSubIssue"
      :hierarchy-error="hierarchyError"
      @navigate="(id: number) => emit('navigate', id)"
      @add-existing="openSubIssuePicker"
      @quick-create="handleQuickCreateSubIssue"
      @toggle="handleToggleSubIssue"
      @dismiss-hierarchy-error="hierarchyError = null"
    />

    <IssueLinkedPagesCard
      ref="linkedPagesCardRef"
      :issue-id="issueId"
      :project-id="projectId"
      :slug="slug"
    />

    <!-- Parent issue link -->
    <div v-if="parent" class="flex items-center gap-2 text-xs text-gray-500 bg-white border border-gray-200 rounded-lg px-3 py-2">
      <span class="text-gray-400">{{ t('issue.parentIssue') }}:</span>
      <span class="w-1.5 h-1.5 rounded-full" :style="{ backgroundColor: stateColor(parent.state_group || '') }"></span>
      <span class="font-medium text-gray-700 cursor-pointer hover:text-indigo-600" @click="emit('navigate', parent.id)">
        #{{ parent.sequence_id }} {{ parent.name }}
      </span>
    </div>

    <!-- Global add-relation button -->
    <div class="flex items-center justify-between">
      <span class="text-sm font-semibold text-gray-700">
        {{ t('issue.relations') }}
        <span v-if="allRelations.length > 0" class="ml-1.5 text-xs bg-gray-100 text-gray-500 px-1.5 py-0.5 rounded-full">{{ allRelations.length }}</span>
      </span>
      <div class="relative">
        <button
          data-test="add-relation-main"
          class="text-xs font-medium text-indigo-600 hover:text-indigo-700 hover:bg-indigo-50 px-2 py-1 rounded transition-colors"
          @click="showTypeMenu = !showTypeMenu"
        >+ {{ t('issue.addRelation') }}</button>
        <div
          v-if="showTypeMenu"
          class="absolute right-0 top-full mt-1 w-48 bg-white border border-gray-200 rounded-lg shadow-lg z-20 py-1"
        >
          <button
            v-for="rt in relationTypes"
            :key="rt.id"
            class="w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 flex items-center gap-2 transition-colors"
            @click="openRelationPickerForType(rt.id)"
          >
            <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: getTypeColor(rt.id) }"></span>
            {{ rt.outward_name || rt.name }}
          </button>
          <div v-if="relationTypes.length === 0" class="px-3 py-4 text-xs text-gray-400 text-center">
            {{ t('issueKanban.noRelationTypes') }}
          </div>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div v-if="allRelations.length === 0" class="bg-white border border-gray-200 rounded-lg px-4 py-8 text-center text-sm text-gray-400">
      {{ t('issueKanban.noRelations') }}
    </div>

    <!-- Grouped tables: one table per relation type -->
    <div
      v-for="group in relationGroups"
      :key="group.typeId"
      class="bg-white border border-gray-200 rounded-lg overflow-hidden"
    >
      <!-- Group header -->
      <div class="flex items-center justify-between px-3 py-2 border-b bg-gray-50/50" :style="{ borderColor: group.color + '30' }">
        <div class="flex items-center gap-2">
          <span class="w-2 h-2 rounded-full shrink-0" :style="{ backgroundColor: group.color }"></span>
          <span data-test="relation-group-header" class="text-xs font-semibold text-gray-700">{{ group.typeName }}</span>
          <span class="text-[10px] text-gray-400 bg-gray-200 px-1.5 py-0.5 rounded-full">{{ group.items.length }}</span>
        </div>
        <button
          class="text-[10px] text-indigo-500 hover:text-indigo-600 font-medium"
          @click="openRelationPickerForType(group.typeId)"
        >+ {{ t('issue.addRelation') }}</button>
      </div>

      <!-- Table header -->
      <table class="w-full text-[10px]">
        <thead>
          <tr class="border-b bg-gray-50 text-gray-500">
            <th class="px-3 py-1.5 text-left w-8">{{ t('issue.direction') }}</th>
            <th class="px-2 py-1.5 text-left">{{ t('issue.relationType') }}</th>
            <th class="px-2 py-1.5 text-left">ID</th>
            <th class="px-2 py-1.5 text-left">{{ t('issue.title') }}</th>
            <th class="px-2 py-1.5 text-left">{{ t('issue.type') }}</th>
            <th class="px-2 py-1.5 text-left">{{ t('issue.status') }}</th>
            <th class="px-2 py-1.5 text-left">{{ t('issue.priority') }}</th>
            <th class="px-2 py-1.5 text-left">{{ t('issue.assignee') }}</th>
            <th class="px-2 py-1.5 w-8"></th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="rel in group.items"
            :key="`${rel.direction}-${rel.id}`"
            class="border-b border-gray-50 last:border-b-0 hover:bg-gray-50/50 transition-colors group"
          >
            <!-- Direction arrow -->
            <td class="px-3 py-2">
              <span
                class="text-xs font-bold"
                :class="rel.direction === 'inbound' ? 'text-amber-500' : 'text-blue-500'"
                :title="rel.direction === 'inbound' ? t('issue.inboundRelation') : t('issue.outboundRelation')"
              >{{ rel.direction === 'inbound' ? '←' : '→' }}</span>
            </td>

            <!-- Relation type badge -->
            <td class="px-2 py-2">
              <span
                class="px-1.5 py-0.5 rounded text-[10px] font-medium whitespace-nowrap"
                :style="{ backgroundColor: group.color + '20', color: group.color }"
              >{{ rel.direction === 'inbound' ? (rel.inward_name || rel.relation_type?.name) : (rel.relation_type?.outward_name || rel.relation_type?.name) }}</span>
            </td>

            <!-- ID -->
            <td class="px-2 py-2 font-mono text-gray-400 whitespace-nowrap">#{{ rel.related_issue?.sequence_id }}</td>

            <!-- Title (clickable) -->
            <td class="px-2 py-2">
              <span
                class="text-xs font-medium text-gray-800 cursor-pointer hover:text-indigo-600 block truncate max-w-[200px]"
                @click="$emit('navigate', rel.related_issue?.id || rel.related_issue_id)"
              >{{ rel.related_issue?.name || rel.related_name }}</span>
            </td>

            <!-- Issue type -->
            <td class="px-2 py-2">
              <span
                v-if="getRelatedIssueType(rel)?.name"
                class="px-1.5 py-0.5 rounded text-[10px] font-medium whitespace-nowrap"
                :style="{ backgroundColor: (getRelatedIssueType(rel)?.color || '#e5e7eb') + '20', color: getRelatedIssueType(rel)?.color || '#6b7280' }"
              >{{ getRelatedIssueType(rel)?.name }}</span>
              <span v-else class="text-gray-300">—</span>
            </td>

            <!-- State -->
            <td class="px-2 py-2">
              <span class="flex items-center gap-1 whitespace-nowrap">
                <span class="w-1.5 h-1.5 rounded-full" :style="{ backgroundColor: stateColor(rel.related_issue?.state_group || '') }"></span>
                <span class="text-xs text-gray-600">{{ rel.related_issue?.state_name || '—' }}</span>
              </span>
            </td>

            <!-- Priority -->
            <td class="px-2 py-2">
              <span class="flex items-center gap-1 whitespace-nowrap" v-if="rel.related_issue?.priority && rel.related_issue.priority !== 'none'">
                <span class="w-1.5 h-1.5 rounded-full" :style="{ backgroundColor: priorityColor(rel.related_issue?.priority || 'none') }"></span>
                <span class="text-xs text-gray-600">{{ t(`issue.priority${(rel.related_issue?.priority || 'none').charAt(0).toUpperCase() + (rel.related_issue?.priority || 'none').slice(1)}`) }}</span>
              </span>
              <span v-else class="text-gray-300">—</span>
            </td>

            <!-- Assignee -->
            <td class="px-2 py-2 text-xs text-gray-500 max-w-[80px] truncate">
              {{ rel.related_issue?.assignees?.[0]?.display_name || rel.related_issue?.assignees?.[0]?.username || '—' }}
            </td>

            <!-- Remove -->
            <td class="px-2 py-2 text-center">
              <button
                data-test="remove-relation"
                class="text-gray-300 hover:text-red-500 text-sm leading-none opacity-0 group-hover:opacity-100 transition-opacity"
                @click="removeRelation(rel.id)"
              >×</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Issue picker dialog -->
    <IssuePickerDialog
      :visible="pickerVisible"
      :project-id="projectId"
      :exclude-id="issueId"
      :title="pickerTitle"
      @close="pickerVisible = false"
      @select="onPickerSelect"
    />

    <!-- Click-away overlay for type menu -->
    <div v-if="showTypeMenu" class="fixed inset-0 z-10" @click="showTypeMenu = false"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast'
import { useProjectPermissions } from '@/composables/usePermission'
import { listIssueRelations, createIssueRelation, deleteIssueRelation, listRelationTypes } from '@/api/relation'
import { createIssue, updateIssue } from '@/api/issue'
import IssuePickerDialog from './IssuePickerDialog.vue'
import IssueLinkedPagesCard from './IssueLinkedPagesCard.vue'
import RelationSubIssuesCard from './RelationSubIssuesCard.vue'
import { stateColor, priorityColor } from '@/composables/useRelationHelpers'
import { IssuePriority } from '@/types/issue'

const COLORS = ['#6366f1', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316']

interface IssueType {
  id: number
  name: string
  color: string
}

interface RelationItem {
  id: number
  direction: string
  issue_id: number
  related_issue_id: number
  relation_type_id: number
  comment?: string | null
  relation_type?: { id: number; name: string; outward_name: string }
  inward_name?: string
  outward_name?: string
  relation_name?: string
  related_name?: string
  related_issue?: {
    id: number
    sequence_id: number
    name: string
    state_name: string
    state_group: string
    priority: string
    assignees?: Array<{ id: number; display_name?: string; username?: string }>
    target_date?: string | null
    issue_type?: IssueType
  }
}

interface RelationGroup {
  typeId: number
  typeName: string
  items: RelationItem[]
  color: string
}

const props = defineProps<{
  issueId: number
  projectId: number
  workspaceId: number
  slug: string
  states: Array<{ id: number; name: string; group: string }>
  parent: any | null
  subIssues: any[]
  issueTypes: IssueType[]
  currentIssueType?: { id: number; level: number } | null
}>()

const emit = defineEmits<{
  navigate: [issueId: number]
  refresh: []
}>()

const { t } = useI18n()
const toast = useToast()
const { can } = useProjectPermissions(() => props.projectId)
// POST /issues uses issue:create; setting parent on another issue uses issue:edit.
const canQuickCreateSubIssue = computed(() => can('issue:create'))
const canLinkExistingSubIssue = computed(() => can('issue:edit'))

const hierarchyError = ref<string | null>(null)

function getApiErrorMessage(err: unknown): string {
  const e = err as { response?: { data?: { message?: string; error?: string } }; message?: string }
  const msg = e?.response?.data?.message || e?.response?.data?.error
  if (typeof msg === 'string' && msg.trim()) return msg
  if (typeof e?.message === 'string' && e.message.trim()) return e.message
  return t('issue.saveFailed')
}

const allRelations = ref<RelationItem[]>([])
const relationTypes = ref<Array<{ id: number; name: string; inward_name: string; outward_name: string }>>([])

// Picker state
const pickerVisible = ref(false)
const pickerTitle = ref('')
const pickerMode = ref<'relation' | 'subissue'>('relation')
const pendingRelationTypeId = ref(0)

// Type menu
const showTypeMenu = ref(false)

const linkedPagesCardRef = ref<InstanceType<typeof IssueLinkedPagesCard> | null>(null)

async function loadLinkedPages() {
  await linkedPagesCardRef.value?.loadLinkedPages()
}

// Group relations by type - only types that have relations
const relationGroups = computed<RelationGroup[]>(() => {
  const groups: Map<number, RelationGroup> = new Map()

  // First pass: collect relation types that actually have items
  for (const rel of allRelations.value) {
    const rtId = rel.relation_type_id
    if (!groups.has(rtId)) {
      const rt = relationTypes.value.find(t => t.id === rtId)
      const idx = relationTypes.value.findIndex(t => t.id === rtId)
      groups.set(rtId, {
        typeId: rtId,
        typeName: rt?.name || rt?.outward_name || rel.relation_type?.name || rel.relation_name || '',
        items: [],
        color: COLORS[idx >= 0 ? idx % COLORS.length : 0],
      })
    }
    groups.get(rtId)!.items.push(rel)
  }

  // Sort: outbound first, then by type color index for consistency
  return Array.from(groups.values())
})

// Counts for sidebar
const outboundCount = computed(() => allRelations.value.filter(r => r.direction === 'outbound').length)
const inboundCount = computed(() => allRelations.value.filter(r => r.direction === 'inbound').length)

// Relation summary for external use (sidebar)
const relationSummary = computed(() => {
  const byType: Record<string, { outbound: number; inbound: number }> = {}
  for (const rt of relationTypes.value) {
    byType[rt.name] = { outbound: 0, inbound: 0 }
  }
  for (const rel of allRelations.value) {
    const typeName = rel.relation_type?.name || rel.relation_name || ''
    if (!byType[typeName]) byType[typeName] = { outbound: 0, inbound: 0 }
    if (rel.direction === 'outbound') byType[typeName].outbound++
    else byType[typeName].inbound++
  }
  return {
    total: allRelations.value.length,
    outbound: outboundCount.value,
    inbound: inboundCount.value,
    byType,
  }
})

function getTypeColor(typeId: number): string {
  const idx = relationTypes.value.findIndex(rt => rt.id === typeId)
  return COLORS[idx >= 0 ? idx % COLORS.length : 0]
}

function getRelatedIssueType(rel: RelationItem): IssueType | null {
  if (!props.issueTypes || !rel.related_issue) return null
  return props.issueTypes.find(t => t.id === rel.related_issue?.issue_type?.id) || rel.related_issue?.issue_type || null
}

onMounted(async () => {
  await Promise.all([loadRelationTypes(), loadRelations()])
})

async function loadRelationTypes() {
  try {
    const types = await listRelationTypes(props.workspaceId)
    relationTypes.value = types || []
  } catch (err) {
    console.error('Failed to load relation types:', err)
  }
}

async function loadRelations() {
  try {
    const rels = await listIssueRelations(props.issueId, 'both')
    allRelations.value = rels || []
  } catch (err) {
    console.error('Failed to load relations:', err)
  }
}

// ---- Relation operations ----
function openRelationPickerForType(relationTypeId: number) {
  showTypeMenu.value = false
  const rt = relationTypes.value.find(t => t.id === relationTypeId)
  pickerTitle.value = rt ? `${t('issue.addRelation')} - ${rt.outward_name || rt.name}` : t('issue.addRelation')
  pendingRelationTypeId.value = relationTypeId
  pickerMode.value = 'relation'
  pickerVisible.value = true
}

function openSubIssuePicker() {
  pickerTitle.value = t('subIssue.selectExisting')
  pickerMode.value = 'subissue'
  pickerVisible.value = true
}

async function onPickerSelect(selectedId: number) {
  pickerVisible.value = false
  if (pickerMode.value === 'subissue') {
    await addSubIssue(selectedId)
  } else {
    await addRelation(selectedId, pendingRelationTypeId.value)
  }
}

async function addSubIssue(childId: number) {
  try {
    hierarchyError.value = null
    await updateIssue(childId, { parent_id: props.issueId })
    emit('refresh')
  } catch (err) {
    console.error('Failed to set parent issue:', err)
    const msg = getApiErrorMessage(err)
    hierarchyError.value = msg
    toast.error(msg)
  }
}

async function handleQuickCreateSubIssue(name: string, typeId?: number) {
  try {
    hierarchyError.value = null
    await createIssue(props.projectId, props.workspaceId, {
      name,
      parent_id: props.issueId,
      priority: IssuePriority.NONE,
      ...(typeId ? { type_id: typeId } : {}),
    })
    emit('refresh')
  } catch (err) {
    console.error('Failed to create sub-issue:', err)
    const msg = getApiErrorMessage(err)
    hierarchyError.value = msg
    toast.error(msg)
  }
}

async function handleToggleSubIssue(subIssueId: number) {
  const sub = props.subIssues.find((s: any) => s.id === subIssueId)
  if (!sub) return
  try {
    const doneGroups = ['done', 'cancelled', 'completed']
    const isDone = doneGroups.includes(sub.state_group)
    const targetGroup = isDone ? 'started' : 'done'
    const targetState = props.states.find((s) => s.group === targetGroup) || props.states[0]
    if (targetState) {
      await updateIssue(subIssueId, { state_id: targetState.id })
      emit('refresh')
    }
  } catch (err) {
    console.error('Failed to toggle sub-issue:', err)
    toast.error(t('issue.saveFailed'))
  }
}

async function addRelation(relatedIssueId: number, relationTypeId: number) {
  try {
    await createIssueRelation(props.issueId, {
      related_issue_id: relatedIssueId,
      relation_type_id: relationTypeId,
    })
    await loadRelations()
  } catch (err) {
    console.error('Failed to create relation:', err)
    toast.error(t('issue.saveFailed'))
  }
}

async function removeRelation(relationId: number) {
  try {
    await deleteIssueRelation(relationId)
    allRelations.value = allRelations.value.filter(r => r.id !== relationId)
  } catch (err) {
    console.error('Failed to remove relation:', err)
  }
}

defineExpose({ relationSummary, loadRelations, loadLinkedPages })
</script>
