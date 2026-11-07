<template>
  <div class="issue-kanban">
    <!-- 看板布局控制工具栏 (groupBy / swimlaneBy 是视图布局，不是筛选) -->
    <div class="bg-white rounded-lg border border-gray-200 mb-4 px-4 py-2.5">
      <div class="flex items-center gap-3 flex-wrap">
        <div class="flex items-center gap-1.5">
          <label class="text-xs text-gray-500">{{ t('issueKanban.groupBy') }}</label>
          <select v-model="groupBy" class="px-2 py-1 border border-gray-300 rounded-md text-sm text-gray-600">
            <option value="state">{{ t('issueKanban.groupByState') }}</option>
            <option value="assignee">{{ t('issueKanban.groupByAssignee') }}</option>
            <option value="priority">{{ t('issueKanban.groupByPriority') }}</option>
            <option value="labels">{{ t('issueKanban.groupByLabels') }}</option>
          </select>
        </div>
        <div class="flex items-center gap-1.5">
          <label class="text-xs text-gray-500">{{ t('issueKanban.swimlane') }}</label>
          <select v-model="swimlaneBy" class="px-2 py-1 border border-gray-300 rounded-md text-sm text-gray-600">
            <option value="">{{ t('issueKanban.noSwimlane') }}</option>
            <option value="assignee">{{ t('issueKanban.swimlaneAssignee') }}</option>
            <option value="priority">{{ t('issueKanban.swimlanePriority') }}</option>
            <option value="type">{{ t('issueKanban.swimlaneType') }}</option>
          </select>
        </div>
        <div class="flex-1" />
        <button @click="showImportModal = true" class="px-3 py-1.5 text-sm text-gray-600 border border-gray-300 rounded-md hover:bg-gray-50">{{ t('common.import') }}</button>
      </div>
    </div>

    <!-- 批量操作工具栏 -->
    <div v-if="selectedIds.size > 0" class="sticky top-0 z-20 bg-indigo-50 dark:bg-indigo-900/20 border border-indigo-200 dark:border-indigo-800 rounded-lg px-4 py-2 flex items-center gap-3 mb-3 flex-wrap">
      <span class="text-sm text-indigo-700 dark:text-indigo-300 font-medium">{{ t('issueList.selected') }} {{ selectedIds.size }} {{ t('common.items') }}</span>

      <div class="relative">
        <button @click="showBatchState = !showBatchState" class="px-2.5 py-1 text-xs border border-indigo-300 dark:border-indigo-700 rounded-md bg-white dark:bg-gray-700 dark:text-gray-200 hover:bg-indigo-50 dark:hover:bg-gray-600 transition-colors">
          {{ t('issueList.changeState') }}
        </button>
        <div v-if="showBatchState" class="absolute left-0 top-full mt-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg z-30 py-1 w-32">
          <button v-for="s in states" :key="s.id" @click="batchChangeState(s.id)" class="w-full text-left px-3 py-1.5 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 dark:text-gray-200">{{ s.name }}</button>
        </div>
      </div>

      <div class="relative">
        <button @click="showBatchPriority = !showBatchPriority" class="px-2.5 py-1 text-xs border border-indigo-300 dark:border-indigo-700 rounded-md bg-white dark:bg-gray-700 dark:text-gray-200 hover:bg-indigo-50 dark:hover:bg-gray-600 transition-colors">
          {{ t('issueList.changePriority') }}
        </button>
        <div v-if="showBatchPriority" class="absolute left-0 top-full mt-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg z-30 py-1 w-32">
          <button v-for="p in priorityOptions" :key="p.value" @click="batchChangePriority(p.value)" class="w-full text-left px-3 py-1.5 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 dark:text-gray-200">{{ p.label }}</button>
        </div>
      </div>

      <div class="relative">
        <button @click="showBatchAssign = !showBatchAssign" class="px-2.5 py-1 text-xs border border-indigo-300 dark:border-indigo-700 rounded-md bg-white dark:bg-gray-700 dark:text-gray-200 hover:bg-indigo-50 dark:hover:bg-gray-600 transition-colors">
          {{ t('issueList.batchAssign') }}
        </button>
        <div v-if="showBatchAssign" class="absolute left-0 top-full mt-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg z-30 p-2 w-48">
          <UserSelect v-model="batchAssigneeId" :users="memberOptions" :placeholder="t('issueList.selectAssignee')" @update:model-value="batchAssign" />
        </div>
      </div>

      <button v-if="canDeleteIssues" @click="execBatchDelete" class="px-2.5 py-1 text-xs border border-red-300 dark:border-red-700 rounded-md bg-white dark:bg-gray-700 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/30 transition-colors">
        {{ t('issueList.batchDelete') }}
      </button>

      <button @click="clearSelection" class="text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300">{{ t('issueList.clearSelection') }}</button>
    </div>

    <!-- Success toast -->
    <div v-if="toastMessage" class="fixed top-4 right-4 z-50 bg-green-50 dark:bg-green-900/50 border border-green-200 dark:border-green-800 text-green-700 dark:text-green-300 text-sm px-4 py-2 rounded-md shadow-lg transition-opacity">
      {{ toastMessage }}
    </div>

    <!-- 看板列 -->
    <div v-if="loading" class="text-center py-12 text-gray-400 text-sm">{{ t('common.loading') }}</div>

    <!-- 无泳道模式 -->
    <div v-else-if="!swimlaneBy" class="grid gap-4" :style="gridStyle">
      <div
        v-for="column in kanbanColumns"
        :key="column.id"
        class="bg-gray-100 dark:bg-gray-800 rounded-lg p-3 min-h-[200px]"
      >
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center space-x-2">
            <span class="w-2.5 h-2.5 rounded-full" :style="{ backgroundColor: column.color }"></span>
            <h3 class="text-sm font-medium text-gray-700">{{ column.label }}</h3>
          </div>
          <div class="flex items-center space-x-1">
            <button v-if="canCreateIssue && groupBy === 'state'" @click="openQuickCreate(column.key as number)" class="w-5 h-5 flex items-center justify-center text-gray-500 hover:text-indigo-600 hover:bg-gray-200 rounded text-sm" :title="t('issueKanban.quickCreate')">+</button>
            <span class="text-xs bg-gray-300 text-gray-600 px-1.5 py-0.5 rounded-full">{{ (groupedIssues[column.key] || []).length }}</span>
          </div>
        </div>
        <div v-if="canCreateIssue && groupBy === 'state' && quickCreateStateId === column.key" class="mb-3">
          <QuickCreateInput :project-id="projectId" :workspace-id="workspaceId" :issue-types="issueTypes" :default-type-id="defaultTypeId ?? undefined" :default-state-id="(column.key as number)" :show-priority="false" inline show-cancel @created="onQuickCreated" @cancel="closeQuickCreate" />
        </div>
        <VueDraggable
          v-model="groupedIssues[column.key]"
          :group="{ name: dragGroupName, pull: true, put: true }"
          :animation="300"
          ghost-class="kanban-ghost"
          drag-class="kanban-drag"
          :sort="true"
          :disabled="dragLocked"
          item-key="id"
          class="space-y-2 min-h-[4px]"
          @update="(evt: any) => onDragUpdate(column.key, evt.newIndex, evt.oldIndex, evt)"
          @add="(evt: any) => onDragAdd(column.key, evt.newIndex, evt)"
        >
          <div
            v-for="issue in groupedIssues[column.key] || []"
            :key="issue.id"
            :data-issue-id="issue.id"
            @click="$emit('select', issue)"
            class="bg-white dark:bg-gray-700 rounded-md border border-gray-200 dark:border-gray-600 p-2.5 cursor-pointer hover:shadow-md hover:border-indigo-300 dark:hover:border-indigo-600 transition-shadow relative group"
          >
            <div class="absolute top-1.5 right-1.5 opacity-0 group-hover:opacity-100 transition-opacity" @click.stop>
              <input type="checkbox" :checked="selectedIds.has(issue.id)" @change="toggleSelect(issue.id)" class="rounded border-gray-300 dark:border-gray-500 w-3.5 h-3.5" />
            </div>
            <div class="flex items-center gap-2">
              <div class="flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity cursor-grab text-gray-400" title="Drag to reorder">
                <svg class="w-3 h-3" viewBox="0 0 12 12" fill="currentColor"><circle cx="3" cy="2" r="1.2"/><circle cx="9" cy="2" r="1.2"/><circle cx="3" cy="6" r="1.2"/><circle cx="9" cy="6" r="1.2"/><circle cx="3" cy="10" r="1.2"/><circle cx="9" cy="10" r="1.2"/></svg>
              </div>
              <div class="flex-1 min-w-0">
                <div class="flex items-start justify-between">
                  <span class="text-xs text-gray-400 font-mono">{{ projectIdentifier }}-{{ issue.sequence_id }}</span>
                  <span :class="priorityDotClass(issue.priority)" class="w-1.5 h-1.5 rounded-full inline-block"></span>
                </div>
                <p class="text-sm text-gray-800 dark:text-gray-100 mt-1 leading-snug line-clamp-2">{{ issue.name }}</p>
                <div class="mt-2 flex items-center justify-between">
                  <div class="flex -space-x-1">
                    <div v-for="(a, idx) in (issue.assignees || []).slice(0, 2)" :key="a.id" class="w-5 h-5 rounded-full border border-white flex items-center justify-center text-[10px] font-medium text-white" :style="{ backgroundColor: assigneeColor(idx) }" :title="a.display_name || a.username">{{ getInitials(a.display_name || a.username) }}</div>
                  </div>
                  <span v-if="issue.cycle" class="text-[10px] text-gray-400 bg-gray-100 px-1.5 py-0.5 rounded">{{ issue.cycle.name }}</span>
                  <button @click.stop="$emit('select', issue)" class="text-[10px] text-indigo-500 hover:text-indigo-700 font-medium">{{ t('issueKanban.details') }}</button>
                </div>
              </div>
            </div>
          </div>
        </VueDraggable>
        <div v-if="!(groupedIssues[column.key] || []).length" class="text-center text-xs text-gray-400 py-6">{{ t('issueKanban.dragHere') }}</div>
      </div>
    </div>

    <!-- 泳道模式 -->
    <div v-else class="swimlane-board space-y-6">
      <div
        v-for="swimlane in swimlaneKeys"
        :key="swimlane.key"
        class="swimlane-row"
      >
        <div class="flex items-center space-x-2 mb-2 px-1">
          <span class="w-3 h-3 rounded-full flex-shrink-0" :style="{ backgroundColor: swimlane.color }"></span>
          <span class="text-sm font-semibold text-gray-700 dark:text-gray-200">{{ swimlane.label }}</span>
          <span class="text-xs text-gray-400">{{ countSwimlaneIssues(swimlane.key) }} {{ t('common.items') }}</span>
        </div>
        <div class="grid gap-3" :style="gridStyle">
          <div
            v-for="column in kanbanColumns"
            :key="column.id"
            class="bg-gray-100 dark:bg-gray-800 rounded-lg p-3 min-h-[120px]"
          >
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center space-x-1.5">
                <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: column.color }"></span>
                <h4 class="text-xs font-medium text-gray-500">{{ column.label }}</h4>
              </div>
              <span class="text-[10px] bg-gray-300 text-gray-600 px-1.5 py-0.5 rounded-full">
                {{ (swimlaneGrouped[swimlane.key]?.[column.key] || []).length }}
              </span>
            </div>
            <VueDraggable
              v-if="swimlaneGrouped[swimlane.key]"
              v-model="swimlaneGrouped[swimlane.key][column.key]"
              :group="{ name: dragGroupName, pull: true, put: true }"
              :animation="300"
              ghost-class="kanban-ghost"
              drag-class="kanban-drag"
              :sort="true"
              :disabled="dragLocked"
              item-key="id"
              class="space-y-1.5 min-h-[4px]"
              @update="(evt: any) => onDragUpdate(column.key, evt.newIndex, evt.oldIndex, evt, swimlane.key)"
              @add="(evt: any) => onDragAdd(column.key, evt.newIndex, evt, swimlane.key)"
            >
              <div
                v-for="issue in swimlaneGrouped[swimlane.key]?.[column.key] || []"
                :key="issue.id"
                :data-issue-id="issue.id"
                @click="$emit('select', issue)"
                class="bg-white dark:bg-gray-700 rounded border border-gray-200 dark:border-gray-600 p-2 cursor-pointer hover:shadow-md hover:border-indigo-300 dark:hover:border-indigo-600 transition-shadow relative group"
              >
                <div class="flex items-center gap-1.5">
                  <div class="flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity cursor-grab text-gray-400" title="Drag to reorder">
                    <svg class="w-2.5 h-2.5" viewBox="0 0 12 12" fill="currentColor"><circle cx="3" cy="2" r="1.2"/><circle cx="9" cy="2" r="1.2"/><circle cx="3" cy="6" r="1.2"/><circle cx="9" cy="6" r="1.2"/><circle cx="3" cy="10" r="1.2"/><circle cx="9" cy="10" r="1.2"/></svg>
                  </div>
                  <div class="flex-1 min-w-0">
                    <div class="flex items-start justify-between">
                      <span class="text-[10px] text-gray-400 font-mono">{{ issue.sequence_id }}</span>
                      <span :class="priorityDotClass(issue.priority)" class="w-1.5 h-1.5 rounded-full inline-block"></span>
                    </div>
                    <p class="text-xs text-gray-800 dark:text-gray-100 mt-0.5 leading-snug line-clamp-2">{{ issue.name }}</p>
                  </div>
                </div>
              </div>
            </VueDraggable>
            <div v-if="!(swimlaneGrouped[swimlane.key]?.[column.key] || []).length" class="text-center text-[10px] text-gray-300 py-4">-</div>
          </div>
        </div>
      </div>
    </div>
    <ImportIssuesModal
      :visible="showImportModal"
      :project-id="projectId"
      :workspace-id="workspaceId"
      @close="showImportModal = false"
      @success="onImportSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from '@/composables/useI18n'
