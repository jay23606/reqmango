<template>
  <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
    <div class="flex items-center justify-between px-3 py-2 border-b bg-gray-50/50">
      <span class="text-xs font-semibold text-gray-700">{{ t('issue.relatedPages') }}</span>
      <button
        v-if="canEditIssue"
        type="button"
        data-test="linked-pages-add-toggle"
        class="text-xs font-medium text-indigo-600 hover:text-indigo-700"
        @click="toggleAddRow"
      >
        {{ t('issue.addPage') }}
      </button>
    </div>

    <div v-if="canEditIssue && showAddRow" class="px-3 py-2 border-b border-gray-100 space-y-2">
      <input
        v-model="searchQuery"
        type="search"
        data-test="linked-pages-search"
        class="w-full text-xs px-2 py-1.5 border border-gray-200 rounded focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
        :placeholder="t('issue.searchPagesPlaceholder')"
        @input="scheduleSearch"
      />
      <div v-if="searching" class="text-[10px] text-gray-400">{{ t('common.loading') }}</div>
      <div v-else-if="searchQuery.trim().length > 0 && searchQuery.trim().length < 2" class="text-[10px] text-gray-400">
        {{ t('issue.linkedPagesSearchMinChars') }}
      </div>
      <div v-else-if="addCandidates.length === 0 && searchQuery.trim().length >= 2" class="text-[10px] text-gray-400">
        {{ t('issue.linkedPagesNoSearchResults') }}
      </div>
      <ul v-else class="max-h-32 overflow-y-auto divide-y divide-gray-50">
        <li v-for="p in addCandidates" :key="p.id">
          <button
            type="button"
            class="w-full text-left px-1 py-1.5 text-xs text-gray-700 hover:bg-indigo-50 rounded"
            @click="handleAdd(p)"
          >
            {{ p.title }}
          </button>
        </li>
      </ul>
    </div>

    <div v-if="loading" class="px-3 py-6 text-center text-xs text-gray-400">{{ t('common.loading') }}</div>
    <div v-else-if="pages.length === 0" class="px-4 py-8 text-center text-sm text-gray-400" data-test="linked-pages-empty">
      {{ t('issue.noPages') }}
    </div>
    <ul v-else class="divide-y divide-gray-50" data-test="linked-pages-list">
      <li
        v-for="p in pages"
        :key="p.id"
        class="flex items-center justify-between gap-2 px-3 py-2 hover:bg-gray-50/50"
      >
        <button
          type="button"
          class="text-left text-sm text-indigo-600 hover:text-indigo-800 hover:underline truncate min-w-0"
          @click="openInPages(p.id)"
        >
          {{ p.title }}
        </button>
        <button
          v-if="canEditIssue"
          type="button"
          data-test="linked-pages-unlink"
          class="shrink-0 text-[10px] text-gray-400 hover:text-red-600"
          @click="handleRemove(p)"
        >
          {{ t('issue.unlinkPage') }}
        </button>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast'
import { useProjectPermissions } from '@/composables/usePermission'
import { listIssuePages, addIssuePage, removeIssuePage } from '@/api/issue'
import { searchPages } from '@/api/page'
import type { Page } from '@/types/page'

const props = defineProps<{
  issueId: number
  projectId: number
  slug: string
}>()

const { t } = useI18n()
const toast = useToast()
const router = useRouter()
const { can } = useProjectPermissions(() => props.projectId)

const canEditIssue = computed(() => can('issue:edit'))

const pages = ref<Page[]>([])
const loading = ref(true)
const showAddRow = ref(false)
const searchQuery = ref('')
const searchResults = ref<Page[]>([])
const searching = ref(false)
let searchTimer: ReturnType<typeof setTimeout> | null = null

const linkedIds = computed(() => new Set(pages.value.map((p) => p.id)))

const addCandidates = computed(() =>
  searchResults.value.filter((p) => !linkedIds.value.has(p.id)),
)

function toggleAddRow() {
  showAddRow.value = !showAddRow.value
  if (!showAddRow.value) {
    searchQuery.value = ''
    searchResults.value = []
  }
}

async function loadLinkedPages() {
  loading.value = true
  try {
    const list = await listIssuePages(props.issueId)
    pages.value = Array.isArray(list) ? list : []
  } catch (e) {
    console.error('Failed to load linked pages:', e)
    pages.value = []
    toast.error(t('issue.linkedPagesLoadFailed'))
  } finally {
    loading.value = false
  }
}

function scheduleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  const q = searchQuery.value.trim()
  if (q.length < 2) {
    searchResults.value = []
    searching.value = false
    return
  }
  searching.value = true
  searchTimer = setTimeout(() => void runSearch(q), 300)
}

async function runSearch(q: string) {
  try {
    const hits = await searchPages(props.projectId, q)
    searchResults.value = Array.isArray(hits) ? hits : []
  } catch (e) {
    console.error('Page search failed:', e)
    searchResults.value = []
  } finally {
    searching.value = false
  }
}

async function handleAdd(p: Page) {
  try {
    await addIssuePage(props.issueId, p.id)
    toast.success(t('issue.linkedPagesAdded'))
    searchQuery.value = ''
    searchResults.value = []
    await loadLinkedPages()
  } catch (e: any) {
    toast.error(e?.response?.data?.message || e?.message || t('issue.saveFailed'))
  }
}

async function handleRemove(p: Page) {
  try {
    await removeIssuePage(props.issueId, p.id)
    toast.success(t('issue.linkedPagesRemoved'))
    await loadLinkedPages()
  } catch (e: any) {
    toast.error(e?.response?.data?.message || e?.message || t('issue.saveFailed'))
  }
}

function openInPages(pageId: number) {
  router.push({
    path: `/workspace/${props.slug}/project/${props.projectId}/pages`,
    query: { page: String(pageId) },
  })
}

onMounted(() => {
  void loadLinkedPages()
})

defineExpose({ loadLinkedPages })
</script>
