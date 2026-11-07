<template>
  <div v-if="visible" class="merge-into-overlay" data-test="merge-into-dialog" @click.self="close">
    <div class="merge-into-dialog bg-white rounded-xl shadow-2xl border border-gray-200 w-[540px] max-h-[560px] flex flex-col">
      <div class="flex items-center justify-between px-5 py-4 border-b border-gray-100 shrink-0">
        <h3 class="text-sm font-semibold text-gray-800">
          {{ phase === 'confirm' ? t('issue.mergeIntoConfirmTitle') : t('issue.mergeIntoTitle') }}
        </h3>
        <button type="button" class="text-gray-400 hover:text-gray-600" @click="close">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
        </button>
      </div>

      <div v-if="phase === 'pick'" class="flex flex-col min-h-0 flex-1">
        <div class="px-5 py-3 border-b border-gray-100 shrink-0">
          <p class="text-xs text-gray-500 mb-2">{{ t('issue.mergeIntoPickTarget') }}</p>
          <input
            ref="searchInput"
            v-model="query"
            type="text"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            :placeholder="t('issue.searchIssuesPlaceholder')"
            data-test="merge-into-search"
            @input="onSearch"
          />
        </div>

        <div class="flex-1 overflow-y-auto min-h-0">
          <div v-if="loading" class="flex justify-center py-12 text-sm text-gray-400">
            {{ t('common.loading') }}
          </div>
          <div v-else-if="query.trim() && results.length === 0" class="py-12 text-center text-sm text-gray-400">
            {{ t('common.noResults') }}
          </div>
          <div v-else-if="!query.trim()" class="py-12 text-center text-sm text-gray-400">
            {{ t('issue.searchIssuesPlaceholder') }}
          </div>
          <div v-else>
            <button
              v-for="item in results"
              :key="item.id"
              type="button"
              class="flex w-full items-center gap-3 px-5 py-3 border-b border-gray-50 last:border-b-0 hover:bg-gray-50 text-left transition-colors"
              data-test="merge-into-result"
              @click="pickTarget(item)"
            >
              <span class="text-[10px] text-gray-400 font-mono shrink-0">#{{ item.sequence_id }}</span>
              <span class="text-sm font-medium text-gray-800 truncate">{{ item.name }}</span>
            </button>
          </div>
        </div>
      </div>

      <div v-else class="px-5 py-4 flex flex-col gap-4" data-test="merge-into-confirm">
        <p class="text-sm text-gray-700">
          {{ t('issue.mergeIntoConfirm', {
            source: sourceLabel,
            target: targetLabel,
          }) }}
        </p>
        <div class="flex items-center gap-2 text-sm text-gray-600 bg-gray-50 rounded-lg px-3 py-2">
          <span class="flex-1 truncate">{{ targetLabel }}</span>
          <button type="button" class="text-xs text-indigo-600 hover:text-indigo-800 underline shrink-0" @click="backToPick">
            {{ t('issue.mergeIntoChangeTarget') }}
          </button>
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <button type="button" class="px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 rounded-lg" @click="close">
            {{ t('issue.cancel') }}
          </button>
          <button
            type="button"
            class="px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg disabled:opacity-50"
            data-test="merge-into-confirm-btn"
            :disabled="!target || merging || !canConfirm"
            @click="confirmMerge"
          >
            {{ merging ? t('common.loading') : t('issue.mergeInto') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast'
import { suggestIssues, mergeIssues, getIssue } from '@/api/issue'

const { t } = useI18n()
const toast = useToast()

const props = defineProps<{
  visible: boolean
  projectId: number
  sourceIssueId: number
  sourceLabel: string
  projectIdentifier?: string
  excludeId?: number
  initialTargetId?: number | null
}>()

const emit = defineEmits<{
  close: []
  merged: [targetIssueId: number]
}>()

type Phase = 'pick' | 'confirm'
const phase = ref<Phase>('pick')
const query = ref('')
const results = ref<Array<{ id: number; sequence_id: number; name: string }>>([])
const loading = ref(false)
const merging = ref(false)
const target = ref<{ id: number; sequence_id: number; name: string } | null>(null)
const searchInput = ref<HTMLInputElement | null>(null)

let debounceTimer: ReturnType<typeof setTimeout> | null = null

const targetLabel = computed(() => {
  if (!target.value) return ''
  const prefix = props.projectIdentifier ? `${props.projectIdentifier}-` : '#'
  return `${prefix}${target.value.sequence_id} ${target.value.name}`
})

const canConfirm = computed(() => {
  if (!target.value) return false
  if (target.value.id === props.sourceIssueId) return false
  return true
})

function onSearch() {
  if (debounceTimer) clearTimeout(debounceTimer)
  const q = query.value.trim()
  if (!q) {
    results.value = []
    loading.value = false
    return
  }
  loading.value = true
  debounceTimer = setTimeout(async () => {
    try {
      const data = await suggestIssues(props.projectId, q, 20)
      const ex = props.excludeId ?? props.sourceIssueId
      results.value = (data || []).filter((item: { id: number }) => item.id !== ex)
    } catch {
      results.value = []
    } finally {
      loading.value = false
    }
  }, 300)
}

function pickTarget(item: { id: number; sequence_id: number; name: string }) {
  if (item.id === props.sourceIssueId) return
  target.value = item
  phase.value = 'confirm'
}

function backToPick() {
  phase.value = 'pick'
}

function close() {
  emit('close')
}

async function loadInitialTarget(id: number) {
  try {
    const issue = await getIssue(id)
    if (issue.id === props.sourceIssueId || issue.project_id !== props.projectId) return
    target.value = { id: issue.id, sequence_id: issue.sequence_id, name: issue.name }
    phase.value = 'confirm'
  } catch {
    /* invalid prefill — stay on pick */
  }
}

async function confirmMerge() {
  if (!target.value || !canConfirm.value || merging.value) return
  merging.value = true
  try {
    await mergeIssues(target.value.id, [props.sourceIssueId])
    emit('merged', target.value.id)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { message?: string } } }
    toast.error(err?.response?.data?.message || t('issue.mergeIntoFailed'))
  } finally {
    merging.value = false
  }
}

watch(
  () => props.visible,
  async (v) => {
    if (v) {
      phase.value = 'pick'
      query.value = ''
      results.value = []
      target.value = null
      merging.value = false
      if (props.initialTargetId) {
        await loadInitialTarget(props.initialTargetId)
      }
      await nextTick()
      if (phase.value === 'pick') searchInput.value?.focus()
    }
  },
  { immediate: true }
)

defineExpose({ phase, canConfirm, target, merging })
</script>

<style scoped>
.merge-into-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 110;
}
.merge-into-dialog {
  max-height: 80vh;
}
</style>