import { useProjectPermissions } from '@/composables/usePermission'
import issueApi from '@/api/issue'
import api from '@/api'
import UserSelect from '@/components/UserSelect.vue'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'
import QuickCreateInput from '@/components/QuickCreateInput.vue'
import { markAutoTriagePending } from '@/utils/autoTriagePref'
import ImportIssuesModal from '@/components/ImportIssuesModal.vue'
import * as issueTypeApi from '@/api/issue-type'
import { VueDraggable } from 'vue-draggable-plus'

const props = defineProps<{ projectId: number; workspaceId: number; projectIdentifier?: string; rql?: string; filterSortBy?: string; filterSortDir?: string; filterSortConfig?: string; filterGroupBy?: string; filterSubGroupBy?: string }>()
const { can: canInProject } = useProjectPermissions(() => props.projectId)
const canDeleteIssues = computed(() => canInProject('issue:delete'))
const canCreateIssue = computed(() => canInProject('issue:create'))
const defaultTypeId = computed(() => {
  const d = issueTypes.value.find((t: any) => t.is_default)
  return d?.id ?? null
})
const emit = defineEmits<{ (e: 'select', issue: any): void }>()
const { t } = useI18n()
const toast = useToast()

const issues = ref<any[]>([])
const states = ref<any[]>([])
const members = ref<any[]>([])
const loading = ref(false)
const groupBy = ref<'state' | 'assignee' | 'priority' | 'labels'>('state')
const swimlaneBy = ref<'assignee' | 'priority' | 'type' | ''>('')

