import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import TimeTrackPanel from '@/components/TimeTrackPanel.vue'
import * as ttApi from '@/api/time-track'

vi.mock('@/composables/useI18n', () => ({
  useI18n: () => ({ t: (k: string) => k }),
}))

vi.mock('@/api/time-track', () => ({
  listTimeTracks: vi.fn(),
  getTimeSummary: vi.fn(),
  startTimer: vi.fn(),
  stopTimer: vi.fn(),
  deleteTimeTrack: vi.fn(),
  createTimeTrack: vi.fn(),
}))

async function mountPanel(isRunning = false) {
  vi.mocked(ttApi.listTimeTracks).mockResolvedValue([])
  vi.mocked(ttApi.getTimeSummary).mockResolvedValue({
    total_hours: 0,
    entry_count: 0,
    is_running: isRunning,
  } as any)
  const wrapper = mount(TimeTrackPanel, { props: { issueId: 1 } })
  await flushPromises()
  return wrapper
}

describe('TimeTrackPanel keyboard shortcut', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('Space on the focused panel root starts the timer', async () => {
    const wrapper = await mountPanel(false)
    await wrapper.trigger('keydown', { key: ' ' })
    await flushPromises()
    expect(ttApi.startTimer).toHaveBeenCalledWith(1)
    expect(ttApi.stopTimer).not.toHaveBeenCalled()
  })

  it('Enter on the focused panel root stops a running timer', async () => {
    const wrapper = await mountPanel(true)
    await wrapper.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect(ttApi.stopTimer).toHaveBeenCalledWith(1)
    expect(ttApi.startTimer).not.toHaveBeenCalled()
  })

  it('ignores auto-repeat keydown from a held key', async () => {
    const wrapper = await mountPanel(false)
    await wrapper.trigger('keydown', { key: ' ', repeat: true })
    await flushPromises()
    expect(ttApi.startTimer).not.toHaveBeenCalled()
    expect(ttApi.stopTimer).not.toHaveBeenCalled()
  })

  it('does not capture Space typed in the manual description input', async () => {
    const wrapper = await mountPanel(false)
    await wrapper.get('[data-testid="time-track-manual-desc"]').trigger('keydown', { key: ' ' })
    await flushPromises()
    expect(ttApi.startTimer).not.toHaveBeenCalled()
  })
})
