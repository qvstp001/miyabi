import { Link } from '@tanstack/react-router'
import { useState, type FormEvent } from 'react'
import {
  useActorSearch, useExploreFeed, usePlaylist, usePlaylists,
  type FeedParams, type ListParams, type Playlist
} from '@/api/explore'
import { EmptyState } from '@/components/empty-state'
import { MediaImage } from '@/components/media-image'
import { ListPagination } from '@/components/list-pagination'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { DiscoverResults } from '@/features/discover/results'
import { ExploreQueryError } from './query-error'

export function ExploreSearchForm({ keyword, placeholder, onSearch }: {
  keyword: string
  placeholder: string
  onSearch: (keyword: string) => void
}) {
  const [draft, setDraft] = useState(keyword)
  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    onSearch(draft.trim())
  }
  return (
    <form onSubmit={submit} className="flex max-w-xl gap-2">
      <Input
        type="search"
        enterKeyHint="search"
        value={draft}
        onChange={event => setDraft(event.target.value)}
        placeholder={placeholder}
      />
      <Button type="submit">搜索</Button>
    </form>
  )
}

export function ExploreFeedResults({ params, onPageChange }: {
  params: FeedParams
  onPageChange: (page: number) => void
}) {
  const query = useExploreFeed(params)
  if (query.isError) {
    return <ExploreQueryError error={query.error} retry={() => void query.refetch()} retrying={query.isFetching} />
  }
  return (
    <div className="space-y-3">
      {query.data?.generated_at ? <p className="text-xs text-muted-foreground">榜单更新：{query.data.generated_at}</p> : null}
      <DiscoverResults
        movies={query.data?.items}
        loading={query.isPending}
        fetching={query.isFetching}
        error={false}
        searching={false}
        hasMore={query.data?.has_more}
        page={params.page}
        onPageChange={onPageChange}
        onRetry={() => void query.refetch()}
      />
    </div>
  )
}

export function PlaylistCards({ lists }: { lists: Playlist[] }) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {lists.map(list => (
        <Link
          key={list.id}
          to="/discover/explore"
          search={{ view: 'lists', listID: list.id }}
          className="rounded-2xl outline-ring"
        >
          <Card className="h-full transition-colors hover:bg-muted/50">
            <CardContent className="space-y-3">
              <h3 className="break-words font-semibold">{list.name || '未命名片单'}</h3>
              {list.description ? <p className="line-clamp-3 whitespace-pre-wrap break-words text-sm text-muted-foreground">{list.description}</p> : null}
              <p className="text-xs text-muted-foreground">{list.movies_count} 部 · {list.views_count} 次浏览 · {list.collections_count} 次收藏</p>
            </CardContent>
          </Card>
        </Link>
      ))}
    </div>
  )
}

export function PlaylistResults({ params, onPageChange, scrollToTop = true }: {
  params: ListParams
  onPageChange: (page: number) => void
  scrollToTop?: boolean
}) {
  const query = usePlaylists(params)
  if (query.isError) {
    return <ExploreQueryError error={query.error} retry={() => void query.refetch()} retrying={query.isFetching} />
  }
  if (query.isPending) return <Skeleton className="h-40 rounded-2xl" />
  return (
    <div className="space-y-4">
      {query.data.items.length ? <PlaylistCards lists={query.data.items} /> : <EmptyState title="暂无片单" />}
      <ListPagination
        page={params.page}
        hasMore={query.data.has_more}
        disabled={query.isFetching}
        onPageChange={onPageChange}
        scrollToTop={scrollToTop}
      />
    </div>
  )
}

export function PlaylistDetail({ id, params, onPageChange }: {
  id: string
  params: FeedParams
  onPageChange: (page: number) => void
}) {
  const query = usePlaylist(id)
  if (query.isError) return <ExploreQueryError error={query.error} retry={() => void query.refetch()} retrying={query.isFetching} />
  if (query.isPending) return <Skeleton className="h-40 rounded-2xl" />
  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <h2 className="break-words text-xl font-semibold">{query.data.name || '未命名片单'}</h2>
        <p className="whitespace-pre-wrap break-words text-sm text-muted-foreground">{query.data.description}</p>
        <p className="text-xs text-muted-foreground">{query.data.movies_count} 部作品</p>
      </div>
      <ExploreFeedResults params={params} onPageChange={onPageChange} />
    </div>
  )
}

export function ActorSearchResults({ keyword }: { keyword: string }) {
  const query = useActorSearch(keyword)
  if (!keyword) return <EmptyState title="输入演员名称开始搜索" />
  if (query.isError) return <ExploreQueryError error={query.error} retry={() => void query.refetch()} retrying={query.isFetching} />
  if (query.isPending) return <Skeleton className="h-40 rounded-2xl" />
  if (!query.data?.length) return <EmptyState title="没有找到演员" />
  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
      {query.data.map(actor => (
        <Link
          key={actor.id}
          to="/discover/search"
          search={{
            kind: 'actor',
            id: actor.id,
            name: actor.name_zht || actor.name || actor.id,
            page: 1,
            main: ''
          }}
          className="overflow-hidden rounded-2xl border outline-ring"
        >
          <div className="aspect-square">
            <MediaImage source={actor.avatar} className="object-cover" />
          </div>
          <div className="space-y-1 p-3">
            <h3 className="break-words text-sm font-semibold">{actor.name_zht || actor.name || actor.id}</h3>
            {actor.name_zht && actor.name !== actor.name_zht ? <p className="break-words text-xs text-muted-foreground">{actor.name}</p> : null}
            <p className="text-xs text-muted-foreground">{actor.videos_count} 部作品</p>
          </div>
        </Link>
      ))}
    </div>
  )
}
