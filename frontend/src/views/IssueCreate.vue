<template>
  <div class="issue-create-page">
    <!-- 头部 -->
    <div class="page-header">
      <button @click="goBack" class="back-btn">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>
      <h1 class="page-title">{{ t('issue.createTitle') }}</h1>
    </div>

    <div class="page-content">
      <!-- 左侧表单 -->
      <div class="form-section">
        <!-- 类型选择 -->
        <div class="form-group">
          <label class="form-label">{{ t('issue.type') }}</label>
          <div class="type-selector">
            <button
              v-for="type in issueTypes"
              :key="type.id"
              class="type-option"
              :class="{ active: selectedTypeId === type.id }"
              @click="selectedTypeId = type.id"
            >
              <div class="type-icon" :style="{ backgroundColor: type.color }">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <span>{{ type.name }}</span>
            </button>
          </div>
          <p v-if="issueTypes.length === 0" class="text-sm text-amber-700 mt-2 bg-amber-50 border border-amber-100 rounded-lg px-3 py-2">
            {{ t('issue.noTypesHint') }}
            <router-link
              :to="`/workspace/${slug}/project/${projectId}/settings?section=issue-types`"
              class="text-indigo-600 hover:text-indigo-700 font-medium ml-1"
            >{{ t('issue.importTypesCta') }} →</router-link>
          </p>
        </div>

        <!-- 模板选择 -->
        <div v-if="availableTemplates.length > 0" class="form-group">
          <label class="form-label">{{ t('issue.template') }}</label>
          <select
            v-model.number="selectedTemplateId"
            @change="handleTemplateChange(selectedTemplateId)"
            class="form-input"
          >
            <option :value="null">{{ t('issue.templatePlaceholder') }}</option>
            <option v-for="tmpl in availableTemplates" :key="tmpl.id" :value="tmpl.id">
              {{ tmpl.name }}{{ tmpl.is_default ? t('issue.templateDefault') : '' }}
            </option>
          </select>
        </div>

        <!-- 标题 -->
        <div class="form-group">
          <label class="form-label">{{ t('issue.titleRequired') }}</label>
          <input
            v-model="formData.name"
            type="text"
            class="form-input title-input"
            :placeholder="t('issue.titlePlaceholder')"
            ref="titleInput"
          />
          <div
            v-if="duplicates.length > 0"
            class="mt-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900"
          >
            <p class="font-medium mb-1">{{ t('issue.duplicateWarning') }}</p>
            <ul class="space-y-2">
              <li v-for="dup in duplicates" :key="dup.id" class="flex flex-wrap items-center gap-2">
                <router-link
                  :to="issueLink(dup.id)"
                  class="text-amber-800 underline hover:text-amber-950"
                  target="_blank"
                >
                  #{{ dup.sequence_id }} {{ dup.name }}
                </router-link>
                <button
                  type="button"
                  class="text-xs px-2 py-0.5 rounded border border-amber-300 text-amber-900 hover:bg-amber-100"
                  data-test="merge-into-suggestion"
                  :disabled="saving"
                  @click="startMergeIntoSuggestion(dup)"
                >
                  {{ t('issue.mergeIntoFromSuggestion') }}
                </button>
              </li>
            </ul>
          </div>
        </div>

        <!-- 描述 -->
        <div class="form-group">
          <label class="form-label">{{ t('issue.description') }}</label>
          <RichTextEditor
            v-model="formData.description"
            :placeholder="t('issue.descriptionPlaceholder')"
          />
        </div>

        <!-- 自定义字段 -->
        <div v-if="visibleLinkedFields.length > 0" class="form-group">
          <label class="form-label">{{ t('issue.customFields') }}</label>
          <div class="custom-fields">
            <div v-for="field in visibleLinkedFields" :key="field.field_id" class="field-item">
              <label class="field-label">
                {{ field.name }}
                <span v-if="isFieldRequired(field)" class="required">*</span>
              </label>
              <CustomFieldValueInput
                :field="field as any"
                :value="customFieldValues[field.field_id]"
                :members="projectMembers"
                @update:value="(v: any) => updateCustomFieldValue(field.field_id, v)"
              />
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧属性 -->
      <div class="properties-section">
        <div class="property-group">
          <label class="property-label">{{ t('issue.state') }}</label>
          <select v-model="formData.state_id" class="property-select">
            <option value="">{{ t('issue.statePlaceholder') }}</option>
            <option v-for="state in states" :key="state.id" :value="state.id">
              {{ state.name }}
            </option>
          </select>
        </div>

        <div class="property-group">
          <label class="property-label">{{ t('issue.priority') }}</label>
          <select v-model="formData.priority" class="property-select">
            <option value="">{{ t('issue.priorityPlaceholder') }}</option>
            <option value="urgent">{{ t('issue.priorityUrgent') }}</option>
            <option value="high">{{ t('issue.priorityHigh') }}</option>
            <option value="medium">{{ t('issue.priorityMedium') }}</option>
            <option value="low">{{ t('issue.priorityLow') }}</option>
            <option value="none">{{ t('issue.priorityNone') }}</option>
          </select>
        </div>

        <div class="property-group">
          <label class="property-label">{{ t('issue.assignee') }}</label>
          <UserSelect
            v-model="formData.assignee_id"
            :users="projectMembers"
            :placeholder="t('issue.assigneePlaceholder')"
            :clearable="true"
          />
        </div>

        <div class="property-group">
          <label class="property-label">{{ t('issue.cycle') }}</label>
          <select v-model="formData.cycle_id" class="property-select">
            <option value="">{{ t('issue.cyclePlaceholder') }}</option>
            <option v-for="cycle in cycles" :key="cycle.id" :value="cycle.id">
              {{ cycle.name }}
            </option>
          </select>
        </div>

        <div class="property-group">
          <label class="property-label">{{ t('issue.module') }}</label>
          <select v-model="formData.module_id" class="property-select">
            <option value="">{{ t('issue.modulePlaceholder') }}</option>
            <option v-for="module in modules" :key="module.id" :value="module.id">
              {{ module.name }}
            </option>
          </select>
        </div>

        <div class="property-group">
          <label class="property-label">{{ t('issue.release') }}</label>
          <select v-model="formData.release_id" class="property-select">
            <option value="">{{ t('issue.releasePlaceholder') }}</option>
            <option v-for="release in releases" :key="release.id" :value="release.id">
              {{ release.name }} ({{ release.version }})
            </option>
          </select>
        </div>

        <div class="property-group">
          <label class="property-label">{{ t('issue.startDate') }}</label>
          <input v-model="formData.start_date" type="date" class="property-input" />
        </div>

        <div class="property-group">
          <label class="property-label">{{ t('issue.targetDate') }}</label>
          <input v-model="formData.target_date" type="date" class="property-input" />
        </div>

        <div class="property-group">
          <label class="property-label">{{ t('issue.labels') }}</label>
          <LabelSelector
            :project-id="projectId"
            v-model="selectedLabelIds"
          />
        </div>
      </div>
    </div>

    <!-- 底部操作 -->
    <div class="page-footer">
      <label class="flex items-center gap-2 text-sm text-gray-600 mr-auto cursor-pointer select-none" :title="t('issue.autoTriageOnCreateHint')">
        <input v-model="autoTriage" type="checkbox" class="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500" />
        <span>{{ t('issue.autoTriageOnCreate') }}</span>
      </label>
      <button @click="goBack" class="btn btn-secondary">{{ t('issue.cancel') }}</button>
      <button @click="submitForm" class="btn btn-primary" :disabled="!canSubmit || saving">
        <svg v-if="saving" class="animate-spin -ml-1 mr-2 h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
        </svg>
        {{ saving ? t('issue.creating') : (duplicates.length > 0 ? t('issue.createAnyway') : t('issue.create')) }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, onBeforeUnmount } from 'vue'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast'
