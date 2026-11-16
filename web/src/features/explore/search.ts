import type { FeedParams } from '@/api/explore'
import { parseSearchPage } from '@/lib/search-schema'
import { validateSorting, type BrowseSorting } from '@/features/discover/filter-options'

export const EXPLORE_VIEWS = [
  { value: 'ranking', label: '排行榜' },
  { value: 'top250', label: 'TOP250' },
  { value: 'lists', label: '片单' },
  { value: 'actors', label: '演员搜索' },
  { value: 'personal', label: '想看 / 看过' }
] as const
export type ExploreView = (typeof EXPLORE_VIEWS)[number]['value']
export type ExploreSearch = BrowseSorting & {
  view?: ExploreView
  zone?: FeedParams['zone']
  period?: FeedParams['period']
  status?: FeedParams['status']
  listMode?: 'search' | 'mine'
  listID?: string
  q?: string
  page?: number
}

export function validateExploreSearch(raw: Record<string, unknown>): ExploreSearch {
  const { view, zone, period, status, listMode, q } = raw
  const listID = typeof raw.listID === 'number' && Number.isSafeInteger(raw.listID)
    ? String(raw.listID)
    : raw.listID
  if (view !== undefined && !EXPLORE_VIEWS.some(item => item.value === view)) throw new Error('内容入口无效')
  if (zone !== undefined && (typeof zone !== 'string' || !['censored', 'uncensored', 'western', 'fc2'].includes(zone))) throw new Error('分区无效')
  if (period !== undefined && (typeof period !== 'string' || !['daily', 'weekly', 'monthly'].includes(period))) throw new Error('榜单周期无效')
  if (status !== undefined && status !== 'want_watch' && status !== 'watched') throw new Error('个人列表类型无效')
  if (listMode !== undefined && listMode !== 'search' && listMode !== 'mine') throw new Error('片单类型无效')
  if (listID !== undefined && (typeof listID !== 'string' || !/^[a-zA-Z0-9_-]+$/.test(listID))) throw new Error('片单 ID 无效')
  if (q !== undefined && typeof q !== 'string') throw new Error('关键词无效')
  return {
    view: view as ExploreSearch['view'],
    zone: zone as ExploreSearch['zone'],
    period: period as ExploreSearch['period'],
    status,
    listMode,
    listID,
    q: typeof q === 'string' ? q.trim() || undefined : undefined,
    page: parseSearchPage(raw.page),
    ...validateSorting(raw)
  }
}

export function updateExploreSearch(search: ExploreSearch, changes: Partial<ExploreSearch>): ExploreSearch {
  return validateExploreSearch({ ...search, ...changes, page: undefined })
}
