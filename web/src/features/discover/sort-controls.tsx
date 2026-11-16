import {
  BROWSE_SORTS,
  type BrowseSort,
  type BrowseSorting,
  type SortOrder
} from './filter-options'
import { FilterSelect } from './filter-select'

export function SortControls({ sort, order, onChange }: {
  sort: BrowseSort
  order: SortOrder
  onChange: (sorting: BrowseSorting) => void
}) {
  return (
    <div className="flex flex-wrap gap-2">
      <FilterSelect
        label="排序"
        value={sort}
        options={BROWSE_SORTS}
        onChange={value => onChange({ sort: value as BrowseSort })}
      />
      <FilterSelect
        label="方向"
        value={order}
        options={[
          { value: 'desc', label: '从高到低 / 从新到旧' },
          { value: 'asc', label: '从低到高 / 从旧到新' }
        ]}
        onChange={value => onChange({ order: value as SortOrder })}
      />
    </div>
  )
}
