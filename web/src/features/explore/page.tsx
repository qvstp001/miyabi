import { getRouteApi } from '@tanstack/react-router'
import { AppPage } from '@/components/app-page'
import { EmptyState } from '@/components/empty-state'
import { PageBackButton } from '@/components/page-back-button'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { CatalogueDisplaySettings } from '@/features/discover/display-settings'
import { DISCOVER_ZONES } from '@/features/discover/constants'
import { FilterSelect } from '@/features/discover/filter-select'
import { SortControls } from '@/features/discover/sort-controls'
import { useCatalogueDisplay } from '@/stores/catalogue-display'
import {
  ActorSearchResults,
  ExploreFeedResults,
  ExploreSearchForm,
  PlaylistDetail,
  PlaylistResults
} from './content'
import { EXPLORE_VIEWS, updateExploreSearch, type ExploreSearch, type ExploreView } from './search'

const route = getRouteApi('/discover_/explore')

export function ExplorePage() {
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const limit = useCatalogueDisplay(state => state.pageSize)
  const view = search.view ?? 'ranking'
  const page = search.page ?? 1
  const change = (changes: Partial<ExploreSearch>) => void navigate({
    search: previous => updateExploreSearch(previous, changes)
  })
  const setPage = (page: number) => void navigate({
    search: previous => ({ ...previous, page: page > 1 ? page : undefined })
  })

  return (
    <AppPage>
      <PageBackButton />
      <PageHeader title="排行榜与片单" description="浏览 JavDB 榜单、片单、演员和个人收藏" />
      <div className="max-w-full overflow-x-auto pb-1">
        <Tabs
          value={view}
          onValueChange={value => change({
            view: value as ExploreView,
            q: undefined,
            listID: undefined,
            sort: undefined,
            order: undefined
          })}
        >
          <TabsList>
            {EXPLORE_VIEWS.map(item => (
              <TabsTrigger key={item.value} value={item.value}>{item.label}</TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="flex flex-wrap items-end gap-2">
          {view === 'ranking' ? (
            <>
              <FilterSelect
                label="分区"
                value={search.zone ?? 'censored'}
                options={DISCOVER_ZONES.filter(item => item.value !== 'anime')}
                onChange={value => change({ zone: value as ExploreSearch['zone'] })}
              />
              <FilterSelect
                label="周期"
                value={search.period ?? 'weekly'}
                options={[
                  { value: 'daily', label: '日榜' },
                  { value: 'weekly', label: '周榜' },
                  { value: 'monthly', label: '月榜' }
                ]}
                onChange={value => change({ period: value as ExploreSearch['period'] })}
              />
            </>
          ) : null}
          {view === 'personal' ? (
            <FilterSelect
              label="个人列表"
              value={search.status ?? 'want_watch'}
              options={[
                { value: 'want_watch', label: '想看' },
                { value: 'watched', label: '看过' }
              ]}
              onChange={value => change({ status: value as ExploreSearch['status'] })}
            />
          ) : null}
          {view === 'lists' && !search.listID ? (
            <FilterSelect
              label="片单来源"
              value={search.listMode ?? 'search'}
              options={[
                { value: 'search', label: '搜索片单' },
                { value: 'mine', label: '我的片单' }
              ]}
              onChange={value => change({ listMode: value as ExploreSearch['listMode'] })}
            />
          ) : null}
          {view === 'lists' && search.listID ? (
            <>
              <Button variant="outline" onClick={() => change({ listID: undefined })}>返回片单列表</Button>
              <SortControls
                sort={search.sort ?? 'release'}
                order={search.order ?? 'desc'}
                onChange={change}
              />
            </>
          ) : null}
        </div>
        <CatalogueDisplaySettings onPageSizeChange={() => setPage(1)} />
      </div>
      {view === 'ranking' || view === 'top250' ? (
        <ExploreFeedResults
          params={{ kind: view, zone: search.zone ?? 'censored', period: search.period ?? 'weekly', page, limit }}
          onPageChange={setPage}
        />
      ) : null}
      {view === 'personal' ? (
        <>
          <p className="text-sm text-muted-foreground">从 JavDB 账号读取，按标记时间排序；“看过”与本地“已浏览”标记独立。</p>
          <ExploreFeedResults
            params={{ kind: 'personal', status: search.status ?? 'want_watch', page, limit }}
            onPageChange={setPage}
          />
        </>
      ) : null}
      {view === 'actors' ? (
        <>
          <ExploreSearchForm
            key={search.q ?? ''}
            keyword={search.q ?? ''}
            placeholder="搜索演员名称"
            onSearch={q => change({ q })}
          />
          <ActorSearchResults keyword={search.q ?? ''} />
        </>
      ) : null}
      {view === 'lists' ? (
        search.listID ? (
          <PlaylistDetail
            id={search.listID}
            params={{
              kind: 'list',
              id: search.listID,
              sort: search.sort ?? 'release',
              order: search.order ?? 'desc',
              page,
              limit
            }}
            onPageChange={setPage}
          />
        ) : (
          <>
            {search.listMode !== 'mine' ? (
              <ExploreSearchForm
                key={search.q ?? ''}
                keyword={search.q ?? ''}
                placeholder="搜索公开片单"
                onSearch={q => change({ q })}
              />
            ) : null}
            {search.listMode === 'mine' || search.q ? (
              <PlaylistResults
                params={{ mode: search.listMode ?? 'search', q: search.q, page, limit }}
                onPageChange={setPage}
              />
            ) : <EmptyState title="输入关键词搜索片单，或切换到我的片单" />}
          </>
        )
      ) : null}
    </AppPage>
  )
}
