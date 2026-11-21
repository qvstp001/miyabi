import { useState } from 'react'
import { DownloadIcon } from 'lucide-react'

import { useDownloadConfig, useSaveDownloadConfig, type DownloadConfig } from '@/api/downloads'
import { usePanAccount } from '@/api/pan'
import { InlineError } from '@/components/error-state'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { SettingsSection } from './shared'

export function DownloadSection() {
  const config = useDownloadConfig()
  const account = usePanAccount()
  const accountID = account.data?.connected ? account.data.account?.id ?? '' : ''
  const directoryID = account.data?.directory?.id ?? ''
  const ready = !!accountID && !!directoryID
  return <SettingsSection title="LitePan 对接" icon={<DownloadIcon className="size-4" />}>
    {config.isPending ? <p>正在读取设置…</p> : config.isError ? <InlineError onRetry={() => void config.refetch()}>设置读取失败</InlineError> : <DownloadConfigForm key={`${config.data.litepan_url}:${config.data.path}:${config.data.has_api_key}:${config.data.account_id}:${config.data.directory_id}:${accountID}:${directoryID}`} initial={config.data} sameTarget={ready && config.data.account_id === accountID && config.data.directory_id === directoryID} accountID={accountID} directoryID={directoryID} ready={ready} />}
  </SettingsSection>
}

function DownloadConfigForm({ initial, sameTarget, ready, accountID, directoryID }: { initial: DownloadConfig; sameTarget: boolean; ready: boolean; accountID: string; directoryID: string }) {
  const [url, setURL] = useState(initial.litepan_url)
  const [path, setPath] = useState(sameTarget ? initial.path : '')
  const [key, setKey] = useState('')
  const save = useSaveDownloadConfig()
  return <form className="space-y-4" onSubmit={event => {
    event.preventDefault()
    save.mutate({ litepan_url: url, path, api_key: key, has_api_key: false, account_id: accountID, directory_id: directoryID }, { onSuccess: () => setKey('') })
  }}>
    {!ready ? <p className="text-sm text-muted-foreground">请先登录 115 并选择下载目录。</p> : !sameTarget ? <p className="text-sm text-muted-foreground">请填写当前 115 下载目录在 LitePan 中的对应路径，之前的目录绑定不会自动沿用。</p> : null}
    <div className="space-y-2"><label className="text-sm" htmlFor="litepan-url">LitePan 地址</label><Input id="litepan-url" type="url" required placeholder="http://192.168.50.97:5211" value={url} onChange={event => setURL(event.target.value)} /></div>
    <div className="space-y-2"><label className="text-sm" htmlFor="litepan-key">任务 API Key</label><Input id="litepan-key" type="password" autoComplete="new-password" placeholder={initial.has_api_key ? '已保存，留空保留' : '在 LitePan 中创建任务 API Key'} value={key} onChange={event => setKey(event.target.value)} /></div>
    <div className="space-y-2"><label className="text-sm" htmlFor="litepan-path">下载目录在 LitePan 中的路径</label><Input id="litepan-path" required placeholder="/115/影视" value={path} onChange={event => setPath(event.target.value)} /></div>
    <p className="text-sm text-muted-foreground">在 LitePan 中创建 Webhook 自动化规则：事件为 <code>miyabi.download.completed</code>，来源为 <code>miyabi</code>，动作为运行覆盖此目录的 STRM 任务。此处路径须与上方 115 下载目录对应。</p>
    <Button type="submit" disabled={save.isPending || !ready}>{save.isPending ? '保存中' : '保存 LitePan 设置'}</Button>
    {save.isSuccess ? <p className="text-sm text-muted-foreground">设置已保存</p> : null}
  </form>
}