import { useRoute, useRouter } from 'vue-router'
import CustomFieldValueInput from '@/components/CustomFieldValueInput.vue'
import RichTextEditor from '@/components/RichTextEditor.vue'
import UserSelect from '@/components/UserSelect.vue'
import LabelSelector from '@/components/LabelSelector.vue'
import type { IssueType } from '@/types/issue-type'
import type { IssueCustomFieldValueUpdate } from '@/types/custom-field'
import type { State } from '@/types/project-settings'
import type { CycleResponse } from '@/types/cycle'
import type { ModuleResponse } from '@/types/module'
import type { User } from '@/types'
import * as issueTypeApi from '@/api/issue-type'
import * as issueApi from '@/api/issue'
import type { DuplicateIssueItem } from '@/api/issue'
import * as stateApi from '@/api/project-settings'
import * as cycleApi from '@/api/cycle'
import * as moduleApi from '@/api/module'
import { releaseApi } from '@/api/release'
import projectApi from '@/api/project'
import { workspaceApi } from '@/api/workspace'
import type { FieldVisibilityEvaluateRequest } from '@/api/conditional-field'
import { useCustomFieldVisibility } from '@/composables/useCustomFieldVisibility'

import * as workItemTemplateApi from '@/api/work-item-template'
import type { WorkItemTemplate } from '@/types/work-item-template'
import { getAutoTriagePref, markAutoTriagePending, setAutoTriagePref } from '@/utils/autoTriagePref'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useToast()

