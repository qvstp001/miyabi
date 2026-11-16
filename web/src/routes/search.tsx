import { createFileRoute } from '@tanstack/react-router'

import { AppPage } from '@/components/app-page'
import { ErrorState } from '@/components/error-state'
import { SearchPage } from '@/features/search/page'
import { validateSearchParams } from '@/features/search/search'

export const Route = createFileRoute('/search')({
  validateSearch: validateSearchParams,
  component: SearchRoute,
  errorComponent: () => (
    <AppPage>
      <ErrorState message="搜索条件无效" />
    </AppPage>
  )
})

function SearchRoute() {
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <SearchPage
      keyword={search.q ?? ''}
      page={search.page ?? 1}
      filters={search}
      onSearch={q => void navigate({
        search: previous => ({ ...previous, q: q || undefined, page: undefined })
      })}
      onFilterChange={filters => void navigate({
        search: previous => ({ ...previous, ...filters, page: undefined })
      })}
      onPageChange={page => void navigate({
        search: previous => ({ ...previous, page: page > 1 ? page : undefined }),
        resetScroll: false
      })}
    />
  )
}
