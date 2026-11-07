import api from './index'
import type { TimeTrack, TimeTrackSummary } from '@/types/time-track'

export async function startTimer(issueId: number, description?: string): Promise<TimeTrack> {
  const r = await api.post(`/issues/${issueId}/time-tracks/start`, { description })
  return r.data
}
export async function stopTimer(issueId: number): Promise<TimeTrack> {
  const r = await api.post(`/issues/${issueId}/time-tracks/stop`)
  return r.data
}
export async function listTimeTracks(issueId: number): Promise<TimeTrack[]> {
  const r = await api.get(`/issues/${issueId}/time-tracks`)
  return r.data
}
export async function getTimeSummary(issueId: number): Promise<TimeTrackSummary> {
  const r = await api.get(`/issues/${issueId}/time-tracks/summary`)
  return r.data
}
export async function deleteTimeTrack(issueId: number, id: number): Promise<void> {
  await api.delete(`/issues/${issueId}/time-tracks/${id}`)
}

/** duration is seconds */
export async function createTimeTrack(
  issueId: number,
  data: { duration: number; description?: string; ended_at?: string }
): Promise<TimeTrack> {
  const r = await api.post(`/issues/${issueId}/time-tracks`, data)
  return r.data
}