watch(() => props.filterGroupBy, (newGroupBy) => {
  if (newGroupBy && newGroupBy !== 'none') {
    const groupMap: Record<string, 'state' | 'assignee' | 'priority' | 'labels'> = {
      'state_id': 'state',
      'assignee_id': 'assignee',
      'priority': 'priority',
      'label': 'labels',
    }
    groupBy.value = groupMap[newGroupBy] || 'state'
  }
})

watch(() => props.filterSubGroupBy, (newSubGroupBy) => {
  if (newSubGroupBy && newSubGroupBy !== 'none') {
    const swimlaneMap: Record<string, 'assignee' | 'priority' | 'type'> = {
      'assignee_id': 'assignee',
      'priority': 'priority',
      'type_id': 'type',
    }
    swimlaneBy.value = swimlaneMap[newSubGroupBy] || ''
  } else {
    swimlaneBy.value = ''
  }
})

watch(groupBy, () => { rebuildGroupedIssues(); rebuildSwimlaneGrouped() })
watch(swimlaneBy, () => rebuildSwimlaneGrouped())

const dragLocked = ref(false)
const pendingRequests = new Map<number, AbortController>()

function computeSortOrder(targetList: any[], newIndex: number, movingDown: boolean): number {
  const prev = newIndex > 0 ? (targetList[newIndex - 1]?.sort_order ?? null) : null
  const next = newIndex < targetList.length - 1 ? (targetList[newIndex + 1]?.sort_order ?? null) : null

  if (!prev && !next) return 65535
  if (!prev && next) return (next as number) / 2
  if (prev && !next) return (prev as number) + 1000

  const mid = ((prev as number) + (next as number)) / 2
  // When prev === next (all same default), midpoint equals both — no change.
  // Offset by ±0.5 based on direction to produce a distinct sort_order.
  if (mid <= (prev as number) || mid >= (next as number)) {
    return (prev as number) + (movingDown ? 0.5 : -0.5)
  }
  return mid
}

