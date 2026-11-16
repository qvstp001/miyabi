import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'

export function FilterSelect({
  label,
  value,
  options,
  onChange,
  disabled = false
}: {
  label: string
  value: string
  options: readonly { value: string; label: string }[]
  onChange: (value: string) => void
  disabled?: boolean
}) {
  return (
    <label className="flex min-w-0 flex-col gap-1.5 text-xs text-muted-foreground">
      {label}
      <Select value={value} onValueChange={onChange} disabled={disabled}>
        <SelectTrigger className="w-full min-w-28 text-foreground">
          <SelectValue />
        </SelectTrigger>
        <SelectContent position="popper" align="start">
          <SelectGroup>
            {options.map(option => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </label>
  )
}
