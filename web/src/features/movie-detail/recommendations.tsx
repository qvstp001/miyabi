import type { MovieReference } from '@/api/discover'
import { MovieCard, MovieGridLayout } from '@/components/movie'
import { MovieDetailTrigger } from './detail-trigger'

// Scrolling recommendations never fetches their catalogue details.
export function MovieRecommendations({ title, movies }: { title: string; movies: MovieReference[] }) {
  return <section className="space-y-4">
    <h2 className="text-xl font-semibold">{title}</h2>
    {movies.length === 0 ? <p className="text-sm text-muted-foreground">暂无相关影片</p> : <MovieGridLayout>
      {movies.slice(0, 8).map(movie => <MovieDetailTrigger key={movie.id} movie={{ id: movie.id }} className="block rounded-2xl outline-ring">
        <MovieCard movie={{ code: movie.code, title: '', cover: movie.thumbnail }} />
      </MovieDetailTrigger>)}
    </MovieGridLayout>}
  </section>
}