const projectId = computed(() => {
  const id = parseInt(route.params.id as string, 10) || parseInt(route.params.projectId as string, 10)
  return id || 0
})
const workspaceId = ref(0)
const slug = computed(() => route.params.slug as string || '')
// Support both route shapes: /workspace/:slug/project/:id/issues/new (slug-based)
// and /workspaces/:workspaceId/projects/:projectId/issues/new (id-based).
const workspaceIdParam = computed(() => {
  const wid = parseInt(route.params.workspaceId as string, 10)
  return isNaN(wid) ? 0 : wid
})
const returnView = computed(() => route.query.view as string || 'list')

// 定义事件
const emit = defineEmits<{
  (e: 'created', issue: any): void
}>()

// 状态
const issueTypes = ref<IssueType[]>([])
const linkedFields = ref<any[]>([])
const { visibleFieldIds, scheduleEvaluate, evaluateNow } = useCustomFieldVisibility(workspaceId)
const states = ref<State[]>([])
const cycles = ref<CycleResponse[]>([])
const modules = ref<ModuleResponse[]>([])
const releases = ref<{ id: number; name: string; version: string }[]>([])
const projectMembers = ref<User[]>([])
const selectedTypeId = ref<number | null>(null)
const customFieldValues = ref<Record<number, IssueCustomFieldValueUpdate>>({})
const saving = ref(false)
const autoTriage = ref(getAutoTriagePref(projectId.value))
watch(autoTriage, (v) => setAutoTriagePref(projectId.value, v))
watch(projectId, (id) => { autoTriage.value = getAutoTriagePref(id) })

const templates = ref<WorkItemTemplate[]>([])
const selectedTemplateId = ref<number | null>(null)

// 表单数据
const formData = ref({
  name: '',
  description: '',
  state_id: '' as number | string,
  priority: '',
  assignee_id: '' as number | string,
  cycle_id: '' as number | string,
  module_id: '' as number | string,
  release_id: '' as number | string,
  start_date: '',
  target_date: ''
})
const selectedLabelIds = ref<number[]>([])
const duplicates = ref<DuplicateIssueItem[]>([])
const pendingMergeTargetId = ref<number | null>(null)
let duplicateTimer: ReturnType<typeof setTimeout> | null = null

function issueLink(issueId: number) {
  if (slug.value) {
    return `/workspace/${slug.value}/project/${projectId.value}/issues/${issueId}`
  }
  return `/workspaces/${workspaceId.value}/projects/${projectId.value}/issues/${issueId}`
}

function issueDetailPath(issueId: number, query?: Record<string, string>) {
  const base = issueLink(issueId)
  if (!query || Object.keys(query).length === 0) return base
  const params = new URLSearchParams(query)
  return `${base}?${params.toString()}`
}

async function startMergeIntoSuggestion(dup: DuplicateIssueItem) {
  pendingMergeTargetId.value = dup.id
  await submitForm()
}

async function runDuplicateCheck() {
  const name = formData.value.name.trim()
  if (!projectId.value || name.length < 2) {
    duplicates.value = []
    return
  }
  try {
    const result = await issueApi.checkDuplicates(projectId.value, {
      name,
      description: formData.value.description || '',
    })
    duplicates.value = result.duplicates || []
  } catch {
    duplicates.value = []
  }
}

watch(
  () => formData.value.name,
  () => {
    if (duplicateTimer) clearTimeout(duplicateTimer)
    duplicateTimer = setTimeout(runDuplicateCheck, 400)
  }
)