const dragGroupName = computed(() => String(props.projectId))

// ── Batch selection ──
const selectedIds = ref(new Set<number>())
const showBatchState = ref(false)
const showBatchPriority = ref(false)
const showBatchAssign = ref(false)
const batchAssigneeId = ref<number | undefined>(undefined)

const priorityOptions = computed(() => [
  { value: 'urgent', label: t('issue.priorityUrgent') },
  { value: 'high', label: t('issue.priorityHigh') },
  { value: 'medium', label: t('issue.priorityMedium') },
  { value: 'low', label: t('issue.priorityLow') },
  { value: 'none', label: t('issue.priorityNone') },
])

const toastMessage = ref('')
let toastTimer: ReturnType<typeof setTimeout> | null = null

function showToast(msg: string) {
  toastMessage.value = msg
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { toastMessage.value = '' }, 2500)
}

const { confirm } = useConfirm()

const memberOptions = computed(() => members.value.map(m => ({
  id: m.user_id,
  display_name: m.user?.display_name || m.user?.username,
  email: m.user?.email
})))

// ═══ Grouping logic ═══
const groupedIssues = ref<Record<string | number, any[]>>({})

function rebuildGroupedIssues() {
  const map: Record<string | number, any[]> = {}

  if (groupBy.value === 'state') {
    states.value.forEach(s => { map[s.id] = [] })
    issues.value.forEach(i => {
      if (map[i.state_id]) map[i.state_id].push(i)
    })
  } else if (groupBy.value === 'assignee') {
    issues.value.forEach(i => {
      if (i.assignees && i.assignees.length > 0) {
        i.assignees.forEach((a: any) => {
          if (!map[a.id]) map[a.id] = []
        })
      }
    })
    map['__unassigned__'] = []
    issues.value.forEach(i => {
      if (i.assignees && i.assignees.length > 0) {
        i.assignees.forEach((a: any) => {
          map[a.id].push(i)
        })
      } else {
        map['__unassigned__'].push(i)
      }
    })
  } else if (groupBy.value === 'priority') {
    ['urgent', 'high', 'medium', 'low', 'none'].forEach(k => { map[k] = [] })
    issues.value.forEach(i => {
      const key = i.priority || 'none'
      if (map[key]) map[key].push(i)
    })
  } else if (groupBy.value === 'labels') {
    issues.value.forEach(i => {
      if (i.labels && i.labels.length > 0) {
        i.labels.forEach((l: any) => {
          if (!map[l.id]) map[l.id] = []
        })
      }
    })
    map['__nolabel__'] = []
    issues.value.forEach(i => {
      if (i.labels && i.labels.length > 0) {
        i.labels.forEach((l: any) => {
          map[l.id].push(i)
        })
      } else {
        map['__nolabel__'].push(i)
      }
    })
  }

  groupedIssues.value = map
}

