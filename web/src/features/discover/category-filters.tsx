import { useState } from 'react'
import { XIcon } from 'lucide-react'
import { useDiscoverTags, type JavDBZone } from '@/api/discover'
import { InlineError } from '@/components/error-state'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Skeleton } from '@/components/ui/skeleton'
import type { CategoryFilters as CategoryValues } from './search'
import { CommonFilterSelect } from './common-filter-select'
import { FilterSelect } from './filter-select'
import { DISCOVER_ZONES } from './constants'

export function CategoryFilters({ category, updateCategory }: {
  category: CategoryValues
  updateCategory: (filters: Partial<CategoryValues>) => void
}) {
  const taxonomy = useDiscoverTags(category.zone)
  const [keyword, setKeyword] = useState('')
  const categories = (taxonomy.data ?? []).filter(
    item => !['main', 'year', 'month', 'duration'].includes(item.id)
  )
  const tagNames = new Map(
    categories.flatMap(group => group.tags.map(tag => [tag.id, tag.name] as const))
  )
  const years = taxonomy.data?.find(item => item.id === 'year')?.tags ?? []
  const mainOptions = taxonomy.data?.find(item => item.id === 'main')?.tags ?? []
  const filtered = categories.map(group => ({
    ...group,
    tags: group.tags.filter(tag => tag.name.toLowerCase().includes(keyword.trim().toLowerCase()))
  })).filter(group => group.tags.length)
  const yearOptions = [
    { value: 'all', label: '全部年份' },
    ...years.map(year => ({ value: year.id, label: year.name }))
  ]
  if (category.year && !years.some(year => year.id === category.year)) {
    yearOptions.push({ value: category.year, label: category.year })
  }
  const toggleTag = (id: string) => updateCategory({
    tagIds: category.tagIds.includes(id)
      ? category.tagIds.filter(tag => tag !== id)
      : [...category.tagIds, id]
  })

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end gap-2">
        <FilterSelect
          label="分区"
          value={category.zone}
          options={DISCOVER_ZONES}
          onChange={value => {
            setKeyword('')
            updateCategory({
              zone: value as JavDBZone, tagIds: [], main: '', year: '', month: ''
            })
          }}
        />
        {taxonomy.isPending ? (
          <Skeleton className="h-9 w-48 rounded-full" />
        ) : taxonomy.isError ? (
          <InlineError onRetry={() => void taxonomy.refetch()} retrying={taxonomy.isFetching}>
            分类加载失败
          </InlineError>
        ) : (
          <>
            <FilterSelect
              label="年份"
              value={category.year || 'all'}
              options={yearOptions}
              onChange={value => updateCategory({ year: value === 'all' ? '' : value, month: '' })}
            />
            <FilterSelect
              label="月份"
              value={category.month || 'all'}
              disabled={!category.year}
              options={[
                { value: 'all', label: '全部月份' },
                ...Array.from({ length: 12 }, (_, i) => ({
                  value: String(i + 1), label: String(i + 1) + ' 月'
                }))
              ]}
              onChange={value => updateCategory({ month: value === 'all' ? '' : value })}
            />
            <Popover>
              <PopoverTrigger asChild>
                <Button variant="outline">
                  选择标签{category.tagIds.length ? '（' + category.tagIds.length + '）' : ''}
                </Button>
              </PopoverTrigger>
              <PopoverContent align="start" className="w-80 max-w-[calc(100vw-2rem)] space-y-3">
                <Input
                  placeholder="搜索标签"
                  value={keyword}
                  onChange={event => setKeyword(event.target.value)}
                />
                <div className="max-h-80 space-y-4 overflow-y-auto p-1">
                  {filtered.length === 0 ? (
                    <p className="text-sm text-muted-foreground">没有匹配的标签</p>
                  ) : filtered.map(group => (
                    <div key={group.id} className="space-y-2">
                      <p className="text-xs font-medium text-muted-foreground">{group.name}</p>
                      <div className="flex flex-wrap gap-3">
                        {group.tags.map(tag => (
                          <label key={tag.id} className="flex cursor-pointer items-center gap-2 text-sm">
                            <Checkbox
                              checked={category.tagIds.includes(tag.id)}
                              onCheckedChange={() => toggleTag(tag.id)}
                            />
                            {tag.name}
                          </label>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              </PopoverContent>
            </Popover>
          </>
        )}
        <Button
          variant="ghost"
          onClick={() => updateCategory({ tagIds: [], year: '', month: '', main: '' })}
          disabled={!category.tagIds.length && !category.year && !category.main}
        >
          清空条件
        </Button>
      </div>
      {mainOptions.length > 0 ? (
        <CommonFilterSelect
          options={mainOptions}
          value={category.main}
          onValueChange={main => updateCategory({ main })}
        />
      ) : null}
      {category.tagIds.length > 0 ? (
        <div className="flex flex-wrap gap-2">
          {category.tagIds.map(id => (
            <Button
              key={id}
              size="sm"
              variant="secondary"
              title={'移除标签：' + (tagNames.get(id) ?? id)}
              onClick={() => toggleTag(id)}
            >
              {tagNames.get(id) ?? id}<XIcon className="size-3" />
            </Button>
          ))}
        </div>
      ) : null}
    </div>
  )
}
