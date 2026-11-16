import { SlidersHorizontalIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Switch } from '@/components/ui/switch'
import {
  useCatalogueDisplay,
  DEFAULT_CATALOGUE_DISPLAY,
  type CatalogueDisplay
} from '@/stores/catalogue-display'
import { FilterSelect } from './filter-select'

export function CatalogueDisplaySettings({ onPageSizeChange }: { onPageSizeChange?: () => void }) {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline"><SlidersHorizontalIcon className="size-4" />展示设置</Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="space-y-4">
        <CatalogueDisplayOptions onPageSizeChange={onPageSizeChange} />
      </PopoverContent>
    </Popover>
  )
}

export function CatalogueDisplayOptions({ onPageSizeChange }: { onPageSizeChange?: () => void }) {
  const settings = useCatalogueDisplay()
  return (
    <div className="space-y-4">
      <FilterSelect
        label="每页数量"
        value={String(settings.pageSize)}
        options={[20, 40, 60].map(size => ({ value: String(size), label: size + ' 部' }))}
        onChange={value => {
          settings.update({ pageSize: Number(value) as CatalogueDisplay['pageSize'] })
          onPageSizeChange?.()
        }}
      />
      <FilterSelect
        label="卡片密度"
        value={settings.density}
        options={[{ value: 'comfortable', label: '舒适' }, { value: 'compact', label: '紧凑' }]}
        onChange={value => settings.update({ density: value as CatalogueDisplay['density'] })}
      />
      {([
        { key: 'showRating', label: '显示评分' },
        { key: 'showDuration', label: '显示片长' },
        { key: 'showDate', label: '显示发行日期' }
      ] as const).map(option => (
        <label key={option.key} className="flex items-center justify-between gap-4 text-sm">
          {option.label}
          <Switch
            checked={settings[option.key]}
            onCheckedChange={value => settings.update({ [option.key]: value })}
          />
        </label>
      ))}
      <Button
        variant="ghost"
        size="sm"
        onClick={() => {
          settings.update(DEFAULT_CATALOGUE_DISPLAY)
          if (settings.pageSize !== DEFAULT_CATALOGUE_DISPLAY.pageSize) onPageSizeChange?.()
        }}
      >
        恢复默认展示
      </Button>
    </div>
  )
}