const kanbanColumns = computed(() => {
  if (groupBy.value === 'state') {
    return states.value.map(s => ({
      id: 'state_' + s.id,
      label: s.name,
      color: s.color || '#6366f1',
      key: s.id,
    }))
  } else if (groupBy.value === 'assignee') {
    const cols: Array<{ id: string; label: string; color: string; key: string | number }> = []
    const seen = new Set<number>()
    issues.value.forEach(i => {
      if (i.assignees && i.assignees.length > 0) {
        i.assignees.forEach((a: any) => {
          if (!seen.has(a.id)) {
            seen.add(a.id)
            cols.push({
              id: 'user_' + a.id,
              label: a.display_name || a.username,
              color: assigneeColor(cols.length),
              key: a.id,
            })
          }
        })
      }
    })
    cols.push({ id: 'unassigned', label: t('issueKanban.unassigned'), color: '#9ca3af', key: '__unassigned__' })
    return cols
  } else if (groupBy.value === 'priority') {
    return [
      { id: 'priority_urgent', label: t('issue.priorityUrgent'), color: '#ef4444', key: 'urgent' },
      { id: 'priority_high', label: t('issue.priorityHigh'), color: '#f97316', key: 'high' },
      { id: 'priority_medium', label: t('issue.priorityMedium'), color: '#eab308', key: 'medium' },
      { id: 'priority_low', label: t('issue.priorityLow'), color: '#22c55e', key: 'low' },
      { id: 'priority_none', label: t('issue.priorityNone'), color: '#9ca3af', key: 'none' },
    ]
  } else {
    const cols: Array<{ id: string; label: string; color: string; key: string | number }> = []
    const seen = new Set<number>()
    issues.value.forEach(i => {
      if (i.labels && i.labels.length > 0) {
        i.labels.forEach((l: any) => {
          if (!seen.has(l.id)) {
            seen.add(l.id)
            cols.push({
              id: 'label_' + l.id,
              label: l.name,
              color: l.color || '#6366f1',
              key: l.id,
            })
          }
        })
      }
    })
    cols.push({ id: 'nolabel', label: t('issueKanban.unassigned'), color: '#9ca3af', key: '__nolabel__' })
    return cols
  }
})

