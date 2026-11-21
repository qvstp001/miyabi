import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { toast } from 'sonner'

import { useDiscoverMagnets } from '@/api/discover'
import { downloadStageLabels, useCreateDownload, useDownloads } from '@/api/downloads'
import { usePanAccount } from '@/api/pan'
import { InlineError } from '@/components/error-state'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { ResourceForm } from '@/features/downloads/resource-form'
import { formatSize } from '@/lib/format'

export function MovieMagnets({ movieID, code }: { movieID: string; code: string }) {
  const [open, setOpen] = useState(false)
  const query = useDiscoverMagnets(movieID, open)
  const account = usePanAccount()
  const downloads = useDownloads()
  const add = useCreateDownload()
  const canDownload = account.data?.connected && !!account.data.directory
  return <section className="space-y-4">
    <h2 className="text-xl font-semibold">下载资源</h2>
    <p className="text-sm text-muted-foreground">按需读取当前影片的资源；也可直接粘贴你找到的 magnet 或 ed2k 链接。</p>
    {!canDownload ? <p className="text-sm">请先在 <Link to="/settings" className="underline">设置</Link> 中登录 115、选择下载目录并配置 LitePan。</p> : null}
    <Button variant="outline" onClick={() => setOpen(value => !value)}>{open ? '收起资源' : '查看影片资源'}</Button>
    {open ? query.isPending ? <p>正在读取资源…</p> : query.isError ? <InlineError onRetry={() => void query.refetch()} retrying={query.isFetching}>资源读取失败，请稍后手动重试。</InlineError> : query.data.length === 0 ? <p className="text-sm text-muted-foreground">暂无资源，可在下方添加链接。</p> : query.data.map(resource => {
      const task = downloads.data?.find(task => task.account_id === account.data?.account?.id && task.directory_id === account.data?.directory?.id && task.uri === resource.uri && task.status !== 'cancelled')
      return <Card key={resource.hash}><CardContent className="space-y-3">
        <h3 className="font-medium break-all">{resource.name}</h3>
        <p className="text-xs text-muted-foreground">{formatSize(resource.size)} {resource.tags?.join(' · ')}</p>
        <p className="break-all font-mono text-xs select-all">{resource.uri}</p>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" variant="outline" onClick={() => void navigator.clipboard.writeText(resource.uri).then(() => toast.success('已复制链接')).catch(() => toast.error('复制失败'))}>复制链接</Button>
          <Button size="sm" disabled={!canDownload || add.isPending || !!task} onClick={() => add.mutate({ code, movie_id: movieID, uri: resource.uri }, { onSuccess: task => task.status === 'failed' ? toast.error(task.error) : toast.success(task.status === 'done' ? '该资源已完成推送' : '已加入下载任务') })}>{task ? task.status === 'failed' ? '任务失败，请到下载页重试' : downloadStageLabels[task.stage] : '下载到 115'}</Button>
          {task ? <Button asChild variant="outline" size="sm"><Link to="/downloads">查看任务</Link></Button> : null}
        </div>
      </CardContent></Card>
    }) : null}
    <ResourceForm code={code} movieID={movieID} />
  </section>
}
