import assert from 'node:assert/strict'
import { isValidElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { toast } from 'sonner'
import { test, vi } from 'vitest'

import type { ScanTask } from '@/api/tasks'
import { taskProgressState, type TaskStage } from '@/features/tasks/task-progress-state'
import { scanStatus } from '@/features/tasks/scan-status'
import { notifyOfflineTask, notifyScanTask } from '@/features/tasks/task-toast'
import { offlineSubmission } from './fixtures'

vi.mock('sonner', () => ({ toast: { info: vi.fn() } }))

test('scan and download workflows advance through their own stages', () => {
  assert.equal(taskProgressState('scanning').value, 0)
  assert.equal(taskProgressState('reconciling').value, 50)
  assert.equal(taskProgressState('reconciling', false, 50).value, 75)
  assert.equal(taskProgressState('downloading', true, 50).value, 100 / 6)
  assert.equal(taskProgressState('scanning', true).value, 100 / 3)
  assert.equal(taskProgressState('done').value, 100)
  assert.equal(taskProgressState('done', true).value, 100)
})

test('waiting phases describe what the workflow is waiting for', () => {
  assert.deepEqual(taskProgressState('queued'), { label: '等待扫描', value: 0 })
  assert.deepEqual(taskProgressState('locating', true), {
    label: '已下载，等待 115 返回文件信息',
    value: 100 / 3
  })
})

test('an unknown or unavailable phase stays indeterminate instead of crashing', () => {
  // Simulate an unknown stage received from a newer backend.
  for (const phase of ['future-stage', 'downloading']) {
    assert.deepEqual(taskProgressState(phase as TaskStage), { label: '等待进度同步', value: null })
  }
})

test('invalid numeric progress cannot escape its current stage or produce NaN', () => {
  for (const value of [-1, NaN, Infinity]) {
    assert.equal(taskProgressState('reconciling', false, value).value, 50)
  }
  assert.equal(taskProgressState('downloading', true, 120).value, 100 / 3)
})

test('scan notifications report indexing progress without metadata stages', () => {
  const scan: ScanTask = {
    id: 1,
    type: 'scan',
    status: 'running',
    progress: 0,
    created_at: '',
    updated_at: '',
    source: { account_id: 'account', directory: { id: 'root', name: 'Movies', path: '/Movies' } },
    scan: {
      stage: 'reconciling',
      current_path: '/Movies',
      directories_discovered: 1,
      directories_scanned: 1,
      files_scanned: 4,
      video_files: 4,
      matched_files: 4,
      unmatched_files: 0,
      movies: 4,
      removed_files: 0,
      removed_movies: 0,
      metadata_total: 0,
      metadata_completed: 0
    }
  }
  const offline = offlineSubmission({ phase: 'downloaded', processing: true, scan_task_id: 1 })
  for (const isOffline of [false, true]) {
    if (isOffline) notifyOfflineTask(offline, { scan })
    else notifyScanTask(scan)
    const options = vi.mocked(toast.info).mock.lastCall?.[1]
    assert.ok(isValidElement(options?.description))
    assert.ok(renderToStaticMarkup(options.description).includes('核对中'))
    scan.status = 'queued'
    scan.retry_at = '2026-11-16T00:00:00Z'
    assert.equal(scanStatus(scan), '等待自动重试')
    scan.paused = true
    assert.equal(scanStatus(scan), '已暂停')
    scan.paused = false
    scan.status = 'running'
    scan.retry_at = undefined
  }
})
