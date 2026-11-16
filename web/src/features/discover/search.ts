import type { JavDBZone } from '@/api/discover'
import { parseSearchPage } from '@/lib/search-schema'
import { DISCOVER_ZONES } from './constants'
import { validateMain, validateSorting, type BrowseSorting } from './filter-options'

export type DiscoverView = 'released' | 'upcoming' | 'category'
export type CategoryFilters = {
  zone: JavDBZone
  tagIds: string[]
  year: string
  month: string
  main: string
}
export type DiscoverSearch = Partial<CategoryFilters> & BrowseSorting & {
  view?: DiscoverView
  releasedPage?: number
  upcomingPage?: number
  categoryPage?: number
}

export function validateDiscoverSearch(search: Record<string, unknown>): DiscoverSearch {
  const { view, zone } = search
  if (view !== undefined && view !== 'released' && view !== 'upcoming' && view !== 'category') {
    throw new Error('浏览类型无效')
  }
  if (zone !== undefined && !DISCOVER_ZONES.some(item => item.value === zone)) {
    throw new Error('影片分区无效')
  }
  // Accept links generated before combination filters were introduced.
  if (
    (search.categoryID !== undefined && typeof search.categoryID !== 'string') ||
    (search.tagID !== undefined && typeof search.tagID !== 'string' &&
      !(typeof search.tagID === 'number' && Number.isSafeInteger(search.tagID)))
  ) {
    throw new Error('分类筛选无效')
  }
  const legacyTag = search.tagID === undefined ? '' : String(search.tagID)
  const rawTags = search.tagIds ?? (legacyTag && search.categoryID !== 'year' ? [legacyTag] : [])
  if (!Array.isArray(rawTags) || rawTags.some(tag => typeof tag !== 'string' || !tag.trim())) {
    throw new Error('标签筛选无效')
  }
  const tagIds = [...new Set(rawTags as string[])].sort()
  // The router parses numeric URL query values, including years and months, as numbers.
  const year = String(search.year ?? (search.categoryID === 'year' ? legacyTag : ''))
  const month = String(search.month ?? '')
  if (year && !/^\d{4}$/.test(year)) throw new Error('年份无效')
  if (month && (!year || !/^(?:[1-9]|1[0-2])$/.test(month))) {
    throw new Error('请先选择年份，再选择月份')
  }
  const main = validateMain(search.main)
  return {
    view: view === 'upcoming' || view === 'category' ? view : undefined,
    releasedPage: parseSearchPage(search.releasedPage),
    upcomingPage: parseSearchPage(search.upcomingPage),
    categoryPage: parseSearchPage(search.categoryPage),
    zone: zone && zone !== 'censored' ? zone as JavDBZone : undefined,
    tagIds: tagIds.length ? tagIds : undefined,
    year: year || undefined,
    month: month || undefined,
    main: search.main === undefined ? undefined : main,
    ...validateSorting(search)
  }
}

export function categoryFromSearch(search: DiscoverSearch): CategoryFilters {
  return {
    zone: search.zone ?? 'censored',
    tagIds: search.tagIds ?? [],
    year: search.year ?? '',
    month: search.month ?? '',
    main: search.main ?? ''
  }
}

export function updateCategorySearch(
  search: DiscoverSearch,
  filters: Partial<CategoryFilters>
): DiscoverSearch {
  const mainChanged = filters.main !== undefined && filters.main !== search.main
  return validateDiscoverSearch({
    ...search,
    ...filters,
    categoryPage: undefined,
    ...(mainChanged ? { releasedPage: undefined, upcomingPage: undefined } : {})
  })
}
