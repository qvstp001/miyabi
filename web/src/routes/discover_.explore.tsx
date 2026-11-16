import { createFileRoute } from '@tanstack/react-router'
import { AppPage } from '@/components/app-page'
import { ErrorState } from '@/components/error-state'
import { ExplorePage } from '@/features/explore/page'
import { validateExploreSearch } from '@/features/explore/search'

export const Route = createFileRoute('/discover_/explore')({
  validateSearch: validateExploreSearch,
  component: ExplorePage,
  errorComponent: () => (
    <AppPage>
      <ErrorState message="JavDB 内容筛选条件无效" />
    </AppPage>
  )
})
