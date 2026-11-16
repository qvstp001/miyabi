import type { DiscoverMovie } from '@/api/discover'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { ListPagination } from '@/components/list-pagination'
import { MovieGridSkeleton } from '@/components/movie'
import { useCatalogueDisplay } from '@/stores/catalogue-display'
import { MovieGrid } from './movie-grid'

export function DiscoverResults({
  movies,
  hasMore,
  loading,
  fetching,
  error,
  errorMessage,
  searching,
  page,
  onPageChange,
  onRetry
}: {
  hasMore?: boolean
  movies: DiscoverMovie[] | undefined
  loading: boolean
  fetching: boolean
  error: boolean
  errorMessage?: string
  searching: boolean
  page: number
  onPageChange: (page: number) => void
  onRetry: () => void
}) {
  const pageSize = useCatalogueDisplay(state => state.pageSize)
  const density = useCatalogueDisplay(state => state.density)
  if (error) {
    return (
      <ErrorState
        message={errorMessage ?? '数据加载失败，请检查网络或代理设置'}
        onRetry={onRetry}
        retrying={fetching}
      />
    )
  }
  if (loading || !movies) return <MovieGridSkeleton count={pageSize} density={density} />
  return (
    <div className="flex flex-col gap-6">
      {movies.length === 0 && page === 1 ? (
        <EmptyState title={searching ? '没有搜索结果' : '暂无内容'} />
      ) : (
        <>
          <MovieGrid movies={movies} />
          <ListPagination
            page={page}
            hasMore={hasMore ?? (movies.length >= pageSize)}
            disabled={fetching}
            onPageChange={onPageChange}
          />
        </>
      )}
    </div>
  )
}
