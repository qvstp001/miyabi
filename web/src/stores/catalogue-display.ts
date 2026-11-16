import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { DISCOVER_PAGE_SIZE } from '@/features/discover/constants'

export type CatalogueDisplay = {
  pageSize: 20 | 40 | 60
  density: 'comfortable' | 'compact'
  showRating: boolean
  showDuration: boolean
  showDate: boolean
}

export const DEFAULT_CATALOGUE_DISPLAY: CatalogueDisplay = {
  pageSize: DISCOVER_PAGE_SIZE,
  density: 'comfortable',
  showRating: true,
  showDuration: false,
  showDate: true
}

type DisplayState = CatalogueDisplay & { update: (values: Partial<CatalogueDisplay>) => void }

export const useCatalogueDisplay = create<DisplayState>()(
  persist(
    set => ({ ...DEFAULT_CATALOGUE_DISPLAY, update: values => set(values) }),
    { name: 'miyabi-catalogue-display' }
  )
)
