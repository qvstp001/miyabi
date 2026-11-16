import { Link, getRouteApi } from '@tanstack/react-router'
import { useDiscoverTags } from '@/api/discover'
import { AppPage } from '@/components/app-page'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { InlineError } from '@/components/error-state'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useCatalogueDisplay } from '@/stores/catalogue-display'
import { categoryFromSearch, updateCategorySearch, type DiscoverView } from './search'
import { BrowseResults } from './browse-results'
import { CategoryContent } from './category-content'
import { CommonFilterSelect } from './common-filter-select'
import { CatalogueDisplaySettings } from './display-settings'
import { SortControls } from './sort-controls'
import { mainValues } from './filter-options'

const route = getRouteApi('/discover')

export function DiscoverPage() {
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const pageSize = useCatalogueDisplay(state => state.pageSize)
  const view = search.view ?? 'released'
  const taxonomy = useDiscoverTags('censored', view !== 'category')
  const page = search[`${view}Page`] ?? 1
  const sort = search.sort ?? (view === 'released' ? 'update' : 'release')
  const order = search.order ?? 'desc'
  const main = search.main ?? (view === 'released' ? 'm' : '')
  const setPage = (view: DiscoverView, page: number) =>
    void navigate({
      search: previous => ({ ...previous, [`${view}Page`]: page > 1 ? page : undefined })
    })
  const resetPages = () => void navigate({
    search: previous => ({
      ...previous, releasedPage: undefined, upcomingPage: undefined, categoryPage: undefined
    })
  })

  return (
    <AppPage>
      <PageHeader title="发现" description="浏览最新发行、即将发行和分类内容">
        <Button asChild variant="outline"><Link to="/discover/explore">排行榜与片单</Link></Button>
      </PageHeader>
      <div className="min-w-0 space-y-6">
        <Tabs
          value={view}
          onValueChange={value => void navigate({
            search: previous => ({
              ...previous, view: value === 'released' ? undefined : value as DiscoverView
            })
          })}
        >
          <TabsList>
            <TabsTrigger value="released">最新发行</TabsTrigger>
            <TabsTrigger value="upcoming">即将发行</TabsTrigger>
            <TabsTrigger value="category">分类浏览</TabsTrigger>
          </TabsList>
        </Tabs>
        <div className="flex flex-wrap items-end justify-between gap-3">
          <SortControls
            sort={sort}
            order={order}
            onChange={sorting => void navigate({
              search: previous => ({
                ...previous,
                ...sorting,
                releasedPage: undefined,
                upcomingPage: undefined,
                categoryPage: undefined
              })
            })}
          />
          <CatalogueDisplaySettings onPageSizeChange={resetPages} />
        </div>
        {view === 'category' ? (
          <CategoryContent
            category={categoryFromSearch(search)}
            onCategoryChange={filters => void navigate({
              search: previous => updateCategorySearch(previous, filters)
            })}
            page={page}
            onPageChange={next => setPage('category', next)}
            pageSize={pageSize}
            sort={sort}
            order={order}
          />
        ) : (
          <>
            {taxonomy.isError ? (
              <InlineError onRetry={() => void taxonomy.refetch()} retrying={taxonomy.isFetching}>
                资源筛选加载失败
              </InlineError>
            ) : (
              <CommonFilterSelect
                options={taxonomy.data?.find(item => item.id === 'main')?.tags ?? []}
                value={main}
                onValueChange={main => void navigate({
                  search: previous => ({
                    ...previous,
                    main,
                    releasedPage: undefined,
                    upcomingPage: undefined,
                    categoryPage: undefined
                  })
                })}
              />
            )}
            <BrowseResults
              key={view}
              params={{ page, limit: pageSize, order, sort, main: mainValues(main) }}
              onPageChange={next => setPage(view, next)}
            />
          </>
        )}
      </div>
    </AppPage>
  )
}
