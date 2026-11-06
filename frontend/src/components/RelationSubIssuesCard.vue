<template>
  <div class="border border-green-200 rounded-lg overflow-hidden bg-green-50/30">
    <!-- Card header -->
    <div
      v-if="hierarchyError"
      class="px-3 py-2 text-xs text-red-800 bg-red-50 border-b border-red-100 flex items-start justify-between gap-2"
      data-test="subissue-hierarchy-error"
    >
      <span class="leading-snug">{{ hierarchyError }}</span>
      <button
        type="button"
        class="shrink-0 text-red-500 hover:text-red-700 px-1"
        :aria-label="t('common.close')"
        @click="$emit('dismiss-hierarchy-error')"
      >×</button>
    </div>

    <div class="flex items-center justify-between px-3 py-2 bg-green-50 border-b border-green-100">
      <div class="flex items-center gap-2">
        <span class="text-xs font-semibold text-green-700">{{ t('subIssue.title') }}</span>
        <span
          v-if="subIssues.length > 0"
          class="px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-green-100 text-green-700"
          data-test="subissue-count-badge"
        >{{ completedCount }}/{{ subIssues.length }}</span>
        <span
          v-else
          class="px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-green-100/80 text-green-600"
          data-test="subissue-count-badge"
        >0</span>
      </div>
      <!-- Add dropdown -->
      <div v-if="canAddSubIssues" class="relative">
        <button
          data-test="add-subissue"
          class="text-[10px] font-medium text-green-600 hover:text-green-800 hover:underline"
          @click="toggleAddMenu"
        >+ {{ t('subIssue.add') }}</button>
        <div
          v-if="showMenu"
          class="absolute right-0 top-full mt-1 w-44 bg-white border border-gray-200 rounded-lg shadow-lg z-10 py-1"
        >
          <button
            v-if="canLinkExisting"
            data-test="add-existing-subissue"
            class="w-full text-left px-3 py-2 text-xs text-gray-700 hover:bg-gray-50 flex items-center gap-2"
            @click="selectExisting"
          >
            <span class="text-blue-500">&#9744;</span>
            {{ t('subIssue.selectExisting') }}
          </button>
          <button
            v-if="canQuickCreate"
            data-test="quick-create-subissue"
            class="w-full text-left px-3 py-2 text-xs text-gray-700 hover:bg-gray-50 flex items-center gap-2"
            @click="startQuickCreate"
          >
            <span class="text-green-500">+</span>
            {{ t('subIssue.quickCreate') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Quick-create inline bar -->
    <div v-if="quickCreating" class="px-3 py-2 border-b border-green-100 bg-white">
      <div class="flex items-center gap-2">
        <!-- Type selector -->
        <div class="relative shrink-0">
          <button
            class="flex items-center gap-1 px-2 py-1.5 border rounded text-[10px] font-medium hover:bg-gray-50 transition-colors"
            :class="selectedTypeId ? 'border-green-300' : 'border-gray-300 text-gray-400'"
            :style="selectedType ? { backgroundColor: selectedType.color + '15', borderColor: selectedType.color + '40', color: selectedType.color } : {}"
            @click.stop="showTypeDropdown = !showTypeDropdown"
            @blur="cancelTypeDropdown"
          >
            <span
              v-if="selectedType"
              class="w-1.5 h-1.5 rounded-full shrink-0"
              :style="{ backgroundColor: selectedType.color }"
            ></span>
            {{ selectedType ? selectedType.name : t('subIssue.noType') }}
            <svg class="w-2.5 h-2.5 opacity-50" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd"/></svg>
          </button>
          <div
            v-if="showTypeDropdown"
            class="absolute left-0 top-full mt-1 w-36 bg-white border border-gray-200 rounded-lg shadow-lg z-20 py-1 max-h-48 overflow-y-auto"
          >
            <button
              class="w-full text-left px-2 py-1.5 text-[10px] text-gray-400 hover:bg-gray-50"
              @click="selectType(0)"
            >{{ t('subIssue.noType') }}</button>
            <button
              v-for="it in allowedIssueTypes"
              :key="it.id"
              class="w-full text-left px-2 py-1.5 text-[10px] hover:bg-gray-50 flex items-center gap-1.5"
              @click="selectType(it.id)"
            >
              <span class="w-1.5 h-1.5 rounded-full shrink-0" :style="{ backgroundColor: it.color }"></span>
              <span class="text-gray-700">{{ it.name }}</span>
            </button>
          </div>
        </div>
        <!-- Name input -->
        <input
          ref="quickInput"
          v-model="quickName"
          type="text"
          class="flex-1 px-2 py-1.5 border border-green-300 rounded text-xs focus:outline-none focus:border-green-500 min-w-0"
          :placeholder="t('subIssue.quickCreatePlaceholder')"
          @keydown.enter="submitQuickCreate"
          @keydown.escape="cancelQuickCreate"
        />
      </div>
    </div>

    <!-- Empty state -->
    <div v-if="subIssues.length === 0 && !quickCreating" class="px-4 py-6 text-center text-xs text-gray-500">
      <p class="text-gray-500">{{ t('subIssue.noSubIssues') }}</p>
      <p class="mt-1 text-[10px] text-gray-400">{{ t('subIssue.emptyHint') }}</p>
      <div v-if="canAddSubIssues" class="mt-4 flex flex-wrap justify-center gap-2">
        <button
          v-if="canQuickCreate"
          type="button"
          data-test="empty-quick-create-subissue"
          class="px-3 py-1.5 text-[10px] font-medium rounded-md bg-green-600 text-white hover:bg-green-700 transition-colors"
          @click="startQuickCreateFromEmpty"
        >
          {{ t('subIssue.quickCreate') }}
        </button>
        <button
          v-if="canLinkExisting"
          type="button"
          data-test="empty-add-existing-subissue"
          class="px-3 py-1.5 text-[10px] font-medium rounded-md border border-green-200 text-green-700 bg-white hover:bg-green-50 transition-colors"
          @click="selectExistingFromEmpty"
        >
          {{ t('subIssue.selectExisting') }}
        </button>
      </div>
    </div>

    <!-- Table of sub-issues -->
    <div v-if="subIssues.length > 0" class="overflow-x-auto">
      <table class="w-full text-[10px]">
        <thead>
          <tr class="border-b border-green-100 text-gray-500">
            <th class="px-3 py-1.5 text-left w-8"></th>
            <th class="px-2 py-1.5 text-left">{{ t('issue.type') }}</th>
            <th class="px-2 py-1.5 text-left">ID</th>
            <th class="px-2 py-1.5 text-left">{{ t('issue.title') }}</th>
            <th class="px-2 py-1.5 text-left">{{ t('issue.status') }}</th>
            <th class="px-2 py-1.5 text-left">{{ t('issue.priority') }}</th>
            <th class="px-2 py-1.5 text-left">{{ t('issue.assignee') }}</th>
            <th class="px-2 py-1.5 text-right">{{ t('issue.targetDate') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="issue in subIssues"
            :key="issue.id"
            class="border-b border-green-50 last:border-b-0 hover:bg-green-50/50 transition-colors"
          >
            <!-- Checkbox -->
            <td class="px-3 py-2">
              <input
                type="checkbox"
                :checked="issue.state_group === 'done'"
                @change="$emit('toggle', issue.id)"
              />
            </td>
            <!-- Type badge -->
            <td class="px-2 py-2">
              <span
                v-if="issue.issue_type"
                class="px-1.5 py-0.5 rounded text-[10px] font-medium whitespace-nowrap"
                :style="{ backgroundColor: (issue.issue_type.color || '#e5e7eb') + '20', color: issue.issue_type.color || '#6b7280' }"
              >{{ issue.issue_type.name }}</span>
              <span v-else class="text-gray-300">—</span>
            </td>
            <!-- ID -->
            <td class="px-2 py-2 font-mono text-gray-400">#{{ issue.sequence_id }}</td>
            <!-- Title (clickable) -->
            <td class="px-2 py-2">
              <span
                class="subissue-clickable text-xs font-medium text-gray-800 cursor-pointer hover:text-indigo-600"
                @click="$emit('navigate', issue.id)"
              >{{ issue.name }}</span>
            </td>
            <!-- State -->
            <td class="px-2 py-2">
              <span class="flex items-center gap-1">
                <span
                  class="w-1.5 h-1.5 rounded-full"
                  :style="{ backgroundColor: stateColor(issue.state_group) }"
                ></span>
                {{ issue.state_name }}
              </span>
            </td>
            <!-- Priority -->
            <td class="px-2 py-2">
              <span class="flex items-center gap-1">
                <span
                  class="w-1.5 h-1.5 rounded-full"
                  :style="{ backgroundColor: priorityColor(issue.priority) }"
                ></span>
                {{ t(`issue.priority${(issue.priority || 'none').charAt(0).toUpperCase() + (issue.priority || 'none').slice(1)}`) }}
              </span>
            </td>
            <!-- Assignee -->
            <td class="px-2 py-2 text-gray-500">
              {{ issue.assignees?.[0]?.display_name || issue.assignees?.[0]?.username || '—' }}
            </td>
            <!-- Due date -->
            <td class="px-2 py-2 text-gray-400 text-right">
              {{ issue.target_date ? formatDate(issue.target_date) : '—' }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'
import { useI18n } from '@/composables/useI18n'
import { stateColor, priorityColor, formatDate } from '@/composables/useRelationHelpers'

const { t } = useI18n()

interface IssueType {
  id: number
  name: string
  color: string
  level?: number
}

interface Assignee {
  id: number
  display_name?: string
  username?: string
}

interface SubIssue {
  id: number
  sequence_id: number
  name: string
  state_name: string
  state_group: string
  priority: string
  assignees?: Assignee[]
  target_date?: string | null
  issue_type?: IssueType
}

const props = withDefaults(
  defineProps<{
    subIssues: SubIssue[]
    issueTypes?: IssueType[]
    parentIssueType?: { id: number; level: number } | null
    /** POST /issues with parent_id — quick create. */
    canQuickCreate?: boolean
    /** PUT other issue parent_id — add existing. */
    canLinkExisting?: boolean
    /** Server message from hierarchy / parent update failure. */
    hierarchyError?: string | null
  }>(),
  {
    canQuickCreate: true,
    canLinkExisting: true,
    hierarchyError: null,
  },
)

const canAddSubIssues = computed(() => props.canQuickCreate || props.canLinkExisting)

// Filter types to only those compatible with the parent issue's type level.
// Child type level must be >= parent type level (cannot be shallower).
const allowedIssueTypes = computed(() => {
  if (!props.issueTypes) return []
  const parentLevel = props.parentIssueType?.level ?? -1
  if (parentLevel < 0) return props.issueTypes
  return props.issueTypes.filter(t => (t.level ?? 0) >= parentLevel)
})

const emit = defineEmits<{
  add: []
  toggle: [issueId: number]
  navigate: [issueId: number]
  'add-existing': []
  'quick-create': [name: string, typeId?: number]
  'dismiss-hierarchy-error': []
}>()

const completedCount = computed(() => props.subIssues.filter((s) => s.state_group === 'done').length)

const showMenu = ref(false)
const quickCreating = ref(false)
const quickName = ref('')
const quickInput = ref<HTMLInputElement | null>(null)
const showTypeDropdown = ref(false)
const selectedTypeId = ref<number>(0)

const selectedType = computed(() => {
  if (!selectedTypeId.value) return null
  return allowedIssueTypes.value.find(t => t.id === selectedTypeId.value) || null
})

function dismissHierarchyAnd(fn: () => void) {
  if (props.hierarchyError) emit('dismiss-hierarchy-error')
  fn()
}

function toggleAddMenu() {
  dismissHierarchyAnd(() => { showMenu.value = !showMenu.value })
}

function selectExisting() {
  dismissHierarchyAnd(() => {
    showMenu.value = false
    emit('add-existing')
  })
}

function selectExistingFromEmpty() {
  dismissHierarchyAnd(() => {
    showMenu.value = false
    emit('add-existing')
  })
}

async function startQuickCreate() {
  dismissHierarchyAnd(() => {
    showMenu.value = false
  })
  quickCreating.value = true
  quickName.value = ''
  showTypeDropdown.value = false
  await nextTick()
  quickInput.value?.focus()
}

async function startQuickCreateFromEmpty() {
  showMenu.value = false
  await startQuickCreate()
}

function selectType(typeId: number) {
  selectedTypeId.value = typeId
  showTypeDropdown.value = false
  quickInput.value?.focus()
}

function cancelTypeDropdown() {
  setTimeout(() => { showTypeDropdown.value = false }, 150)
}

function submitQuickCreate() {
  const name = quickName.value.trim()
  if (!name) return
  quickCreating.value = false
  quickName.value = ''
  if (selectedTypeId.value) {
    emit('quick-create', name, selectedTypeId.value)
  } else {
    emit('quick-create', name)
  }
}

function cancelQuickCreate() {
  quickCreating.value = false
  quickName.value = ''
}
</script>