onBeforeUnmount(() => {
  if (duplicateTimer) clearTimeout(duplicateTimer)
})

function buildVisibilityEvaluateBody(): FieldVisibilityEvaluateRequest | null {
  if (!workspaceId.value) return null
  const stateRaw = formData.value.state_id
  const stateNum = typeof stateRaw === 'string' ? parseInt(stateRaw, 10) : Number(stateRaw)
  if (!stateNum || stateNum <= 0) return null
  const priRaw = formData.value.priority
  const priority =
    priRaw !== '' && priRaw !== null && priRaw !== undefined
      ? String(priRaw).trim().toLowerCase()
      : 'none'
  const assigneeRaw = formData.value.assignee_id
  const assigneeNum =
    assigneeRaw === '' || assigneeRaw === null
      ? 0
      : typeof assigneeRaw === 'string'
        ? parseInt(assigneeRaw, 10)
        : assigneeRaw
  const assignee_ids = assigneeNum > 0 ? [assigneeNum] : []
  return {
    issue_type_id: selectedTypeId.value ?? undefined,
    state_id: stateNum,
    priority,
    assignee_ids,
    label_ids: [...selectedLabelIds.value],
  }
}

function isCustomFieldVisible(field: any): boolean {
  const set = visibleFieldIds.value
  if (!set) return true
  return set.has(field.field_id)
}

const visibleLinkedFields = computed(() =>
  linkedFields.value.filter((f) => isCustomFieldVisible(f))
)

// 计算属性
const canSubmit = computed(() => {
  // 标题必填
  if (!formData.value.name.trim()) return false

  // 检查必填的自定义字段
  for (const field of linkedFields.value) {
    if (isFieldRequired(field) && !hasFieldValue(field.field_id)) {
      return false
    }
  }

  return true
})

const availableTemplates = computed(() => {
  if (!selectedTypeId.value) return templates.value
  return templates.value.filter(t => !t.issue_type_id || t.issue_type_id === selectedTypeId.value)
})

// 判断字段是否必填（仅当前上下文可见时计为必填）
function isFieldRequired(field: any): boolean {
  if (!field.is_required) return false
  return isCustomFieldVisible(field)
}

// 判断字段是否有值
function hasFieldValue(fieldId: number): boolean {
  const value = customFieldValues.value[fieldId]
  if (!value) return false
  return !!(value.text_value || value.number_value !== undefined ||
    value.boolean_value !== undefined || value.date_value || value.url_value ||
    (value.json_value && value.json_value.length > 0))
}

async function loadTemplates() {
  try {
    templates.value = await workItemTemplateApi.listWorkItemTemplates(projectId.value)
  } catch (e) {
    console.error('Failed to load templates:', e)
  }
}

function applyTemplate(template: WorkItemTemplate | null) {
  if (!template) return

  const defaults = template.defaults

  if (defaults.name_prefix) {
    formData.value.name = defaults.name_prefix
  }

  if (defaults.priority) {
    formData.value.priority = defaults.priority
  }

  if (defaults.state_id) {
    formData.value.state_id = defaults.state_id
  }

  if (defaults.assignee_ids && defaults.assignee_ids.length > 0) {
    formData.value.assignee_id = defaults.assignee_ids[0]
  }

  if (defaults.label_ids && defaults.label_ids.length > 0) {
    selectedLabelIds.value = [...defaults.label_ids]
  }

  if (defaults.module_id) {
    formData.value.module_id = defaults.module_id
  }

  if (defaults.description_html) {
    formData.value.description = defaults.description_html
  }
}

function applyDefaultTemplateForType(typeId: number | null) {
  // Prefer type-scoped default; fall back to project-wide default (no issue_type_id).
  const defaultTemplate =
    (typeId
      ? templates.value.find((t) => t.is_default && t.issue_type_id === typeId)
      : undefined) ||
    templates.value.find((t) => t.is_default && !t.issue_type_id)

  if (defaultTemplate) {
    selectedTemplateId.value = defaultTemplate.id
    applyTemplate(defaultTemplate)
  } else {
    selectedTemplateId.value = null
  }
}

