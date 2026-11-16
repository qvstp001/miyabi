import { Children, isValidElement, type ReactElement, type ReactNode } from 'react'
import { expect, test, vi } from 'vitest'

import { useLibraryMovies, useStartLibraryScan } from '@/api/library'
import { useTasks } from '@/api/tasks'
import { LibraryPage } from '@/features/library/page'
import { LibraryScanButton } from '@/features/library/scan-button'
import { notifyScanTask, notifyTaskError } from '@/features/tasks/task-toast'

vi.mock('@/api/library', () => ({
  LIBRARY_PAGE_SIZE: 20,
  useLibraryMovies: vi.fn(),
  useStartLibraryScan: vi.fn()
}))
vi.mock('@/api/tasks', () => ({
  useTasks: vi.fn(),
  isScanTask: () => true,
  isTaskActive: (task: { status: string }) => ['queued', 'running'].includes(task.status)
}))
vi.mock('@/features/tasks/task-events', () => ({
  useTaskConnection: () => ({ status: 'connected' })
}))
vi.mock('@/features/tasks/task-toast', () => {
  const notifyScanTask = vi.fn()
  return { notifyScanTask, useNotifyScanTask: () => notifyScanTask, notifyTaskError: vi.fn() }
})

function elements(node: ReactNode): ReactElement<Record<string, unknown>>[] {
  return Children.toArray(node).filter(isValidElement) as ReactElement<Record<string, unknown>>[]
}

test('scan control retains its header position and component identity while pages load or fail', () => {
  vi.mocked(useTasks).mockReturnValue({ data: [] } as unknown as ReturnType<typeof useTasks>)
  const onPageChange = vi.fn()
  const states = [
    {
      isSuccess: true,
      data: { source: { account_id: '1', directory: { id: '2' } }, movies: [], total: 0 }
    },
    { isPending: true },
    { isError: true },
    { isSuccess: true, data: { movies: [], total: 0 } }
  ]
  for (const state of states) {
    vi.mocked(useLibraryMovies).mockReturnValue(
      state as unknown as ReturnType<typeof useLibraryMovies>
    )
    const page = LibraryPage({ page: 2, onPageChange })
    const header = elements(page.props.children)[0]!
    const control = elements(header.props.children as ReactNode)[0]!
    expect(control.type).toBe(LibraryScanButton)
    expect(control.props.loading).toBe(state.isPending)
    ;(control.props.onStarted as () => void)()
  }
  expect(onPageChange.mock.calls).toEqual([[1], [1], [1], [1]])
})

test('loading and hidden controls still retain the scan mutation hook', () => {
  vi.mocked(useStartLibraryScan).mockReturnValue({ isPending: true } as ReturnType<
    typeof useStartLibraryScan
  >)
  const props = {
    scanning: false,
    connected: true,
    onStarted: vi.fn()
  }
  LibraryScanButton({ ...props, loading: true, available: false })
  expect(useStartLibraryScan).toHaveBeenCalledTimes(1)
  expect(LibraryScanButton({ ...props, loading: false, available: false })).toBeNull()
  expect(useStartLibraryScan).toHaveBeenCalledTimes(2)
})

test('scan submission keeps success navigation and failure notification', () => {
  const mutate = vi.fn()
  vi.mocked(useStartLibraryScan).mockReturnValue({
    isPending: false,
    mutate
  } as unknown as ReturnType<typeof useStartLibraryScan>)
  const onStarted = vi.fn()
  const tree = LibraryScanButton({
    loading: false,
    available: true,
    scanning: false,
    connected: true,
    onStarted
  })!
  const button = elements(tree.props.children)[0]!
  ;(button.props.onClick as () => void)()
  expect(mutate.mock.calls[0]![0]).toBeUndefined()
  const callbacks = mutate.mock.calls[0]![1]
  const task = { id: 123 }
  callbacks.onSuccess(task)
  expect(notifyScanTask).toHaveBeenCalledWith(task)
  expect(onStarted).toHaveBeenCalledOnce()
  callbacks.onError(new Error('request failed'))
  expect(notifyTaskError).toHaveBeenCalledWith(
    'scan:submit-error',
    '无法创建同步任务',
    '请检查后端服务和网络后重试。'
  )
})

test('only the sync control is available and it disables during scanning', () => {
  vi.mocked(useStartLibraryScan).mockReturnValue({
    isPending: false,
    mutate: vi.fn()
  } as unknown as ReturnType<typeof useStartLibraryScan>)
  const props = {
    loading: false,
    available: true,
    scanning: false,
    connected: true,
    onStarted: vi.fn()
  }
  const controls = elements(LibraryScanButton(props)!.props.children)
  expect(controls).toHaveLength(1)
  expect(elements(controls[0]!.props.children as ReactNode).at(-1)?.props.children).toBe('同步媒体库')
  expect(elements(LibraryScanButton({ ...props, scanning: true })!.props.children)[0]!.props.disabled).toBe(true)
})

test('local-only libraries can sync and local import activity keeps the sync control busy', () => {
  const local = { account_id: 'local', directory: { id: 'C:/media', path: 'C:/media' } }
  const remote = { account_id: '100', directory: { id: '10', path: '/Movies' } }
  for (const source of [undefined, remote]) {
    vi.mocked(useLibraryMovies).mockReturnValue({
      isSuccess: true,
      data: { source, local_source: local, movies: [], total: 0 }
    } as unknown as ReturnType<typeof useLibraryMovies>)
    vi.mocked(useTasks).mockReturnValue({
      data: [
        { id: 2, source: source ?? local, status: 'done' },
        { id: 1, source: local, status: 'running' }
      ]
    } as unknown as ReturnType<typeof useTasks>)
    const page = LibraryPage({ page: 1, onPageChange: vi.fn() })
    const header = elements(page.props.children)[0]!
    const control = elements(header.props.children as ReactNode)[0]!
    expect(control.props.available).toBe(true)
    expect(control.props.scanning).toBe(true)
  }
})
