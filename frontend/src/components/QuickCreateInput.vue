<template>
  <div class="quick-create-input" :class="inline ? '' : 'px-4 py-2 bg-gray-50 border-b border-gray-200'">
    <div class="flex items-center space-x-2">
      <div class="flex-1 flex items-center space-x-2">
        <select
          v-model="quickCreate.type_id"
          class="px-2 py-1.5 border border-gray-300 rounded-md text-sm bg-white"
        >
          <option v-if="issueTypes.length === 0" value="">{{ t('issueList.loading') }}</option>
          <option value="">{{ t('issue.allTypes') }}</option>
          <option v-for="t in issueTypes" :key="t.id" :value="t.id">{{ t.name }}</option>
        </select>
        <input
          ref="titleInput"
          v-model="quickCreate.title"
          type="text"
          :placeholder="effectivePlaceholder"
          class="flex-1 px-3 py-1.5 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          @keydown.enter="handleCreate"
          @keydown.esc="handleCancel"
        />
        <select
          v-if="showPriority"
          v-model="quickCreate.priority"
          class="px-2 py-1.5 border border-gray-300 rounded-md text-sm bg-white"
        >
          <option value="none">{{ t('issue.priorityNone') }}</option>
          <option value="urgent">{{ t('issue.priorityUrgent') }}</option>
          <option value="high">{{ t('issue.priorityHigh') }}</option>
          <option value="medium">{{ t('issue.priorityMedium') }}</option>
          <option value="low">{{ t('issue.priorityLow') }}</option>
        </select>
      </div>
      <div class="flex items-center space-x-1">
        <button
          @click="handleCreate"
          :disabled="creating"
          class="px-3 py-1.5 bg-indigo-600 text-white text-sm rounded-md hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ creating ? t('issueList.creating') : (duplicates.length > 0 ? t('issue.createAnyway') : t('common.create')) }}
        </button>
        <button
          v-if="showCancel"
          @click="handleCancel"
          class="px-3 py-1.5 text-sm text-gray-600 border border-gray-300 rounded-md hover:bg-gray-50"
        >
          {{ t('common.cancel') }}
        </button>
      </div>
    </div>
    <label
      class="mt-1.5 inline-flex items-center gap-1.5 text-[11px] text-gray-500 cursor-pointer select-none"
      :title="t('issue.autoTriageOnCreateHint')"
    >
      <input v-model="autoTriage" type="checkbox" class="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500" />
      <span>{{ t('issue.autoTriageOnCreate') }}</span>
    </label>
    <p v-if="activeTemplateName" class="mt-1 text-[11px] text-gray-400">
      {{ t('workItemTemplate.appliedHint', { name: activeTemplateName }) }}
    </p>
    <div
      v-if="duplicates.length > 0"
      class="mt-1.5 text-xs text-amber-800 bg-amber-50 border border-amber-200 rounded px-2 py-1.5"
    >
      <span class="font-medium">{{ t('issue.duplicateWarning') }}</span>
      <span class="ml-1">
        <template v-for="(dup, i) in duplicates" :key="dup.id">
          <router-link
            :to="`/workspaces/${workspaceId}/projects/${projectId}/issues/${dup.id}`"
            class="underline hover:text-amber-950"
            target="_blank"
          >#{{ dup.sequence_id }} {{ dup.name }}</router-link><span v-if="i < duplicates.length - 1"> · </span>
        </template>
      </span>
    </div>
    <p v-if="titleError" class="mt-1.5 text-xs text-red-600" data-test="quick-create-title-error">{{ titleError }}</p>
    <div
      v-if="apiErrorMessage"
      class="mt-1.5 text-xs text-red-700 bg-red-50 border border-red-100 rounded px-2 py-1.5 space-y-1"
      data-test="quick-create-api-error"
    >
      <p>{{ apiErrorMessage }}</p>
      <router-link
        v-if="fullFormHref"
        :to="fullFormHref"
        class="text-indigo-600 hover:text-indigo-800 font-medium underline inline-block"
        data-test="quick-create-full-form-link"
      >{{ t('quickCreate.fullFormLink') }}</router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import issueApi from '@/api/issue'
