import type { LibraryMovie } from '@/api/library'
import { Badge } from '@/components/ui/badge'

export function LibraryMovieStatus({ movie }: { movie: LibraryMovie }) {
  return (
    <>
      {movie.scrape_status === 'done' ? (
        <Badge variant="success">已入库</Badge>
      ) : (
        <Badge variant="outline">已索引</Badge>
      )}
    </>
  )
}
