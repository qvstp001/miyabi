import { createFileRoute } from '@tanstack/react-router'

import { AppPage } from '@/components/app-page'
import { PageHeader } from '@/components/page-header'
import { Card, CardContent } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { AppearanceSection } from '@/features/settings/appearance-section'
import { DownloadSection } from '@/features/settings/download-section'
import { JavDBSection } from '@/features/settings/javdb-section'
import { NetworkSection } from '@/features/settings/network-section'
import { PanSection } from '@/features/settings/pan-section'
import { PrivacySection } from '@/features/settings/privacy-section'

export const Route = createFileRoute('/settings')({
  component: SettingsPage
})

function SettingsPage() {
  return (
    <AppPage showBackTop={false}>
      <PageHeader title="设置" description="管理应用选项和其他配置" />

      <Card>
        <CardContent className="space-y-8">
          <AppearanceSection />
          <Separator />
          <PrivacySection />
          <Separator />
          <NetworkSection />
          <Separator />
          <JavDBSection />
          <Separator />
          <PanSection />
          <Separator />
          <DownloadSection />
        </CardContent>
      </Card>
    </AppPage>
  )
}