import type { DuplicateIssueItem } from '@/api/issue'
import { listWorkItemTemplates } from '@/api/work-item-template'
import type { WorkItemTemplate } from '@/types/work-item-template'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast'
import { getAutoTriagePref, setAutoTriagePref } from '@/utils/autoTriagePref'

const route = useRoute()
const { t } = useI18n()
const toast = useToast()

const props = withDefaults(defineProps<{
  projectId: number
  workspaceId: number
  issueTypes: any[]
  defaultTypeId?: number
  defaultStateId?: number
  placeholder?: string
  showPriority?: boolean
  showCancel?: boolean
  autoFocus?: boolean
  inline?: boolean
}>(), {
  placeholder: '',
  showPriority: true,
  showCancel: false,
  autoFocus: false,
  inline: false
})

const effectivePlaceholder = computed(() => props.placeholder || t('quickCreate.placeholder'))

const emit = defineEmits<{
  (e: 'created', issue: any): void
  (e: 'cancel'): void
}>()

const titleInput = ref<HTMLInputElement | null>(null)
const creating = ref(false)
const titleError = ref('')
const apiErrorMessage = ref('')
const autoTriage = ref(getAutoTriagePref(props.projectId))
watch(autoTriage, (v) => setAutoTriagePref(props.projectId, v))
watch(() => props.projectId, (id) => { autoTriage.value = getAutoTriagePref(id) })
const duplicates = ref<DuplicateIssueItem[]>([])
const templates = ref<WorkItemTemplate[]>([])
const activeTemplate = ref<WorkItemTemplate | null>(null)
let duplicateTimer: ReturnType<typeof setTimeout> | null = null

const quickCreate = reactive({
  title: '',
  type_id: '' as string | number | null,
  priority: 'none' as string,
  state_id: props.defaultStateId || null as number | null,
  description_html: '' as string,
})

const activeTemplateName = computed(() => activeTemplate.value?.name || '')

/** Prefer explicit default, else project default flag, else first type — used when sending type_id if the user leaves “all types”. */
const resolvedDefaultTypeId = computed((): number | null => {
  if (props.defaultTypeId != null && props.defaultTypeId > 0) return props.defaultTypeId
  const marked = props.issueTypes?.find((x: any) => x.is_default)
  if (marked?.id) return marked.id
  const first = props.issueTypes?.[0]
  return first?.id ?? null
})

function resolveTypeIdForRequest(): number | undefined {
  if (quickCreate.type_id !== '' && quickCreate.type_id != null) {
    const n = Number(quickCreate.type_id)
    return Number.isFinite(n) && n > 0 ? n : undefined
  }
  const fallback = resolvedDefaultTypeId.value
  return fallback != null && fallback > 0 ? fallback : undefined
}

const fullFormHref = computed(() => {
  const slug = route.params.slug as string | undefined
  if (!slug || !props.projectId) return null
  const q: Record<string, string> = { view: (route.query.view as string) || 'list' }
  const name = quickCreate.title.trim()
  if (name) q.name = name
  const tid = resolveTypeIdForRequest()
  if (tid != null) q.type_id = String(tid)
  if (quickCreate.state_id) q.state_id = String(quickCreate.state_id)
  return { path: `/workspace/${slug}/project/${props.projectId}/issues/new`, query: q }
})

function extractApiErrorMessage(e: any): string {
  const data = e?.response?.data
  if (typeof data?.message === 'string' && data.message.trim()) return data.message.trim()
  if (Array.isArray(data?.errors) && data.errors.length) {
    const first = data.errors[0]
    if (typeof first === 'string') return first
    if (first?.message) return String(first.message)
  }
  if (typeof e?.message === 'string' && e.message.trim()) return e.message.trim()
  return t('quickCreate.createFailedGeneric')
}

watch(() => props.defaultTypeId, (newVal) => {
  if (newVal && !quickCreate.type_id) {
    quickCreate.type_id = newVal
  }
}, { immediate: true })

