import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { ApiError, apiGet, apiPost, apiPut } from './client'
import { usePanAccount } from './pan'

export type DownloadInput = { code: string; movie_id?: string; uri: string }
export type DownloadTask = {
  id: number
  code: string
  movie_id: string
  uri: string
  status: 'running' | 'done' | 'failed' | 'cancelled'
  stage: 'submitting' | 'downloading' | 'locating' | 'planning' | 'organizing' | 'pushing' | 'cancelling' | 'done'
  progress: number
  error?: string
  files: { id: string; name: string }[] | null
  account_id: string
  directory_id: string
}
export type DownloadConfig = { litepan_url: string; path: string; api_key?: string; has_api_key: boolean; account_id?: string; directory_id?: string }

export const downloadStageLabels: Record<DownloadTask['stage'], string> = {
  submitting: '正在提交到 115',
  downloading: '115 下载中',
  locating: '等待 115 返回文件位置',
  planning: '正在读取下载结果',
  organizing: '正在重命名和移动文件',
  pushing: '正在推送 LitePan',
  cancelling: '正在取消 115 任务',
  done: '已推送 LitePan'
}

export function useDownloads() {
  return useQuery({
    queryKey: ['downloads'],
    queryFn: ({ signal }) => apiGet<DownloadTask[]>('/api/downloads', undefined, signal),
    refetchInterval: query => query.state.data?.some(task => task.status === 'running') ? 5000 : false,
    refetchOnMount: 'always'
  })
}

export function useCreateDownload() {
  const client = useQueryClient()
  const account = usePanAccount()
  return useMutation({
    mutationFn: (input: DownloadInput) => {
      const accountID = account.data?.account?.id
      const directoryID = account.data?.directory?.id
      if (!account.data?.connected || !accountID || !directoryID) {
        throw new ApiError('请先登录 115 并选择下载目录', 400, 'PAN_DIRECTORY_REQUIRED')
      }
      return apiPost<DownloadTask>('/api/downloads', { ...input, account_id: accountID, directory_id: directoryID })
    },
    onSettled: () => client.invalidateQueries({ queryKey: ['downloads'] }),
    meta: { errorTitle: '提交下载失败' }
  })
}

export function useDownloadAction(action: 'retry' | 'cancel') {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiPost<DownloadTask>(`/api/downloads/${id}/${action}`),
    onSettled: () => client.invalidateQueries({ queryKey: ['downloads'] }),
    meta: { errorTitle: action === 'retry' ? '重试失败' : '取消失败' }
  })
}

export function useDownloadConfig() {
  return useQuery({ queryKey: ['download-settings'], queryFn: ({ signal }) => apiGet<DownloadConfig>('/api/settings/downloads', undefined, signal) })
}

export function useSaveDownloadConfig() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (config: DownloadConfig) => apiPut<DownloadConfig>('/api/settings/downloads', config),
    onSuccess: config => client.setQueryData(['download-settings'], config),
    meta: { errorTitle: '保存 LitePan 设置失败' }
  })
}
