<template>
  <div class="project-page min-h-screen bg-gray-50 dark:bg-gray-900">
    <!-- 加载状态 -->
    <div v-if="loading" class="flex items-center justify-center h-64">
      <svg class="animate-spin h-8 w-8 text-indigo-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
    </div>

    <!-- 项目内容 -->
    <div v-else-if="project" class="space-y-4">
      <!-- 项目头部 - clean top bar -->
      <div class="px-2">
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center gap-3 min-w-0">
            <div class="w-9 h-9 rounded-lg flex items-center justify-center text-white text-sm font-bold shrink-0" :style="{ backgroundColor: project.color || '#6366f1' }">
              {{ project.name?.charAt(0)?.toUpperCase() || 'P' }}
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-gray-900 dark:text-gray-100 truncate">{{ project.name }}</h1>
              <div class="flex items-center gap-2 mt-0.5">
                <span class="text-xs text-gray-400 dark:text-gray-500 font-mono">{{ project.identifier }}</span>
                <span v-if="latestUpdateStatus" class="text-xs px-2 py-0.5 rounded-full" :class="getUpdateStatusColor(latestUpdateStatus)">{{ getUpdateStatusLabel(latestUpdateStatus) }}</span>
              </div>
            </div>
          </div>
          <div class="flex items-center gap-2 shrink-0">
            <button
              @click="runProjectSummary"
              :disabled="projectSummarizing"
              class="inline-flex items-center gap-1.5 px-3 py-1.5 border border-indigo-200 text-indigo-700 text-xs font-medium rounded-md hover:bg-indigo-50 transition disabled:opacity-50"
              data-test="project-ai-summary"
            >
              {{ projectSummarizing ? t('common.loading') : t('project.aiSummary') }}
            </button>
            <button
              v-if="canCreateIssue"
              @click="router.push(`/workspace/${route.params.slug}/project/${projectId}/issues/new?view=${issueView}`)"
              class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-indigo-600 text-white text-xs font-medium rounded-md hover:bg-indigo-700 transition shadow-sm"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>
              {{ t('project.create') }}
            </button>
          </div>
        </div>

        <!-- C1: Project AI summary panel -->
        <div
          v-if="projectSummary || projectSummaryError"
          class="mb-3 rounded-lg border border-indigo-100 bg-indigo-50/60 dark:bg-indigo-950/30 dark:border-indigo-900 p-4"
          data-test="project-ai-summary-panel"
        >
          <div class="flex items-start justify-between gap-3 mb-2">
            <h3 class="text-sm font-semibold text-indigo-900 dark:text-indigo-100">{{ t('project.aiSummary') }}</h3>
            <div class="flex items-center gap-2 shrink-0">
              <button
                v-if="projectSummary"
                @click="saveProjectSummaryAsPage"
                :disabled="projectSummarySaving"
                class="px-2.5 py-1 text-xs rounded-md bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-50"
              >{{ projectSummarySaving ? t('common.loading') : t('project.aiSummarySavePage') }}</button>
              <button @click="clearProjectSummary" class="text-xs text-gray-500 hover:text-gray-700">✕</button>
            </div>
          </div>
          <p v-if="projectSummaryError" class="text-sm text-red-600">{{ projectSummaryError }}</p>
          <template v-else-if="projectSummary">
            <p class="text-sm text-gray-800 dark:text-gray-200 whitespace-pre-wrap">{{ projectSummary.summary }}</p>
            <ul v-if="projectSummary.insights?.length" class="mt-2 space-y-1">
              <li class="text-xs font-medium text-gray-500">{{ t('project.aiSummaryInsights') }}</li>
              <li v-for="(ins, i) in projectSummary.insights" :key="i" class="text-sm text-gray-700 dark:text-gray-300 pl-2">• {{ ins }}</li>
            </ul>
            <ul v-if="projectSummary.bottlenecks?.length" class="mt-2 space-y-1">
              <li class="text-xs font-medium text-gray-500">{{ t('project.aiSummaryRisks') }}</li>
              <li v-for="b in projectSummary.bottlenecks" :key="b.issue_id" class="text-sm text-gray-700 dark:text-gray-300 pl-2">
                • #{{ b.issue_id }} {{ b.issue_name }} ({{ b.days_in_state }}d / {{ b.state_name }})
              </li>
            </ul>
          </template>
        </div>

        <!-- 标签页导航 - horizontal pills -->
        <nav class="flex items-center gap-1 border-b border-gray-200 dark:border-gray-800 pb-2">
          <button v-for="tab in computedTabs" :key="tab.id" @click="activeTab = tab.id"
            class="px-3.5 py-1.5 text-sm font-medium rounded-md transition-colors"
            :class="activeTab === tab.id ? 'bg-gray-200/70 dark:bg-gray-800 text-gray-900 dark:text-gray-100' : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800/50'">
            {{ tab.name }}
          </button>
          <button @click="showPageConfig = true" class="px-2 py-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded hover:bg-gray-100 dark:hover:bg-gray-800 transition text-sm" :title="t('project.pageConfig')">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>
          </button>
        </nav>
      </div>

      <!-- 标签页内容 -->
      <div v-if="activeTab === 'issues'">
        <QuickCreateInput
          v-if="canCreateIssue"
          :project-id="projectId"
          :workspace-id="workspaceId"
          :issue-types="issueTypes"
          :default-type-id="defaultTypeId"
          :default-state-id="defaultStateId"
          :inline="true"
          @created="onQuickCreated"
        />
        <FilterBar
          :key="'filter-' + projectId"
          :project-id="projectId"
          :workspace-id="workspaceId"
          :current-view="issueView"
          :project-identifier="project?.identifier || ''"
          :current-columns="currentColumns"
          @view-change="handleViewChange"
          @filters-changed="handleFiltersChanged"
          @columns-changed="handleColumnsChanged"
        />

        <IssueList
          v-if="issueView === 'list'"
          :key="'list-' + issueRefreshKey"
          :project-id="projectId"
          :workspace-id="workspaceId"
          :rql="currentRQL"
          :filter-sort-config="sortConfigJson"
          :filter-sort-by="currentSortBy[0]?.key"
          :filter-sort-dir="currentSortBy[0]?.direction"
          :filter-group-by="currentGroupBy?.key"
          :filter-sub-group-by="currentSubGroupBy?.key"
          :search-term="searchTerm"
          :columns="currentColumns"
          @select="openDetailPanel"
          @delete="handleDeleteIssue"
          @columns-changed="handleColumnsChanged"
        />
        <IssueKanban
          v-else-if="issueView === 'kanban'"
          :key="'kanban-' + issueRefreshKey"
          :project-id="projectId"
          :workspace-id="workspaceId"
          :project-identifier="project?.identifier || ''"
          :rql="currentRQL"
          :filter-sort-config="sortConfigJson"
          :filter-sort-by="currentSortBy[0]?.key"
          :filter-sort-dir="currentSortBy[0]?.direction"
          :filter-group-by="currentGroupBy?.key"
          :filter-sub-group-by="currentSubGroupBy?.key"
          @select="openDetailPanel"
        />
        <IssueTreeView
          v-else-if="issueView === 'tree'"
          :key="'tree-' + issueRefreshKey"
          :project-id="projectId"
          :workspace-id="workspaceId"
          :rql="currentRQL"
          :filter-sort-config="sortConfigJson"
          :filter-sort-by="currentSortBy[0]?.key"
          :filter-sort-dir="currentSortBy[0]?.direction"
          :filter-group-by="currentGroupBy?.key"
          :filter-sub-group-by="currentSubGroupBy?.key"
          @select="openDetailPanel"
        />
        <IssueCalendar
          v-else-if="issueView === 'calendar'"
          :project-id="projectId"
          :workspace-id="workspaceId"
          :rql="currentRQL"
          :filter-sort-config="sortConfigJson"
          :filter-sort-by="currentSortBy[0]?.key"
          :filter-sort-dir="currentSortBy[0]?.direction"
          @select="openDetailPanel"
        />
        <IssueGantt
          v-else-if="issueView === 'gantt'"
          :project-id="projectId"
          :workspace-id="workspaceId"
          :rql="currentRQL"
          :filter-sort-config="sortConfigJson"
          :filter-sort-by="currentSortBy[0]?.key"
          :filter-sort-dir="currentSortBy[0]?.direction"
          @select="openDetailPanel"
          @create="router.push(`/workspace/${route.params.slug}/project/${projectId}/issues/new?view=tree`)"
        />
      </div>

      <!-- 侧滑详情面板 -->
      <IssueDetailPanel
        :issue-id="detailIssueId"
        :visible="detailPanelVisible"
        :workspace-id="workspaceId"
        :project-id="projectId"
        @close="closeDetailPanel"
        @delete="handleDetailDelete"
        @refresh="handleDetailRefresh"
      />

      <CycleDetailPanel
        :cycle="selectedCycle"
        :visible="cyclePanelVisible"
        @close="cyclePanelVisible = false"
      />

      <ModuleDetailPanel
        :module="selectedModule"
        :visible="modulePanelVisible"
        :project-id="projectId"
        :workspace-id="workspaceId"
        @close="modulePanelVisible = false"
        @edit="handleModuleEdit"
      />

      <ModuleFormModal
        :visible="moduleFormVisible"
        :edit-module="editingModule"
        :workspace-id="workspaceId"
        :project-id="projectId"
        @close="moduleFormVisible = false; editingModule = null"
        @saved="moduleFormVisible = false; editingModule = null; modulePanelVisible = false"
      />

      <div v-if="activeTab === 'cycles'">
        <CycleList
          :project-id="projectId"
          :workspace-id="workspaceId"
          @select="openCyclePanel"
          @create="goToCycleCreate"
        />
      </div>

      <div v-if="activeTab === 'modules'">
        <ModuleList
          :project-id="projectId"
          :workspace-id="workspaceId"
          @select="openModulePanel"
          @create="moduleFormVisible = true"
          @delete="handleModuleDelete"
        />
      </div>

      <div v-if="activeTab === 'metrics' || activeTab === 'reports'">
        <MetricsView :project-id="projectId" />
      </div>

      <div v-if="activeTab === 'updates'">
        <div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-lg font-semibold">{{ t('project.updates.title') }}</h3>
            <button
              @click="showUpdateForm = true"
              class="px-3 py-1.5 bg-indigo-600 text-white text-sm rounded-md hover:bg-indigo-700"
            >{{ t('project.updates.publish') }}</button>
          </div>

          <div v-if="updatesLoading" class="text-gray-500 text-sm">{{ t('common.loading') }}</div>

          <div v-else-if="projectUpdates.length === 0" class="text-center py-8 text-gray-400 text-sm">
            {{ t('project.updates.empty') }}
          </div>

          <div v-else class="space-y-4">
            <div v-for="update in projectUpdates" :key="update.id" class="border border-gray-100 dark:border-gray-700 rounded-lg p-4">
              <div class="flex items-center justify-between mb-2">
                <div class="flex items-center gap-2">
                  <span class="text-sm font-medium">{{ update.author?.display_name || t('common.unknown') }}</span>
                  <span class="text-xs px-2 py-0.5 rounded-full" :class="getUpdateStatusColor(update.status)">{{ getUpdateStatusLabel(update.status) }}</span>
                </div>
                <span class="text-xs text-gray-400">{{ new Date(update.created_at).toLocaleString(locale === 'zh-CN' ? 'zh-CN' : 'en-US') }}</span>
              </div>
              <p class="text-sm text-gray-700 dark:text-gray-300">{{ update.content }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-else class="text-center py-12">
      <p class="text-gray-500">{{ t('project.notFound') }}</p>
      <button @click="goBack" class="mt-4 text-indigo-600 hover:text-indigo-800 text-sm">{{ t('common.back') }}</button>
    </div>
  </div>

  <!-- AI Copilot Panel -->
  <AICopilot
    :visible="showAIChat"
    :project-id="projectId"
    :workspace-id="workspaceId"
    :project-name="project?.name"
    view="project"
    @close="showAIChat = false"
    @quick-create="handleAIQuickCreate"
    @save-as-page="handleAISaveAsPage"
    @issue-created="triggerRefresh()"
  />
  <CommandPalette
    :visible="showCommandPalette"
    :workspace-slug="slug"
    :project-id="projectId"
    :workspace-id="workspaceId"
    @close="showCommandPalette = false"
    @open-copilot="showAIChat = true; showCommandPalette = false"
  />
  <PageTabConfig v-if="showPageConfig" :project-id="projectId" @close="showPageConfig = false" @saved="tabs => { pageTabs = tabs }" />

  <!-- Update Form Modal -->
  <div v-if="showUpdateForm" class="fixed inset-0 bg-black/30 z-50 flex items-center justify-center">
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl p-6 w-full max-w-md mx-4">
      <h2 class="text-lg font-semibold mb-4">{{ t('project.updates.publish') }}</h2>
      <div class="space-y-3">
        <div>
          <label class="text-sm text-gray-600 mb-1 block">{{ t('common.status') }}</label>
          <select v-model="updateForm.status" class="w-full border rounded-lg px-3 py-2 dark:bg-gray-700 dark:border-gray-600">
            <option value="on_track">{{ t('project.updates.statusOnTrack') }}</option>
            <option value="at_risk">{{ t('project.updates.statusAtRisk') }}</option>
            <option value="off_track">{{ t('project.updates.statusOffTrack') }}</option>
          </select>
        </div>
        <div>
          <label class="text-sm text-gray-600 mb-1 block">{{ t('project.updates.content') }}</label>
          <textarea v-model="updateForm.content" class="w-full border rounded-lg px-3 py-2 dark:bg-gray-700 dark:border-gray-600" rows="4" :placeholder="t('project.updates.contentPlaceholder')"></textarea>
        </div>
      </div>
      <div class="flex justify-end gap-2 mt-5">
        <button @click="showUpdateForm = false" class="px-4 py-2 text-gray-600 hover:bg-gray-100 rounded-lg">{{ t('project.updates.cancel') }}</button>
        <button @click="submitUpdate" class="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700">{{ t('project.updates.submit') }}</button>
      </div>
    </div>
  </div>

  <button
    @click="showAIChat = true"
    class="fixed bottom-16 right-4 w-12 h-12 bg-gradient-to-r from-indigo-500 to-purple-600 text-white rounded-full shadow-lg hover:shadow-xl hover:scale-105 transition flex items-center justify-center text-xl z-30"
    title="AI Assistant (Ctrl+J)"
  >🤖</button>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast'
import { workspaceApi } from '@/api/workspace'
import { projectApi, listPageTabs } from '@/api/project'
import { issueApi } from '@/api/issue'
import { projectUpdateApi, type ProjectUpdate } from '@/api/project-update'
import type { Workspace } from '@/types'
import type { ProjectResponse } from '@/types/project'
import type { ProjectPageTab } from '@/types/project-page-tab'
import { useConfirm } from '@/composables/useConfirm'

import IssueList from '@/components/IssueList.vue'
import IssueKanban from '@/components/IssueKanban.vue'
import IssueTreeView from '@/components/IssueTreeView.vue'
import IssueCalendar from '@/components/IssueCalendar.vue'
import IssueGantt from '@/components/IssueGantt.vue'
import IssueDetailPanel from '@/components/IssueDetailPanel.vue'
import CycleList from '@/components/CycleList.vue'
import CycleDetailPanel from '@/components/CycleDetailPanel.vue'
import ModuleList from '@/components/ModuleList.vue'
import ModuleDetailPanel from '@/components/ModuleDetailPanel.vue'
import ModuleFormModal from '@/components/ModuleFormModal.vue'
import FilterBar from '@/components/FilterBar.vue'
import QuickCreateInput from '@/components/QuickCreateInput.vue'
import AICopilot from '@/components/AICopilot.vue'
import CommandPalette from '@/components/CommandPalette.vue'
import PageTabConfig from '@/components/PageTabConfig.vue'
import MetricsView from '@/views/MetricsView.vue'
import { analyzeWithAI } from '@/api/ai'
import { createPage } from '@/api/page'
import type { CycleResponse } from '@/types/cycle'
import type { ModuleResponse } from '@/types/module'
import { useModuleStore } from '@/stores/module'
import { extractSearchTerm } from '@/utils/highlight'
import { markAutoTriagePending } from '@/utils/autoTriagePref'
import { useProjectPermissions } from '@/composables/usePermission'

const route = useRoute()
const router = useRouter()
const { t, locale } = useI18n()
const toast = useToast()

const workspace = ref<Workspace | null>(null)
const project = ref<ProjectResponse | null>(null)

const projectSummarizing = ref(false)
const projectSummarySaving = ref(false)
const projectSummaryError = ref('')
const projectSummary = ref<{
  summary?: string
  insights?: string[]
  bottlenecks?: Array<{ issue_id: number; issue_name: string; days_in_state: number; state_name: string }>
} | null>(null)

const projectSummaryMarkdown = computed(() => {
  if (!projectSummary.value) return ''
  const insights = (projectSummary.value.insights || []).map((i) => `- ${i}`).join('\n')
  const risks = (projectSummary.value.bottlenecks || []).map(
    (b) => `- #${b.issue_id} ${b.issue_name} (${b.days_in_state}d / ${b.state_name})`,
  ).join('\n')
  return [
    `# ${project.value?.name || 'Project'} — ${t('project.aiSummary')}`,
    '',
    projectSummary.value.summary || '',
    insights ? `\n## ${t('project.aiSummaryInsights')}\n${insights}` : '',
    risks ? `\n## ${t('project.aiSummaryRisks')}\n${risks}` : '',
  ].filter(Boolean).join('\n')
})

async function runProjectSummary() {
  if (!projectId.value) return
  projectSummarizing.value = true
  projectSummaryError.value = ''
  projectSummary.value = null
  try {
    // issue_id=0 → project-scoped analyze (DEF-02 keeps issue-scoped when id > 0)
    projectSummary.value = await analyzeWithAI(projectId.value, 0)
  } catch (e: any) {
    projectSummaryError.value = e?.response?.data?.message || e.message || t('project.aiSummaryFailed')
  } finally {
    projectSummarizing.value = false
  }
}

async function saveProjectSummaryAsPage() {
  if (!projectSummary.value || !projectId.value || !workspaceId.value) return
  projectSummarySaving.value = true
  try {
    await createPage(projectId.value, workspaceId.value, {
      title: `${project.value?.name || 'Project'} — ${t('project.aiSummary')}`,
      content: projectSummaryMarkdown.value,
    })
    toast.success(t('project.aiSummarySaveSuccess'))
  } catch (e: any) {
    toast.error(e?.response?.data?.message || e.message || t('project.aiSummarySaveFailed'))
  } finally {
    projectSummarySaving.value = false
  }
}

function clearProjectSummary() {
  projectSummary.value = null
  projectSummaryError.value = ''
}
function normalizeProjectTab(tab: string | undefined | null): string {
  if (!tab) return 'issues'
  if (tab === 'reports') return 'metrics'
  return tab
}

const loading = ref(false)
const activeTab = ref(normalizeProjectTab(route.query.tab as string))
const issueView = ref<'list' | 'kanban' | 'tree' | 'calendar' | 'gantt'>((route.query.view as any) || 'list')
const issueRefreshKey = ref(0)
function triggerRefresh() { issueRefreshKey.value++ }

const currentRQL = ref('')
const currentSortBy = ref<any[]>([])  // multi-sort array
const currentGroupBy = ref<any>(null)
const currentSubGroupBy = ref<any>(null)
const currentColumns = ref<string[]>([])

const searchTerm = ref('')

// Compute sort_config JSON string for API
const sortConfigJson = computed(() => {
  if (!currentSortBy.value || currentSortBy.value.length === 0) return ''
  const config = currentSortBy.value.map((s: any) => ({
    field: s.key || s.field,
    dir: s.direction || s.dir || 'desc'
  }))
  return JSON.stringify(config)
})

function handleFiltersChanged(rql: string, sortBy: any[] = [], groupBy: any = null, subGroupBy: any = null, quickSearch = '') {
  currentRQL.value = rql
  currentSortBy.value = sortBy
  currentGroupBy.value = groupBy
  currentSubGroupBy.value = subGroupBy
  searchTerm.value = quickSearch || extractSearchTerm(rql)
  triggerRefresh()
}

function handleColumnsChanged(columns: string[]) {
  currentColumns.value = columns
}

function handleViewChange(view: 'list' | 'kanban' | 'tree' | 'calendar' | 'gantt') {
  issueView.value = view
  router.replace({
    query: {
      ...route.query,
      view,
    },
  })
}

// Note: FilterBar handles its own data loading (states, cycles, members, modules, issueTypes, labels, customFields)


const detailIssueId = ref<number | null>(null)
const detailPanelVisible = ref(false)

const selectedCycle = ref<CycleResponse | null>(null)
const cyclePanelVisible = ref(false)

const selectedModule = ref<ModuleResponse | null>(null)
const modulePanelVisible = ref(false)
const moduleFormVisible = ref(false)
const editingModule = ref<ModuleResponse | null>(null)

// AI state
const showAIChat = ref(false)
const showCommandPalette = ref(false)

// AI result action handlers
function handleAIQuickCreate(_preview: Record<string, any>) {
  // AICopilot handles quick create internally by switching to Create tab
  showAIChat.value = true
}

function handleAISaveAsPage(data: { title: string; content: string }) {
  const slug = route.params.slug as string
  const pageUrl = `/workspace/${slug}/project/${projectId.value}/pages`
  // Open page create in new tab with prefilled content via URL hash
  const encoded = encodeURIComponent(`# ${data.title}\n\n${data.content}`)
  window.open(`${pageUrl}?content=${encoded}`, '_blank')
}

// Project updates
const projectUpdates = ref<ProjectUpdate[]>([])
const showUpdateForm = ref(false)
const updateForm = ref({ status: 'on_track', content: '' })
const latestUpdateStatus = ref<string>('')
const updatesLoading = ref(false)

async function loadUpdates() {
  updatesLoading.value = true
  try {
    projectUpdates.value = await projectUpdateApi.list(projectId.value, 10)
    if (projectUpdates.value.length > 0) {
      latestUpdateStatus.value = projectUpdates.value[0].status
    }
  } catch { projectUpdates.value = [] }
  finally { updatesLoading.value = false }
}

async function submitUpdate() {
  if (!updateForm.value.content?.trim()) {
      toast.warning(t('project.updates.contentRequired'))
      return
    }
  try {
    await projectUpdateApi.create(projectId.value, updateForm.value.status, updateForm.value.content)
    updateForm.value = { status: 'on_track', content: '' }
    showUpdateForm.value = false
    await loadUpdates()
  } catch (err) { console.error('Failed to submit update:', err) }
}

function getUpdateStatusLabel(s: string) {
  const map: Record<string, string> = { on_track: t('project.updates.statusOnTrack'), at_risk: t('project.updates.statusAtRisk'), off_track: t('project.updates.statusOffTrack') }
  return map[s] || s
}
function getUpdateStatusColor(s: string) {
  const map: Record<string, string> = { on_track: 'bg-green-100 text-green-700', at_risk: 'bg-yellow-100 text-yellow-700', off_track: 'bg-red-100 text-red-700' }
  return map[s] || 'bg-gray-100'
}

// Reset active tab when switching between projects (Vue Router reuses the component)
watch(() => route.params.id, () => {
  activeTab.value = normalizeProjectTab(route.query.tab as string)
  issueView.value = (route.query.view as any) || 'list'
})

// Watch query.tab changes (e.g. TopBar nav with ?tab=metrics|reports)
watch(() => route.query.tab, (tab) => {
  if (tab && typeof tab === 'string') {
    activeTab.value = tab === 'reports' ? 'metrics' : tab
  }
})

watch(activeTab, (tab) => {
  if (tab === 'settings') {
    router.push(`/workspace/${route.params.slug}/project/${projectId.value}/settings`)
    return
  }
  if (tab === 'pages') {
    router.push(`/workspace/${route.params.slug}/project/${projectId.value}/pages`)
    return
  }
  if (tab === 'dashboards') {
    router.push(`/workspace/${route.params.slug}/project/${projectId.value}/dashboards`)
    return
  }
  detailPanelVisible.value = false
  cyclePanelVisible.value = false
  modulePanelVisible.value = false
})

function openDetailPanel(issue: any) {
  detailIssueId.value = issue.id
  detailPanelVisible.value = true
  router.push({
    query: {
      ...route.query,
      issue: String(issue.id),
    },
  })
}

function onQuickCreated(issue: any) {
  triggerRefresh()
  if (issue?.auto_triage_started && issue.id) {
    markAutoTriagePending(issue.id)
    toast.success(t('issue.autoTriageStarted'))
    openDetailPanel(issue)
  }
}

function closeDetailPanel() {
  detailPanelVisible.value = false
  detailIssueId.value = null
  const q = { ...route.query }
  delete q.issue
  router.replace({ query: q })
}

function handleDetailDelete(issue: any) {
  handleDeleteIssue(issue)
}

function handleDetailRefresh() {
  triggerRefresh()
}

function openCyclePanel(cycle: CycleResponse) {
  selectedCycle.value = cycle
  cyclePanelVisible.value = true
}

function goToCycleCreate() {
  const slug = route.params.slug as string
  router.push(`/workspace/${route.params.slug}/project/${projectId.value}/cycles/new?ws=${slug}`)
}

function openModulePanel(module: ModuleResponse | any) {
  selectedModule.value = module as ModuleResponse
  modulePanelVisible.value = true
}

function handleModuleEdit(module: ModuleResponse) {
  editingModule.value = module
  moduleFormVisible.value = true
}

const moduleStore = useModuleStore()
const { confirm } = useConfirm()

async function handleModuleDelete(module: ModuleResponse | any) {
  if (await confirm(t('project.deleteModuleConfirm', { name: module.name }))) {
    try {
      await moduleStore.deleteModuleAction(module.id)
    } catch (err) {
      console.error('Failed to delete module:', err)
      toast.error(t('project.deleteFailed'))
    }
  }
}

const workspaceId = ref(0)
const projectId = ref(0)
const slug = ref('')

const { can: canInProject } = useProjectPermissions(() => projectId.value)
const canCreateIssue = computed(() => canInProject('issue:create'))

// Deep-links like ?tab=settings|pages|dashboards set activeTab on initial
// load, but the watcher on activeTab only fires on *changes* (not the
// initial value), and there are no template blocks for those tab ids.
// Redirect to the real route once projectId is resolved (ref goes 0 → real id).
watch(projectId, (id) => {
  const tab = activeTab.value
  if (id && (tab === 'settings' || tab === 'pages' || tab === 'dashboards')) {
    router.push(`/workspace/${slug.value}/project/${id}/${tab}`)
  }
}, { once: true })

const pageTabs = ref<ProjectPageTab[]>([])
const issueTypes = ref<any[]>([])
const states = ref<any[]>([])
const showPageConfig = ref(false)

const defaultTypeId = computed(() => {
  const defaultType = issueTypes.value.find((t: any) => t.is_default)
  return defaultType?.id || null
})

const defaultStateId = computed(() => {
  const preferred =
    states.value.find((s: any) => s.is_default) ||
    states.value.find((s: any) => s.group === 'unstarted') ||
    states.value.find((s: any) => s.group === 'backlog')
  return preferred?.id || null
})
const defaultTabs = computed(() => [
  { id: 'issues', name: t('project.tab.issues') },
  { id: 'cycles', name: t('project.tab.cycles') },
  { id: 'modules', name: t('project.tab.modules') },
  { id: 'updates', name: t('project.tab.updates') },
  { id: 'pages', name: t('project.tab.pages') },
  { id: 'metrics', name: t('project.tab.metrics') },
  { id: 'dashboards', name: t('project.tab.dashboards') },
  { id: 'settings', name: t('project.tab.settings') },
])

async function loadPageTabs() {
  try {
    const tabs = await listPageTabs(projectId.value)
    if (tabs.length > 0) pageTabs.value = tabs
  } catch { /* use defaults */ }
}

const computedTabs = computed(() => {
  if (pageTabs.value.length > 0) {
    return pageTabs.value.filter(t => t.visible).map(t => ({ id: t.route_key || `custom_${t.id}`, name: t.name }))
  }
  return defaultTabs.value
})

function goBack() {
  router.push(`/workspace/${route.params.slug}`)
}

async function handleDeleteIssue(issue: any) {
  if (await confirm(t('project.deleteIssueConfirm', { name: issue?.name || 'selected issues' }))) {
    try {
      await issueApi.deleteIssue(issue.id)
      closeDetailPanel()
      triggerRefresh()
    } catch (err) {
      console.error('Failed to delete issue:', err)
      toast.error(t('project.deleteFailed'))
    }
  }
}

function syncDetailFromQuery() {
  const raw = route.query.issue
  const id = raw ? parseInt(String(raw), 10) : NaN
  if (id > 0) {
    detailIssueId.value = id
    detailPanelVisible.value = true
  } else {
    detailIssueId.value = null
    detailPanelVisible.value = false
  }
}

watch(() => route.query.issue, () => {
  syncDetailFromQuery()
})

watch(() => route.query.view, (view) => {
  if (view && typeof view === 'string') {
    issueView.value = view as any
  }
})

onMounted(async () => {
  loading.value = true
  slug.value = route.params.slug as string
  const id = parseInt(route.params.id as string)

  try {
    workspace.value = await workspaceApi.getBySlug(slug.value)
    workspaceId.value = workspace.value.id
    projectId.value = id
    project.value = await projectApi.getProject(id)
    // Filter data is loaded by FilterBar component internally
    await loadUpdates()
    await loadPageTabs()
    try {
      const issueTypeApi = await import('@/api/issue-type')
      issueTypes.value = await issueTypeApi.getIssueTypes(workspaceId.value, projectId.value)
    } catch { issueTypes.value = [] }
    try {
      const stateApi = await import('@/api/project-settings')
      states.value = await stateApi.listStates(projectId.value)
    } catch { states.value = [] }
  } catch (err) {
    console.error('Failed to load project:', err)
  } finally {
    loading.value = false
  }
  syncDetailFromQuery()
  const handleKeydown = (e: KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'j') {
      e.preventDefault()
      showAIChat.value = !showAIChat.value
    }
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
      e.preventDefault()
      showCommandPalette.value = !showCommandPalette.value
    }
    if (e.key === 'Escape' && detailPanelVisible.value) {
      e.preventDefault()
      closeDetailPanel()
    }
  }
  document.addEventListener('keydown', handleKeydown)
  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeydown)
  })
})
</script>

<style scoped>
.project-page {
  @apply p-6;
}
</style>
