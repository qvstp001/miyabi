import type { ReactNode } from 'react'
import { cn } from 'cn'

export function MovieGridLayout({ children, density = 'comfortable' }: {
  children: ReactNode
  density?: 'comfortable' | 'compact'
}) {
  return (
    <div className={cn(
      'grid gap-3 sm:gap-4',
      density === 'compact'
        ? 'grid-cols-2 lg:grid-cols-4 xl:grid-cols-5'
        : 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 lg:gap-6 xl:grid-cols-4'
    )}>
      {children}
    </div>
  )
}
