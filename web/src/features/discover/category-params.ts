import type { BrowseMoviesParams } from '@/api/discover'
import type { CategoryFilters } from './search'
import { mainValues } from './filter-options'

export function categoryBrowseParams({ zone, tagIds, year, month, main }: CategoryFilters): BrowseMoviesParams {
  return {
    zone,
    ...(main ? { main: mainValues(main) } : {}),
    ...(tagIds.length ? { tagIds } : {}),
    ...(year ? { year } : {}),
    ...(year && month ? { month } : {})
  }
}
