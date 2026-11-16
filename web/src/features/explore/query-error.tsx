import { Link } from '@tanstack/react-router'
import { describeApiError, isJavDBLoginRequired } from '@/api/client'
import { ErrorState } from '@/components/error-state'
import { Button } from '@/components/ui/button'

export function ExploreQueryError({ error, retry, retrying }: {
  error: Error
  retry: () => void
  retrying: boolean
}) {
  if (isJavDBLoginRequired(error)) {
    return (
      <div className="space-y-3 rounded-2xl border p-6 text-center">
        <p className="font-medium">此功能需要登录 JavDB</p>
        <p className="text-sm text-muted-foreground">请在设置中保存 JavDB 账号，再回来重试。</p>
        <div className="flex justify-center gap-2">
          <Button asChild variant="outline"><Link to="/settings">去设置</Link></Button>
          <Button onClick={retry} disabled={retrying}>重试</Button>
        </div>
      </div>
    )
  }
  return <ErrorState message={describeApiError(error)} onRetry={retry} retrying={retrying} />
}