function handleTemplateChange(templateId: number | null) {
  if (!templateId) {
    selectedTemplateId.value = null
    return
  }
  const template = templates.value.find(t => t.id === templateId)
  if (template) {
    selectedTemplateId.value = templateId
    applyTemplate(template)
  }
}

// 加载数据
async function loadData() {
  try {
    // Resolve workspaceId: prefer route param (id-based route), fall back to slug lookup.
    if (workspaceIdParam.value > 0) {
      workspaceId.value = workspaceIdParam.value
    } else if (slug.value) {
      try {
        const ws = await workspaceApi.getBySlug(slug.value)
        workspaceId.value = ws.id
      } catch (e) {
        console.error('Failed to load workspace:', e)
      }
    }

    // 加载工作项类型 - 如果 API 不可用则使用默认类型
    try {
      const typesRes = await issueTypeApi.getIssueTypes(workspaceId.value, projectId.value)
      issueTypes.value = typesRes

      // 设置默认类型
      const defaultType = typesRes.find((t: any) => t.is_default) || typesRes[0]
      if (defaultType) {
        selectedTypeId.value = defaultType.id
      } else {
        selectedTypeId.value = null
      }
    } catch (e) {
      console.warn('Issue types API not available')
      issueTypes.value = []
      selectedTypeId.value = null
    }

    // 加载状态
    try {
      const statesRes = await stateApi.listStates(projectId.value)
      states.value = statesRes
      if (statesRes.length > 0) {
        const defaultState =
          statesRes.find((s: any) => s.is_default) ||
          statesRes.find((s: any) => s.group === 'unstarted') ||
          statesRes.find((s: any) => s.group === 'backlog') ||
          statesRes[0]
        formData.value.state_id = defaultState.id
      }
    } catch (e) {
      console.error('Failed to load states:', e)
    }

    // 加载周期
    try {
      const cyclesRes = await cycleApi.listCycles(projectId.value)
      cycles.value = cyclesRes.items
    } catch (e) {
      console.error('Failed to load cycles:', e)
    }

    // 加载模块
    try {
      const modulesRes = await moduleApi.listModules(projectId.value, workspaceId.value)
      modules.value = modulesRes
    } catch (e) {
      console.error('Failed to load modules:', e)
    }

    // 加载发布版本
    try {
      const releasesRes = await releaseApi.list(projectId.value)
      releases.value = releasesRes
    } catch (e) {
      console.error('Failed to load releases:', e)
    }

    // 加载项目成员 + 默认负责人
    try {
      const membersRes = await projectApi.listProjectMembers(projectId.value)
      projectMembers.value = membersRes.map((m: any) => m.user)
    } catch (e) {
      console.error('Failed to load project members:', e)
    }

    try {
      const project = await projectApi.getProject(projectId.value)
      if (!formData.value.assignee_id && project?.default_assignee_id) {
        const mid = Number(project.default_assignee_id)
        if (mid > 0 && projectMembers.value.some((m: any) => m.id === mid)) {
          formData.value.assignee_id = mid
        }
      }
    } catch (e) {
      console.error('Failed to load project default assignee:', e)
    }

    // 加载工作项模板
    try {
      await loadTemplates()
    } catch (e) {
      console.error('Failed to load templates:', e)
    }
  } catch (error) {
    console.error('Failed to load data:', error)
  }
}

// 加载类型的关联字段
async function loadLinkedFields(typeId: number) {
  try {
    const fields = await issueTypeApi.getIssueTypeFields(typeId, projectId.value)
    linkedFields.value = fields
    await evaluateNow(buildVisibilityEvaluateBody)
  } catch (error) {
    console.error('Failed to load linked fields:', error)
    linkedFields.value = []
  }
}

// 更新自定义字段值
function updateCustomFieldValue(fieldId: number, value: IssueCustomFieldValueUpdate) {
  customFieldValues.value[fieldId] = value
}