watch(() => quickCreate.type_id, () => {
  applyDefaultTemplate()
})

async function loadTemplates() {
  try {
    templates.value = await listWorkItemTemplates(props.projectId)
    applyDefaultTemplate()
  } catch {
    templates.value = []
  }
}

function resolveDefaultTemplate(typeId: number | null): WorkItemTemplate | null {
  const typed = typeId
    ? templates.value.find((t) => t.is_default && t.issue_type_id === typeId)
    : undefined
  if (typed) return typed
  return templates.value.find((t) => t.is_default && !t.issue_type_id) || null
}

function applyDefaultTemplate() {
  const typeId = quickCreate.type_id ? Number(quickCreate.type_id) : null
  const tpl = resolveDefaultTemplate(typeId)
  activeTemplate.value = tpl
  if (!tpl) return
  const d = tpl.defaults || {}
  if (d.priority) quickCreate.priority = d.priority
  if (d.state_id) quickCreate.state_id = Number(d.state_id)
  else if (props.defaultStateId) quickCreate.state_id = props.defaultStateId
  if (d.description_html) quickCreate.description_html = d.description_html
  // Prefill name prefix only when title empty
  if (d.name_prefix && !quickCreate.title.trim()) {
    quickCreate.title = d.name_prefix
  }
}

onMounted(() => {
  loadTemplates()
  if (props.autoFocus) titleInput.value?.focus()
})

async function runDuplicateCheck() {
  const name = quickCreate.title.trim()
  if (!props.projectId || name.length < 2) {
    duplicates.value = []
    return
  }
  try {
    const result = await issueApi.checkDuplicates(props.projectId, { name })
    duplicates.value = result.duplicates || []
  } catch {
    duplicates.value = []
  }
}

watch(() => quickCreate.title, () => {
  titleError.value = ''
  apiErrorMessage.value = ''
  if (duplicateTimer) clearTimeout(duplicateTimer)
  duplicateTimer = setTimeout(runDuplicateCheck, 400)
})

onBeforeUnmount(() => {
  if (duplicateTimer) clearTimeout(duplicateTimer)
})

async function handleCreate() {
  if (creating.value) return
  const rawTitle = quickCreate.title.trim()
  if (!rawTitle) {
    titleError.value = t('quickCreate.titleRequired')
    toast.warning(t('quickCreate.titleRequired'))
    return
  }

  creating.value = true
  apiErrorMessage.value = ''
  titleError.value = ''
  try {
    let name = rawTitle
    const prefix = activeTemplate.value?.defaults?.name_prefix
    if (prefix && !name.startsWith(prefix)) {
      name = prefix + name
    }
    const typeId = resolveTypeIdForRequest()
    const issueData: any = {
      name,
      priority: quickCreate.priority,
      auto_triage: autoTriage.value,
    }
    if (typeId != null) {
      issueData.type_id = typeId
    }
    if (quickCreate.state_id) {
      issueData.state_id = quickCreate.state_id
    }
    if (quickCreate.description_html) {
      issueData.description_html = quickCreate.description_html
    }
    const issue = await issueApi.createIssue(props.projectId, props.workspaceId, issueData)
    emit('created', issue)
    quickCreate.title = ''
    quickCreate.priority = 'none'
    quickCreate.description_html = ''
    duplicates.value = []
    applyDefaultTemplate()
    titleInput.value?.focus()
  } catch (e: any) {
    console.error('Failed to quick create issue:', e)
    const status = e?.response?.status
    const msg = extractApiErrorMessage(e)
    apiErrorMessage.value = msg
    toast.error(msg)
    if (status >= 400 && status < 500) {
      /* keep title for correction */
    }
  } finally {
    creating.value = false
  }
}

function handleCancel() {
  quickCreate.title = ''
  quickCreate.priority = 'none'
  quickCreate.description_html = ''
  duplicates.value = []
  titleError.value = ''
  apiErrorMessage.value = ''
  emit('cancel')
}
</script>