const gridStyle = computed(() => ({
  gridTemplateColumns: `repeat(${kanbanColumns.value.length}, minmax(260px, 1fr))`
}))

// ── Swimlane support ──
const swimlaneKeys = computed(() => {
  if (!swimlaneBy.value) return []

  if (swimlaneBy.value === 'assignee') {
    const seen = new Set<string>()
    const keys: Array<{ key: string; label: string; color: string }> = []
    issues.value.forEach(i => {
      if (i.assignees && i.assignees.length > 0) {
        i.assignees.forEach((a: any) => {
          const k = String(a.id)
          if (!seen.has(k)) {
            seen.add(k)
            keys.push({ key: k, label: a.display_name || a.username, color: assigneeColor(keys.length) })
          }
        })
      }
    })
    keys.push({ key: '__none__', label: t('issueKanban.unassigned'), color: '#9ca3af' })
    return keys
  }

  if (swimlaneBy.value === 'priority') {
    return [
      { key: 'urgent', label: t('issue.priorityUrgent'), color: '#ef4444' },
      { key: 'high', label: t('issue.priorityHigh'), color: '#f97316' },
      { key: 'medium', label: t('issue.priorityMedium'), color: '#eab308' },
      { key: 'low', label: t('issue.priorityLow'), color: '#22c55e' },
      { key: 'none', label: t('issue.priorityNone'), color: '#9ca3af' },
    ]
  }

  if (swimlaneBy.value === 'type') {
    const seen = new Set<number>()
    const keys: Array<{ key: string; label: string; color: string }> = []
    issues.value.forEach(i => {
      const tid = i.issue_type_id
      if (tid && !seen.has(tid)) {
        seen.add(tid)
        const it = issueTypes.value.find((type: any) => type.id === tid)
        keys.push({ key: String(tid), label: it?.name || t('issueKanban.unassigned'), color: it?.color || '#6366f1' })
      }
    })
    keys.push({ key: '__none__', label: t('issueKanban.unassigned'), color: '#9ca3af' })
    return keys
  }

  return []
})

function getSwimlaneKeyForIssue(issue: any): string {
  if (swimlaneBy.value === 'assignee') {
    if (issue.assignees && issue.assignees.length > 0) return String(issue.assignees[0].id)
    return '__none__'
  }
  if (swimlaneBy.value === 'priority') {
    return issue.priority || 'none'
  }
  if (swimlaneBy.value === 'type') {
    return issue.issue_type_id ? String(issue.issue_type_id) : '__none__'
  }
  return '__none__'
}

const swimlaneGrouped = ref<Record<string, Record<string | number, any[]>>>({})

function rebuildSwimlaneGrouped() {
  if (!swimlaneBy.value) {
    swimlaneGrouped.value = {}
    return
  }

  const result: Record<string, Record<string | number, any[]>> = {}
  swimlaneKeys.value.forEach(s => { result[s.key] = {} })
  // Pre-initialize all column arrays so v-model never receives undefined
  swimlaneKeys.value.forEach(sk => {
    kanbanColumns.value.forEach(c => { result[sk.key][c.key] = [] })
  })

  issues.value.forEach(i => {
    const sk = getSwimlaneKeyForIssue(i)
    if (!result[sk]) result[sk] = {}

    if (groupBy.value === 'state') {
      if (!result[sk][i.state_id]) result[sk][i.state_id] = []
      result[sk][i.state_id].push(i)
    } else if (groupBy.value === 'assignee') {
      if (i.assignees && i.assignees.length > 0) {
        i.assignees.forEach((a: any) => {
          if (!result[sk][a.id]) result[sk][a.id] = []
          result[sk][a.id].push(i)
        })
      } else {
        if (!result[sk]['__unassigned__']) result[sk]['__unassigned__'] = []
        result[sk]['__unassigned__'].push(i)
      }
    } else if (groupBy.value === 'priority') {
      const k = i.priority || 'none'
      if (!result[sk][k]) result[sk][k] = []
      result[sk][k].push(i)
    } else if (groupBy.value === 'labels') {
      if (i.labels && i.labels.length > 0) {
        i.labels.forEach((l: any) => {
          if (!result[sk][l.id]) result[sk][l.id] = []
          result[sk][l.id].push(i)
        })
      } else {
        if (!result[sk]['__nolabel__']) result[sk]['__nolabel__'] = []
        result[sk]['__nolabel__'].push(i)
      }
    }
  })

  swimlaneGrouped.value = result
}

function countSwimlaneIssues(swimlaneKey: string): number {
  if (!swimlaneGrouped.value) return 0
  const cols = swimlaneGrouped.value[swimlaneKey]
  if (!cols) return 0
  let count = 0
  for (const k of Object.keys(cols)) {
    count += (cols[k] || []).length
  }
  return count
}

