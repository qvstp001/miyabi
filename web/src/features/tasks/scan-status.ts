import { isTaskActive, type ScanTask } from '@/api/tasks'

const statusLabels = {
  queued: '等待扫描',
  running: '扫描中',
  done: '处理完成',
  failed: '处理失败'
}

export function scanStatus(task: ScanTask) {
  if (!isTaskActive(task)) return statusLabels[task.status]
  if (task.paused) return task.status === 'running' ? '正在保存进度并暂停' : '已暂停'
  if (task.retry_at) return '等待自动重试'
  if (task.status === 'running' && scanStage(task) === 'reconciling') return '核对中'
  return statusLabels[task.status]
}

export function scanStage(task: ScanTask): ScanTask['scan']['stage'] {
  if (!isTaskActive(task)) return task.status === 'done' ? 'done' : task.scan.stage
  return task.scan.stage === 'done' ? 'reconciling' : task.scan.stage
}

export function scanCount(task: ScanTask) {
  return `识别到 ${task.scan.movies} 部`
}
