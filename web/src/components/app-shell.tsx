import type { PropsWithChildren } from 'react'
import { CompassIcon, DownloadIcon, SearchIcon, SettingsIcon } from 'lucide-react'

import { FloatingNav, type FloatingNavItem } from '@/components/floating-nav'
import { Toaster } from '@/components/ui/sonner'

const NAV_ITEMS: FloatingNavItem[] = [
  { id: 'discover', label: '发现', icon: CompassIcon, to: '/discover' },
  { id: 'search', label: '搜索', icon: SearchIcon, to: '/search' },
  { id: 'downloads', label: '下载', icon: DownloadIcon, to: '/downloads' },
  { id: 'settings', label: '设置', icon: SettingsIcon, to: '/settings' }
]

export function AppShell({ children }: PropsWithChildren) {
  return (
    <div className="relative min-h-dvh">
      <FloatingNav items={NAV_ITEMS} />
      {children}
      <Toaster position="top-right" closeButton duration={6000} />
    </div>
  )
}