// 提交表单
async function submitForm() {
  // 检查标题
  if (!formData.value.name.trim()) {
    toast.warning(t('issue.titleRequiredMsg'))
    return
  }
  
  // 检查类型
  if (!selectedTypeId.value) {
    toast.warning(t('issue.typeRequiredMsg'))
    return
  }

  await evaluateNow(buildVisibilityEvaluateBody)

  // 检查必填自定义字段
  const missingFields = linkedFields.value
    .filter(field => isFieldRequired(field) && !hasFieldValue(field.field_id))
    .map(field => field.name)
  if (missingFields.length > 0) {
    toast.warning(t('issue.requiredFieldsMsg', { fields: missingFields.join(', ') }))
    return
  }

  saving.value = true
  try {
    const data: any = {
      name: formData.value.name,
      type_id: selectedTypeId.value,
      priority: formData.value.priority || 'none',
      auto_triage: autoTriage.value,
    }
    
    // Add description if provided
    if (formData.value.description && formData.value.description.trim() !== '<p></p>') {
      data.description_html = formData.value.description
    }
    
    // Add state_id if selected
    if (formData.value.state_id) {
      const stateId = typeof formData.value.state_id === 'string' 
        ? parseInt(formData.value.state_id) 
        : formData.value.state_id
      if (stateId > 0) {
        data.state_id = stateId
      }
    }
    
    // Add dates in RFC3339 format
    if (formData.value.start_date) {
      data.start_date = formData.value.start_date + 'T00:00:00Z'
    }
    if (formData.value.target_date) {
      data.target_date = formData.value.target_date + 'T00:00:00Z'
    }
    
    // Add assignee if selected
    if (formData.value.assignee_id) {
      const assigneeId = typeof formData.value.assignee_id === 'string'
        ? parseInt(formData.value.assignee_id)
        : formData.value.assignee_id
      if (assigneeId > 0) {
        data.assignee_ids = [assigneeId]
      }
    }

    // Add labels if selected
    if (selectedLabelIds.value.length > 0) {
      data.label_ids = selectedLabelIds.value
    }

    // Add module if selected
    if (formData.value.module_id) {
      const moduleId = typeof formData.value.module_id === 'string'
        ? parseInt(formData.value.module_id)
        : formData.value.module_id
      if (moduleId > 0) {
        data.module_ids = [moduleId]
      }
    }

    // Add cycle if selected
    if (formData.value.cycle_id) {
      const cycleId = typeof formData.value.cycle_id === 'string'
        ? parseInt(formData.value.cycle_id)
        : formData.value.cycle_id
      if (cycleId > 0) {
        data.cycle_id = cycleId
      }
    }

    // Add release if selected
    if (formData.value.release_id) {
      const releaseId = typeof formData.value.release_id === 'string'
        ? parseInt(formData.value.release_id)
        : formData.value.release_id
      if (releaseId > 0) {
        data.release_id = releaseId
      }
    }

    // Add custom field values
    const cfValues: Record<number, string> = {}
    for (const [fieldId, v] of Object.entries(customFieldValues.value)) {
      const val = v as any
      const field = linkedFields.value.find((f: any) => f.field_id === parseInt(fieldId))
      if (field && !isCustomFieldVisible(field)) {
        continue
      }
      let value = ''
      switch (field?.field_type) {
        case 'text':  value = val.text_value || ''; break
        case 'number': value = val.number_value !== undefined ? String(val.number_value) : ''; break
        case 'boolean': value = val.boolean_value !== undefined ? String(val.boolean_value) : ''; break
        case 'date':  value = val.date_value || ''; break
        case 'url':   value = val.url_value || ''; break
        case 'dropdown':
        case 'member': value = val.json_value?.length ? JSON.stringify(val.json_value) : ''; break
        default: value = val.text_value || ''
      }
      if (value !== '') {
        cfValues[parseInt(fieldId)] = value
      }
    }
    if (Object.keys(cfValues).length > 0) {
      data.custom_field_values = cfValues
    }

    const issue = await issueApi.createIssue(projectId.value, workspaceId.value, data)

    if (issue?.auto_triage_started && issue.id) {
      markAutoTriagePending(issue.id)
      toast.success(t('issue.autoTriageStarted'))
    } else {
      toast.success(t('issue.createSuccess'))
    }

    emit('created', issue)

    if (pendingMergeTargetId.value && issue?.id) {
      const mergeQuery = { mergeTargetId: String(pendingMergeTargetId.value) }
      pendingMergeTargetId.value = null
      router.push(issueDetailPath(issue.id, mergeQuery))
      return
    }

    const query: Record<string, string> = { view: returnView.value }
    if (issue?.id) query.issue = String(issue.id)
    if (slug.value) {
      router.push({ path: `/workspace/${slug.value}/project/${projectId.value}`, query })
    } else {
      router.push({ path: `/workspaces/${workspaceId.value}/projects/${projectId.value}`, query })
    }
  } catch (error: any) {
    console.error('Failed to create issue:', error)
    const errorMsg = error.response?.data?.message || error.message || t('issue.unknownError')
    toast.error(t('issue.createFailed', { msg: errorMsg }))
  } finally {
    saving.value = false
  }
}

