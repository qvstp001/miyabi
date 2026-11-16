import type { JavDBZone } from '@/api/discover'
import { DISCOVER_ZONES } from '@/features/discover/constants'
import { SEARCH_SORTS, SEARCH_FILTERS } from '@/features/discover/filter-options'
import { parseSearchPage } from '@/lib/search-schema'

export type SearchFilters = {
  zone?: JavDBZone
  sort?: 'release' | 'score'
  filter?: 'magnets' | 'subtitle' | 'can_play'
}
export type SearchParams = SearchFilters & { q?: string; page?: number }

export function validateSearchParams(search: Record<string, unknown>): SearchParams {
  const { zone, sort, filter } = search
  const q = search.q ?? ''
  if (typeof q !== 'string') throw new Error('搜索条件无效')
  if (zone !== undefined && !DISCOVER_ZONES.some(item => item.value === zone)) {
    throw new Error('搜索分区无效')
  }
  if (sort !== undefined && !SEARCH_SORTS.some(item => item.value === sort && item.value !== 'default')) {
    throw new Error('搜索排序无效')
  }
  if (filter !== undefined && !SEARCH_FILTERS.some(item => item.value === filter && item.value !== 'all')) {
    throw new Error('搜索资源条件无效')
  }
  return {
    q: q.trim() || undefined,
    page: parseSearchPage(search.page),
    zone: zone as JavDBZone | undefined,
    sort: sort as SearchFilters['sort'],
    filter: filter as SearchFilters['filter']
  }
}
