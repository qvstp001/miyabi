import { useId, useState } from 'react'
import { toast } from 'sonner'

import { useCreateDownload } from '@/api/downloads'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

export function ResourceForm({ code, movieID }: { code?: string; movieID?: string }) {
  const id = useId()
  const [number, setNumber] = useState(code ?? '')
  const [uri, setURI] = useState('')
  const add = useCreateDownload()
  return (
    <form className="space-y-3" onSubmit={event => {
      event.preventDefault()
      add.mutate({ code: code ?? number, movie_id: movieID, uri }, { onSuccess: task => {
        if (task.status === 'failed') { toast.error(task.error ?? '115 提交失败'); return }
        setURI('')
        toast.success(task.status === 'done' ? '该资源已完成推送' : '已加入下载任务')
      } })
    }}>
      {!code ? <div className="space-y-2"><label htmlFor={`${id}-code`} className="text-sm">影片番号</label><Input id={`${id}-code`} required maxLength={100} value={number} onChange={event => setNumber(event.target.value)} placeholder="例如 ABC-123" /></div> : null}
      <div className="space-y-2"><label htmlFor={`${id}-uri`} className="text-sm">添加 magnet 或 ed2k 链接</label><Input id={`${id}-uri`} required maxLength={16384} value={uri} onChange={event => setURI(event.target.value)} placeholder="magnet:?xt=urn:btih:… 或 ed2k://|file|…" /></div>
      <Button type="submit" disabled={add.isPending || !uri.trim() || !(code ?? number).trim()}>{add.isPending ? '正在提交' : '下载到指定 115 目录'}</Button>
    </form>
  )
}