function priorityDotClass(p: string) {
  const m: Record<string, string> = { urgent: 'bg-red-500', high: 'bg-orange-500', medium: 'bg-yellow-500', low: 'bg-green-500', none: 'bg-gray-400' }
  return m[p] || m.none
}
function getInitials(n: string) { return (n || '?')[0]?.toUpperCase() || '?' }
function assigneeColor(i: number) { return ['#6366f1', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6'][i % 5] }

function getColumnArray(columnKey: string | number, swimlaneKey?: string): any[] | undefined {
  if (swimlaneKey) {
    return swimlaneGrouped.value[swimlaneKey]?.[columnKey]
  }
  return groupedIssues.value[columnKey]
}

function getIssueId(evt: any): number | null {
  const id = evt?.item?.dataset?.issueId
  return id ? Number(id) : null
}

async function onDragUpdate(columnKey: string | number, newIndex: number, oldIndex: number, evt: any, swimlaneKey?: string) {
  console.log('[kanban] onDragUpdate called', { columnKey, newIndex, oldIndex, swimlaneKey, hasItem: !!evt?.item, dataset: evt?.item?.dataset })
  if (dragLocked.value || newIndex === oldIndex) return

  const issueId = getIssueId(evt)
  console.log('[kanban] issueId from DOM dataset:', issueId)
  if (!issueId) return

  const prevCtrl = pendingRequests.get(issueId)
  if (prevCtrl) prevCtrl.abort()
  const ctrl = new AbortController()
  pendingRequests.set(issueId, ctrl)

  const columnIssues = getColumnArray(columnKey, swimlaneKey)
  if (!columnIssues) { console.log('[kanban] columnIssues is empty'); return }
  const sortOrder = computeSortOrder(columnIssues, newIndex, newIndex > oldIndex)
  console.log('[kanban] computed sortOrder:', sortOrder, 'columnIssues count:', columnIssues.length)

  try {
    const r = await issueApi.updateIssue(issueId, { sort_order: sortOrder })
    console.log('[kanban] updateIssue success, returned sort_order:', r.sort_order)
  } catch (err: any) {
    console.error('[kanban] updateIssue failed:', err?.message || err)
    if (err?.name === 'CanceledError' || err?.code === 'ERR_CANCELED') return
    await loadIssues()
    showToast(t('issueKanban.dragRevert'))
    dragLocked.value = true
    setTimeout(() => { dragLocked.value = false }, 500)
  } finally {
    if (pendingRequests.get(issueId) === ctrl) pendingRequests.delete(issueId)
  }
}

async function onDragAdd(columnKey: string | number, newIndex: number, evt: any, swimlaneKey?: string) {
  if (dragLocked.value) return

  const issueId = getIssueId(evt)
  if (!issueId) return

  const prevCtrl = pendingRequests.get(issueId)
  if (prevCtrl) prevCtrl.abort()
  const ctrl = new AbortController()
  pendingRequests.set(issueId, ctrl)

  try {
    if (groupBy.value === 'state') {
      await issueApi.updateIssue(issueId, { state_id: columnKey as number })
    } else if (groupBy.value === 'assignee') {
      const assigneeIds = columnKey === '__unassigned__' ? [] : [columnKey as number]
      await issueApi.updateIssue(issueId, { assignee_ids: assigneeIds })
    } else if (groupBy.value === 'priority') {
      await issueApi.updateIssue(issueId, { priority: columnKey as string as any })
    } else if (groupBy.value === 'labels') {
      if (columnKey !== '__nolabel__') {
        try { await issueApi.addIssueLabel(issueId, columnKey as number) } catch { /* already exists */ }
      }
    }

    const columnIssues = getColumnArray(columnKey, swimlaneKey)
    if (columnIssues) {
      const sortOrder = computeSortOrder(columnIssues, newIndex, true)
      await issueApi.updateIssue(issueId, { sort_order: sortOrder })
    }
  } catch (err: any) {
    if (err?.name === 'CanceledError' || err?.code === 'ERR_CANCELED') return
    await loadIssues()
    showToast(t('issueKanban.dragRevert'))
    dragLocked.value = true
    setTimeout(() => { dragLocked.value = false }, 500)
  } finally {
    if (pendingRequests.get(issueId) === ctrl) pendingRequests.delete(issueId)
  }
}

const quickCreateStateId = ref<number | null>(null)
function openQuickCreate(stateId: number) { quickCreateStateId.value = stateId }
function closeQuickCreate() { quickCreateStateId.value = null }

watch(canCreateIssue, (ok) => {
  if (!ok) closeQuickCreate()
})
function onQuickCreated(issue?: any) {
  quickCreateStateId.value = null
  loadIssues()
  if (issue?.auto_triage_started && issue.id) {
    markAutoTriagePending(issue.id)
    toast.success(t('issue.autoTriageStarted'))
    emit('select', issue)
  }
}

const showImportModal = ref(false)
function onImportSuccess() { loadIssues() }

// ═══ Data loading — filters ONLY via props.rql (unified FilterBar) ═══
async function loadIssues() {
  loading.value = true
  try {
    const params: any = { limit: 200 }
    if (props.rql) params.rql = props.rql
    if (props.filterSortConfig) {
      params.sort_config = props.filterSortConfig
    } else if (props.filterSortBy) {
      params.sort_by = props.filterSortBy
      params.sort_dir = props.filterSortDir || 'desc'
    }
    const result = await issueApi.listIssues(props.projectId, props.workspaceId, params)
    issues.value = result.items
  } catch (e) { console.error('Failed to load issues:', e) }
  finally { loading.value = false }
}

onMounted(async () => {
  await Promise.all([loadIssues(), loadStates(), loadMembers(), loadIssueTypes()])
  rebuildGroupedIssues()
  rebuildSwimlaneGrouped()
})
watch(() => [props.rql, props.filterSortBy, props.filterSortDir, props.filterSortConfig], () => loadIssues())
watch([issues, states], () => {
  rebuildGroupedIssues()
  rebuildSwimlaneGrouped()
})

async function loadStates() {
  try { const r = await api.get(`/projects/${props.projectId}/settings/states`); states.value = r.data?.data ?? r.data ?? [] } catch (e) { /* */ }
}
async function loadMembers() {
  try { const r = await api.get(`/workspaces/${props.workspaceId}/members`); members.value = r.data } catch (e) { /* */ }
}
const issueTypes = ref<any[]>([])
async function loadIssueTypes() {
  try { issueTypes.value = await issueTypeApi.getIssueTypes(props.workspaceId, props.projectId) } catch { /* */ }
}

// ── Batch actions ──
function toggleSelect(id: number) {
  const s = new Set(selectedIds.value); s.has(id) ? s.delete(id) : s.add(id); selectedIds.value = s
}

async function batchChangeState(stateId: number) {
  showBatchState.value = false
  try {
    await issueApi.bulkUpdateIssues(props.projectId, [...selectedIds.value], { state_id: stateId })
    clearSelection()
    showToast(t('issueList.toastStateUpdated'))
    loadIssues()
  } catch (e) { console.error('Batch state failed:', e) }
}

async function batchChangePriority(priority: string) {
  showBatchPriority.value = false
  try {
    await issueApi.bulkUpdateIssues(props.projectId, [...selectedIds.value], { priority: priority as any })
    clearSelection()
    showToast(t('issueList.toastPriorityUpdated'))
    loadIssues()
  } catch (e) { console.error('Batch priority failed:', e) }
}

async function batchAssign(userId: string | number | undefined) {
  showBatchAssign.value = false
  if (!userId) return
  const uid = typeof userId === 'string' ? Number(userId) : userId
  try {
    await issueApi.bulkUpdateIssues(props.projectId, [...selectedIds.value], { assignee_ids: [uid] })
    clearSelection()
    showToast(t('issueList.toastAssigned'))
    loadIssues()
  } catch (e) { console.error('Batch assign failed:', e) }
}

function clearSelection() {
  selectedIds.value = new Set()
  showBatchState.value = false
  showBatchPriority.value = false
  showBatchAssign.value = false
}

async function execBatchDelete() {
  if (!(await confirm(t('issueList.confirmDelete', { 0: String(selectedIds.value.size) })))) return
  try {
    await issueApi.bulkDeleteIssues([...selectedIds.value])
    clearSelection()
    showToast(t('issueList.toastDeleted'))
    loadIssues()
  } catch (e) { console.error('Batch delete failed:', e) }
}

onUnmounted(() => {
  for (const ctrl of pendingRequests.values()) ctrl.abort()
  pendingRequests.clear()
  if (toastTimer) clearTimeout(toastTimer)
})
</script>

<style scoped>
.kanban-ghost {
  opacity: 0.4;
  background-color: #dbeafe;
  border: 2px dashed #60a5fa;
  border-radius: 0.5rem;
}

.kanban-drag {
  opacity: 0.6;
  transform: rotate(2deg);
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
}
</style>
