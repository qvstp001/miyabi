export const BROWSE_SORTS = [
  { value: 'hit', label: '热度' },
  { value: 'release', label: '发行时间' },
  { value: 'score', label: '评分' },
  { value: 'update', label: '更新时间' },
  { value: 'want_watch_count', label: '想看人数' },
  { value: 'watched_count', label: '看过人数' }
] as const

export type BrowseSort = (typeof BROWSE_SORTS)[number]['value']
export type SortOrder = 'asc' | 'desc'
export type BrowseSorting = { sort?: BrowseSort; order?: SortOrder }

export const SEARCH_SORTS = [
  { value: 'default', label: '默认排序' },
  { value: 'release', label: '发行时间' },
  { value: 'score', label: '评分' }
] as const

export const SEARCH_FILTERS = [
  { value: 'all', label: '全部资源' },
  { value: 'magnets', label: '含磁力' },
  { value: 'subtitle', label: '含字幕' },
  { value: 'can_play', label: '可播放' }
] as const

export function validateSorting(search: Record<string, unknown>): BrowseSorting {
  const { sort, order } = search
  if (sort !== undefined && !BROWSE_SORTS.some(item => item.value === sort)) {
    throw new Error('排序条件无效')
  }
  if (order !== undefined && order !== 'asc' && order !== 'desc') {
    throw new Error('排序方向无效')
  }
  return { sort: sort as BrowseSort | undefined, order }
}

export function mainValues(main: string): string[] {
  return main ? main.split(',') : []
}

export function validateMain(main: unknown): string {
  if (main === undefined || main === '') return ''
  if (
    typeof main !== 'string' ||
    main.split(',').some(value => !['p', 'm', 'c', 's', 'i', 'v'].includes(value))
  ) {
    throw new Error('通用筛选无效')
  }
  return [...new Set(main.split(','))].sort().join(',')
}
