<template>
  <div class="border-t border-gray-100 pt-4 mt-4">
    <div class="flex items-center justify-between mb-3">
      <h4 class="text-sm font-semibold text-gray-700">{{ t('timeTrack.title') }}</h4>
      <div class="flex items-center gap-2">
        <span v-if="summary" class="text-xs text-gray-500">
          {{ summary.total_hours.toFixed(1) }}h / {{ summary.entry_count }} {{ t('timeTrack.entries') }}
        </span>
        <button
          v-if="!isRunning"
          data-testid="time-track-start"
          @click="start"
          :disabled="loading"
          class="px-3 py-1 bg-green-600 text-white text-xs rounded hover:bg-green-700 disabled:opacity-50"
        >
          ▶ {{ t('timeTrack.start') }}
        </button>
        <button
          v-else
          data-testid="time-track-stop"
          @click="stop"
          :disabled="loading"
          class="px-3 py-1 bg-red-500 text-white text-xs rounded hover:bg-red-600 disabled:opacity-50"
        >
          ⏹ {{ t('timeTrack.stop') }}
        </button>
      </div>
    </div>

    <div v-if="isRunning" class="mb-3 p-2 bg-green-50 border border-green-200 rounded text-xs text-green-700 flex items-center gap-2">
      <span class="animate-pulse">🔴</span> {{ t('timeTrack.running') }}
    </div>

    <form class="mb-3 flex flex-wrap items-end gap-2" @submit.prevent="submitManual">
      <div>
        <label class="block text-[10px] text-gray-400 mb-0.5">{{ t('timeTrack.manualMinutes') }}</label>
        <input
          v-model.number="manualMinutes"
          data-testid="time-track-manual-minutes"
          type="number"
          min="1"
          class="w-20 px-2 py-1 border border-gray-200 rounded text-xs"
        />
      </div>
      <div class="flex-1 min-w-[8rem]">
        <label class="block text-[10px] text-gray-400 mb-0.5">{{ t('timeTrack.description') }}</label>
        <input
          v-model="manualDesc"
          data-testid="time-track-manual-desc"
          type="text"
          class="w-full px-2 py-1 border border-gray-200 rounded text-xs"
        />
      </div>
      <button
        type="submit"
        data-testid="time-track-manual-submit"
        :disabled="loading || !manualMinutes || manualMinutes < 1"
        class="px-3 py-1 bg-gray-800 text-white text-xs rounded hover:bg-gray-900 disabled:opacity-50"
      >
        {{ t('timeTrack.logTime') }}
      </button>
    </form>

    <div v-if="entries.length" class="space-y-1 max-h-40 overflow-y-auto">
      <div v-for="e in entries" :key="e.id" class="flex items-center justify-between text-xs py-1.5 px-2 rounded hover:bg-gray-50">
        <div class="flex items-center gap-2">
          <span class="text-gray-400">{{ formatDate(e.started_at) }}</span>
          <span class="text-gray-600">{{ e.description || t('timeTrack.noDescription') }}</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="text-gray-500 font-mono">{{ formatDuration(e.duration) }}</span>
          <button @click="remove(e.id)" class="text-gray-400 hover:text-red-500">✕</button>
        </div>
      </div>
    </div>
    <div v-else class="text-xs text-gray-400 py-2">{{ t('timeTrack.noEntries') }}</div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as ttApi from '@/api/time-track'
import type { TimeTrack, TimeTrackSummary } from '@/types/time-track'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()
const props = defineProps<{ issueId: number }>()

const entries = ref<TimeTrack[]>([])
const summary = ref<TimeTrackSummary | null>(null)
const isRunning = ref(false)
const loading = ref(false)
const manualMinutes = ref<number | null>(null)
const manualDesc = ref('')

onMounted(() => load())

async function load() {
  try {
    const [e, s] = await Promise.all([ttApi.listTimeTracks(props.issueId), ttApi.getTimeSummary(props.issueId)])
    entries.value = e
    summary.value = s
    isRunning.value = s.is_running
  } catch (_) { /* ignore */ }
}

async function start() {
  loading.value = true
  try { await ttApi.startTimer(props.issueId); await load() } catch (_) {}
  finally { loading.value = false }
}

async function stop() {
  loading.value = true
  try { await ttApi.stopTimer(props.issueId); await load() } catch (_) {}
  finally { loading.value = false }
}

async function remove(id: number) {
  try { await ttApi.deleteTimeTrack(props.issueId, id); await load() } catch (_) {}
}

async function submitManual() {
  if (!manualMinutes.value || manualMinutes.value < 1) return
  loading.value = true
  try {
    await ttApi.createTimeTrack(props.issueId, {
      duration: Math.round(manualMinutes.value * 60),
      description: manualDesc.value || undefined,
    })
    manualMinutes.value = null
    manualDesc.value = ''
    await load()
  } catch (_) {}
  finally { loading.value = false }
}

function formatDuration(sec: number): string {
  if (!sec) return '0m'
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  return h > 0 ? `${h}h ${m}m` : `${m}m`
}

function formatDate(d: string): string {
  return new Date(d).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}
</script>
