import { useQuery } from '@tanstack/react-query'
import { apiGet } from './client'
import type { DiscoverMovie, JavDBZone } from './discover'
import type { BrowseSorting } from '@/features/discover/filter-options'

export type ContentPage<T> = { items: T[]; has_more: boolean; generated_at?: string }
export type ExploreMovie = DiscoverMovie & { ranking?: number }
export type ActorSummary = {
  id: string
  name: string
  name_zht: string
  avatar: string
  videos_count: number
}
export type Playlist = {
  id: string
  name: string
  description: string
  movies_count: number
  views_count: number
  collections_count: number
}
export type MovieReview = {
  id: string
  username: string
  content: string
  score: number
  likes_count: number
  created_at: string
}
export type FeedParams = BrowseSorting & {
  kind: 'ranking' | 'top250' | 'list' | 'personal'
  zone?: Exclude<JavDBZone, 'anime'>
  period?: 'daily' | 'weekly' | 'monthly'
  id?: string
  status?: 'want_watch' | 'watched'
  page: number
  limit: number
}
export type ListParams = {
  mode: 'search' | 'mine' | 'related'
  q?: string
  movie_id?: string
  page: number
  limit: number
}

export const exploreKeys = {
  all: ['discover', 'explore'] as const,
  feed: (params: FeedParams) => ['discover', 'explore', 'feed', params] as const,
  actors: (q: string) => ['discover', 'explore', 'actors', q] as const,
  lists: (params: ListParams) => ['discover', 'explore', 'lists', params] as const,
  list: (id: string) => ['discover', 'explore', 'list', id] as const,
  reviews: (id: string, page: number) => ['discover', 'explore', 'reviews', id, page] as const
}

export function useExploreFeed(params: FeedParams, enabled = true) {
  return useQuery({
    queryKey: exploreKeys.feed(params),
    queryFn: ({ signal }) => apiGet<ContentPage<ExploreMovie>>('/api/discover/feed', params, signal),
    enabled,
    staleTime: params.kind === 'personal' ? 0 : 60_000
  })
}

export function useActorSearch(keyword: string) {
  const q = keyword.trim()
  return useQuery({
    queryKey: exploreKeys.actors(q),
    queryFn: ({ signal }) => apiGet<ActorSummary[]>('/api/discover/actors', { q }, signal),
    enabled: q.length > 0,
    staleTime: 0
  })
}

export function usePlaylists(params: ListParams, enabled = true) {
  return useQuery({
    queryKey: exploreKeys.lists(params),
    queryFn: ({ signal }) => apiGet<ContentPage<Playlist>>('/api/discover/lists', params, signal),
    enabled,
    staleTime: 0
  })
}

export function usePlaylist(id: string) {
  return useQuery({
    queryKey: exploreKeys.list(id),
    queryFn: ({ signal }) => apiGet<Playlist>('/api/discover/lists/' + encodeURIComponent(id), undefined, signal),
    enabled: !!id,
    staleTime: 0
  })
}

export function useMovieReviews(id: string, page: number) {
  return useQuery({
    queryKey: exploreKeys.reviews(id, page),
    queryFn: ({ signal }) => apiGet<ContentPage<MovieReview>>(
      '/api/discover/movies/' + encodeURIComponent(id) + '/reviews', { page, limit: 20 }, signal
    ),
    staleTime: 60_000
  })
}
