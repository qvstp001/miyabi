import type { DiscoverMovie } from '@/api/discover'
import { MovieCard } from '@/components/movie'
import { Badge } from '@/components/ui/badge'
import { MovieResourceBadges, MovieStateBadge } from '@/components/movie/movie-badges'
import { MovieSubscribeButton } from './movie-subscribe-button'
import { useCatalogueDisplay } from '@/stores/catalogue-display'
import { MovieDetailTrigger } from '@/features/movie-detail/detail-trigger'

export function DiscoverMovieCard({ movie }: { movie: DiscoverMovie }) {
  const display = useCatalogueDisplay()
  const stats = [
    display.showDate && movie.release_date,
    display.showRating && movie.rating > 0 && '评分 ' + movie.rating.toFixed(1),
    display.showDuration && movie.duration > 0 && movie.duration + ' 分钟'
  ].filter(Boolean).join(' · ')
  const subscribable = movie.release_status === 'upcoming' && movie.magnets_count === 0
  return (
    <MovieDetailTrigger movie={{ id: movie.id }} className="block rounded-2xl outline-ring">
      <MovieCard
        movie={movie}
        description={stats || undefined}
        state={<MovieStateBadge movie={movie} />}
        coverOverlay={
          subscribable || movie.ranking ? (
            <div className="absolute top-2 right-2 flex flex-col items-end gap-2">
              {movie.ranking ? <Badge variant="frosted">#{movie.ranking}</Badge> : null}
              {subscribable ? <MovieSubscribeButton movie={movie} /> : null}
            </div>
          ) : undefined
        }
      >
        <MovieResourceBadges movie={movie} />
      </MovieCard>
    </MovieDetailTrigger>
  )
}
