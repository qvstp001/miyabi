import type { NamedEntity } from '@/api/discover'
import { Checkbox } from '@/components/ui/checkbox'
import { mainValues } from './filter-options'

export function CommonFilterSelect({ options, value, onValueChange }: {
  options: NamedEntity[]
  value: string
  onValueChange: (value: string) => void
}) {
  const selected = mainValues(value)
  return (
    <div className="flex flex-wrap gap-x-4 gap-y-3">
      {options.map(option => (
        <label key={option.id} className="flex cursor-pointer items-center gap-2 text-sm">
          <Checkbox
            checked={selected.includes(option.id)}
            onCheckedChange={checked =>
              onValueChange(
                (checked
                  ? [...selected, option.id]
                  : selected.filter(id => id !== option.id)
                ).sort().join(',')
              )
            }
          />
          {option.name}
        </label>
      ))}
    </div>
  )
}
