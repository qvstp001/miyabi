import { useState } from 'react'
import { useMovieReviews } from '@/api/explore'
import { EmptyState } from '@/components/empty-state'
import { ListPagination } from '@/components/list-pagination'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { PlaylistResults } from '@/features/explore/content'
import { ExploreQueryError } from '@/features/explore/query-error'

// Mount each reader only after the user opens it; browsing details stays lightweight.
export function MovieCommunity({ movieId }: { movieId: string }) {
  const [section, setSection] = useState<'reviews' | 'lists'>()
  return (
    <section className="space-y-4">
      <div className="flex gap-2">
        <Button variant={section === 'reviews' ? 'secondary' : 'outline'} onClick={() => setSection(section === 'reviews' ? undefined : 'reviews')}>评论</Button>
        <Button variant={section === 'lists' ? 'secondary' : 'outline'} onClick={() => setSection(section === 'lists' ? undefined : 'lists')}>相关片单</Button>
      </div>
      {section === 'reviews' ? <MovieReviews key={movieId} movieId={movieId} /> : null}
      {section === 'lists' ? <RelatedPlaylists key={movieId} movieId={movieId} /> : null}
    </section>
  )
}

function RelatedPlaylists({ movieId }: { movieId: string }) {
  const [page, setPage] = useState(1)
  return <PlaylistResults params={{ mode: 'related', movie_id: movieId, page, limit: 10 }} onPageChange={setPage} scrollToTop={false} />
}

function MovieReviews({ movieId }: { movieId: string }) {
  const [page, setPage] = useState(1)
  const query = useMovieReviews(movieId, page)
  if (query.isError) return <ExploreQueryError error={query.error} retry={() => void query.refetch()} retrying={query.isFetching} />
  if (query.isPending) return <Skeleton className="h-40 rounded-2xl" />
  return (
    <div className="space-y-4">
      {query.data.items.length === 0 ? <EmptyState title="暂无评论" /> : (
        <div className="divide-y rounded-2xl border px-4">
          {query.data.items.map(review => (
            <article key={review.id} className="space-y-2 py-4">
              <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
                <span className="font-medium">{review.username || 'JavDB 用户'}</span>
                <span className="text-xs text-muted-foreground">{review.created_at}</span>
              </div>
              {review.score > 0 ? <p className="text-xs text-muted-foreground">评分 {review.score} / 5</p> : null}
              <p className="whitespace-pre-wrap break-words text-sm leading-6">{review.content || '未填写短评'}</p>
              <p className="text-xs text-muted-foreground">{review.likes_count} 人觉得有用</p>
            </article>
          ))}
        </div>
      )}
      <ListPagination page={page} hasMore={query.data.has_more} disabled={query.isFetching} onPageChange={setPage} scrollToTop={false} />
    </div>
  )
}
