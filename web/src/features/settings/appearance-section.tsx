import { MonitorCogIcon, MonitorIcon, MoonIcon, SunIcon } from 'lucide-react'
import { useTheme } from 'next-themes'

import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { CatalogueDisplayOptions } from '@/features/discover/display-settings'
import { SettingRow, SettingsSection } from './shared'

const THEME_OPTIONS = [
  { value: 'system', icon: MonitorIcon },
  { value: 'light', icon: SunIcon },
  { value: 'dark', icon: MoonIcon }
]

export function AppearanceSection() {
  const { theme, setTheme } = useTheme()

  return (
    <SettingsSection icon={<MonitorCogIcon className="size-4" />} title="外观">
      <SettingRow title="主题" description="控制应用的明暗色主题" inline>
        <Tabs value={theme} onValueChange={setTheme}>
          <TabsList>
            {THEME_OPTIONS.map(option => (
              <TabsTrigger key={option.value} value={option.value}>
                <option.icon className="size-4" />
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </SettingRow>
      <SettingRow title="JavDB 展示" description="应用于发现、搜索及关联作品列表，保存在当前浏览器">
        <div className="w-64 max-w-full">
          <CatalogueDisplayOptions />
        </div>
      </SettingRow>
    </SettingsSection>
  )
}
