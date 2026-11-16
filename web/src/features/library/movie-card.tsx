import type { LibraryMovie } from '@/api/library'
import { MovieCard } from '@/components/movie'
import { MovieDetailTrigger } from '@/features/movie-detail/detail-trigger'
import { LibraryMovieStatus } from './movie-status'

export function LibraryMovieCard({ movie }: { movie: LibraryMovie }) {
  return (
    <div className="relative h-full min-w-0">
      <MovieDetailTrigger
        movie={{ libraryId: movie.id }}
        className="block h-full min-w-0 rounded-2xl outline-ring"
      >
        <MovieCard movie={movie} titleTooltip={false}>
          <div className="flex min-h-7 items-center">
            <LibraryMovieStatus movie={movie} />
          </div>
        </MovieCard>
      </MovieDetailTrigger>
    </div>
  )
}
