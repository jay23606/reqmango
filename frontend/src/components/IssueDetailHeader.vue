<template>
  <div class="flex items-center justify-between px-6 py-3 bg-white border-b border-gray-200">
    <div class="flex items-center gap-3 min-w-0 flex-1">
      <button
        data-test="back-btn"
        class="p-1 -ml-1 text-gray-400 hover:text-gray-600 transition-colors shrink-0"
        @click="$emit('back')"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="15 18 9 12 15 6" />
        </svg>
      </button>

      <span
        class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium shrink-0"
        :style="{ backgroundColor: issue?.issue_type?.color + '20', color: issue?.issue_type?.color }"
      >
        {{ issue?.issue_type?.name }}
      </span>

      <span class="text-xs text-gray-400 font-mono shrink-0">{{ projectIdentifier }}-{{ issue?.sequence_id }}</span>

      <input
        data-test="issue-title-input"
        class="flex-1 min-w-0 text-sm font-semibold text-gray-800 bg-transparent border border-transparent rounded px-1.5 py-0.5 hover:border-gray-200 focus:border-indigo-300 focus:bg-white focus:outline-none focus:ring-2 focus:ring-indigo-100"
        :value="localTitle"
        :disabled="savingTitle"
        @input="onTitleInput"
        @blur="commitTitle"
        @keydown.enter.prevent="($event.target as HTMLInputElement).blur()"
      />

      <span
        v-if="issue?.priority && issue.priority !== 'none'"
        class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-medium shrink-0"
        :class="priorityClass(issue.priority)"
      >
        <span class="w-1.5 h-1.5 rounded-full" :class="priorityDot(issue.priority)"></span>
        {{ priorityLabel(issue.priority) }}
      </span>
    </div>

    <div class="flex items-center gap-2 shrink-0">
      <button
        class="p-1.5 rounded transition-colors"
        :class="isWatching ? 'text-indigo-500 hover:bg-indigo-50' : 'text-gray-400 hover:text-gray-600 hover:bg-gray-50'"
        :title="isWatching ? t('issue.unwatch') : t('issue.watch')"
        @click="$emit('toggleWatch')"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/></svg>
      </button>

      <button
        class="p-1.5 text-gray-400 hover:text-indigo-500 hover:bg-indigo-50 rounded transition-colors"
        :title="t('issue.copyLink')"
        @click="copyLink"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"/></svg>
      </button>
      <span v-if="copied" class="text-[10px] text-green-600">{{ t('issue.copied') }}</span>

      <button
        v-if="canMerge"
        type="button"
        data-test="merge-into-btn"
        class="px-3 py-1.5 text-xs font-medium text-gray-700 border border-gray-200 rounded-md hover:bg-gray-50 transition-colors"
        @click="$emit('merge')"
      >
        {{ t('issue.mergeInto') }}
      </button>

      <button
        v-if="canDelete"
        data-test="delete-btn"
        class="px-3 py-1.5 text-xs font-medium text-red-500 border border-red-200 rounded-md hover:bg-red-50 transition-colors"
        @click="$emit('delete')"
      >
        {{ t('common.delete') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast'
import { useProjectPermissions } from '@/composables/usePermission'

const props = defineProps<{
  issue: any
  projectIdentifier?: string
  isWatching?: boolean
}>()

const emit = defineEmits<{
  back: []
  delete: []
  merge: []
  toggleWatch: []
  'update:title': [title: string]
}>()

const { t } = useI18n()
const toast = useToast()
const { can } = useProjectPermissions(() => props.issue?.project_id)
const canDelete = computed(() => can('issue:delete'))
const canMerge = computed(() => can('issue:edit'))
const copied = ref(false)
const localTitle = ref(props.issue?.name || '')
const savingTitle = ref(false)
let titleDirty = false

watch(() => props.issue?.name, (name) => {
  if (!titleDirty) localTitle.value = name || ''
}, { immediate: true })

function onTitleInput(e: Event) {
  titleDirty = true
  localTitle.value = (e.target as HTMLInputElement).value
}

function commitTitle() {
  const next = localTitle.value.trim()
  const prev = (props.issue?.name || '').trim()
  titleDirty = false
  if (!next || next === prev) {
    localTitle.value = props.issue?.name || ''
    return
  }
  savingTitle.value = true
  emit('update:title', next)
  savingTitle.value = false
}

function copyLink() {
  const url = window.location.href
  navigator.clipboard.writeText(url).then(() => {
    copied.value = true
    setTimeout(() => copied.value = false, 2000)
  }).catch(() => {
    toast.error(t('issue.copyFailed'))
  })
}

function priorityClass(p: string) {
  const m: Record<string, string> = {
    urgent: 'bg-red-50 text-red-700', high: 'bg-orange-50 text-orange-700',
    medium: 'bg-yellow-50 text-yellow-700', low: 'bg-green-50 text-green-700',
    none: 'bg-gray-50 text-gray-600'
  }
  return m[p] || m.none
}
function priorityDot(p: string) {
  const m: Record<string, string> = {
    urgent: 'bg-red-500', high: 'bg-orange-500', medium: 'bg-yellow-500',
    low: 'bg-green-500', none: 'bg-gray-400'
  }
  return m[p] || m.none
}
function priorityLabel(p: string) {
  const m: Record<string, string> = {
    urgent: t('issue.priorityUrgent'), high: t('issue.priorityHigh'),
    medium: t('issue.priorityMedium'), low: t('issue.priorityLow'), none: t('issue.priorityNone')
  }
  return m[p] || p
}
</script>
