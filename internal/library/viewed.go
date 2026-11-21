package library

import (
	"context"

	"github.com/ppxb/miyabi/internal/browsehistory"
)

const maxViewedMovies = 5000

func (s *Service) ViewedMovieIDs(ctx context.Context) ([]string, error) {
	return browsehistory.New(s.database).ViewedMovieIDs(ctx)
}

func (s *Service) AddViewedMovieIDs(ctx context.Context, ids []string) error {
	return browsehistory.New(s.database).AddViewedMovieIDs(ctx, ids)
}
