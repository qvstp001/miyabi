import type { CategoryFilters as CategoryValues } from './search'
import { BrowseResults } from './browse-results'
import { CategoryFilters } from './category-filters'
import { categoryBrowseParams } from './category-params'
import type { BrowseSort, SortOrder } from './filter-options'

export function CategoryContent({
  category,
  onCategoryChange,
  page,
  onPageChange,
  pageSize,
  sort,
  order
}: {
  category: CategoryValues
  onCategoryChange: (filters: Partial<CategoryValues>) => void
  page: number
  onPageChange: (page: number) => void
  pageSize: number
  sort: BrowseSort
  order: SortOrder
}) {
  return (
    <div className="space-y-6">
      <CategoryFilters category={category} updateCategory={onCategoryChange} />
      <BrowseResults
        params={{ page, limit: pageSize, sort, order, ...categoryBrowseParams(category) }}
        onPageChange={onPageChange}
      />
    </div>
  )
}