// 返回
function goBack() {
  router.back()
}

// 监听类型变化
watch(selectedTypeId, (newTypeId) => {
  if (newTypeId) {
    loadLinkedFields(newTypeId)
  }
  applyDefaultTemplateForType(newTypeId)
})

watch(
  [
    () => workspaceId.value,
    () => selectedTypeId.value,
    () => formData.value.state_id,
    () => formData.value.priority,
    () => formData.value.assignee_id,
    () => selectedLabelIds.value,
  ],
  () => {
    scheduleEvaluate(buildVisibilityEvaluateBody)
  },
  { deep: true }
)

function applyCreateQueryPrefill() {
  const q = route.query
  const rawName = q.name
  if (typeof rawName === 'string' && rawName.trim()) {
    formData.value.name = rawName.trim()
  }
  const rawType = q.type_id
  if (rawType != null && String(rawType) !== '') {
    const tid = parseInt(String(rawType), 10)
    if (!Number.isNaN(tid) && tid > 0 && issueTypes.value.some((t) => t.id === tid)) {
      selectedTypeId.value = tid
    }
  }
  const rawState = q.state_id
  if (rawState != null && String(rawState) !== '') {
    const sid = parseInt(String(rawState), 10)
    if (!Number.isNaN(sid) && sid > 0 && states.value.some((s) => s.id === sid)) {
      formData.value.state_id = sid
    }
  }
}

onMounted(async () => {
  await loadData()
  applyCreateQueryPrefill()
  applyDefaultTemplateForType(selectedTypeId.value)
})

</script>

<style scoped>
.issue-create-page {
  @apply min-h-screen bg-white flex flex-col;
}

.page-header {
  @apply flex items-center space-x-3 px-6 py-3 bg-white border-b border-gray-100;
}

.back-btn {
  @apply p-2 rounded-lg hover:bg-gray-100 transition;
}

.page-title {
  @apply text-base font-semibold text-gray-800;
}

.page-content {
  @apply flex flex-1;
}

.form-section {
  @apply flex-1 bg-white border-r border-gray-100 p-6;
}

.properties-section {
  @apply w-64 bg-white p-6 h-fit;
}

.form-group {
  @apply mb-5;
}

.form-label {
  @apply block text-xs font-medium text-gray-400 uppercase tracking-wide mb-2;
}

.required {
  @apply text-red-500;
}

.title-input {
  @apply text-lg font-medium;
}

.form-input,
.form-textarea,
.property-select,
.property-input {
  @apply w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-1 focus:ring-blue-400 focus:border-blue-400;
}

.form-textarea {
  @apply resize-none;
}

.type-selector {
  @apply flex flex-wrap gap-2;
}

.type-option {
  @apply flex items-center space-x-2 px-3 py-2 rounded-lg border border-gray-100 hover:border-gray-200 transition;
}

.type-option.active {
  @apply border-gray-300 bg-gray-50;
}

.type-icon {
  @apply w-6 h-6 rounded flex items-center justify-center text-white;
}

.property-group {
  @apply mb-4;
}

.property-label {
  @apply block text-xs font-medium text-gray-400 uppercase tracking-wide mb-1.5;
}

.custom-fields {
  @apply space-y-3;
}

.field-item {
  @apply border border-gray-100 rounded-lg p-3;
}

.field-label {
  @apply block text-sm font-medium text-gray-600 mb-2;
}

.page-footer {
  @apply flex items-center justify-end space-x-3 px-6 py-3 bg-white border-t border-gray-100;
}

.btn {
  @apply px-4 py-2 rounded-lg font-medium transition;
}

.btn-primary {
  @apply bg-neutral-900 text-white hover:bg-neutral-800 disabled:opacity-50 disabled:cursor-not-allowed;
}

.btn-secondary {
  @apply bg-gray-100 text-gray-600 hover:bg-gray-200;
}
</style>
