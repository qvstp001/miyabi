import type { DiscoverMovie } from '@/api/discover'
import { MovieGridLayout } from '@/components/movie'
import { DiscoverMovieCard } from './movie-card'
import { useCatalogueDisplay } from '@/stores/catalogue-display'

export function MovieGrid({ movies }: { movies: DiscoverMovie[] }) {
  const density = useCatalogueDisplay(state => state.density)
  return (
    <MovieGridLayout density={density}>
      {movies.map(movie => (
        <DiscoverMovieCard key={movie.id} movie={movie} />
      ))}
    </MovieGridLayout>
  )
}
