import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { apiGet, apiPost, apiPut } from '@/api/client'
import { exploreKeys } from './explore'

export type JavDBConfig = {
  username: string
  password: string
}

export type JavDBLoginInfo = {
  user_id?: string
  username?: string
}

export const javdbKeys = {
  config: ['settings', 'javdb'] as const
}

// Search failures before login are cached per keyword; refresh them once the
// account becomes usable so the page stops showing the login prompt.
const searchQueryKey = ['discover', 'search'] as const

export function useJavDBConfig() {
  return useQuery({
    queryKey: javdbKeys.config,
    queryFn: ({ signal }) => apiGet<JavDBConfig>('/api/settings/javdb', undefined, signal),
    staleTime: 15_000
  })
}

export function useUpdateJavDBConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    onMutate: () => queryClient.cancelQueries({ queryKey: exploreKeys.all }),
    mutationFn: (config: JavDBConfig) => apiPut<JavDBConfig>('/api/settings/javdb', config),
    onSuccess: async next => {
      queryClient.setQueryData(javdbKeys.config, next)
      await queryClient.cancelQueries({ queryKey: exploreKeys.all })
      await queryClient.resetQueries({ queryKey: exploreKeys.all })
      void queryClient.invalidateQueries({ queryKey: searchQueryKey })
    }
  })
}

export function useTestJavDBLogin() {
  const queryClient = useQueryClient()
  return useMutation({
    onMutate: () => queryClient.cancelQueries({ queryKey: exploreKeys.all }),
    mutationFn: (config: JavDBConfig) =>
      apiPost<JavDBLoginInfo>('/api/settings/javdb/test', config),
    onSuccess: async () => {
      await queryClient.cancelQueries({ queryKey: exploreKeys.all })
      await queryClient.resetQueries({ queryKey: exploreKeys.all })
      void queryClient.invalidateQueries({ queryKey: searchQueryKey })
    }
  })
}
