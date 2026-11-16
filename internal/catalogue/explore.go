package catalogue

import (
	"context"

	"github.com/ppxb/miyabi/internal/domain"
	"github.com/ppxb/miyabi/internal/javdb"
)

// Explorer is separate from the scraper-facing catalogue client contract.
type Explorer interface {
	Feed(context.Context, javdb.FeedOptions) (javdb.Page[javdb.FeedMovie], error)
	SearchActors(context.Context, string) ([]javdb.ActorSummary, error)
	Playlists(context.Context, javdb.ListOptions) (javdb.Page[javdb.Playlist], error)
	Playlist(context.Context, string) (javdb.Playlist, error)
	Reviews(context.Context, string, int, int) (javdb.Page[javdb.Review], error)
}

type ExploreMovie struct {
	Movie
	Ranking int `json:"ranking,omitempty"`
}

func (service *Service) explorer() (Explorer, error) {
	client, ok := service.javdb.(Explorer)
	if !ok {
		return nil, domain.E(domain.KindUpstream, "JavDB 内容服务不可用", nil)
	}
	return client, nil
}

// Feeds are deliberately not stored in shared server caches: some are account-private.
func (service *Service) ExploreFeed(ctx context.Context, options javdb.FeedOptions) (javdb.Page[ExploreMovie], error) {
	service.accountMu.RLock()
	defer service.accountMu.RUnlock()
	client, err := service.explorer()
	if err != nil {
		return javdb.Page[ExploreMovie]{}, err
	}
	feed, err := client.Feed(ctx, options)
	if err != nil {
		return javdb.Page[ExploreMovie]{}, err
	}
	movies := make([]domain.Movie, len(feed.Items))
	for i, item := range feed.Items {
		movies[i] = item.Movie
	}
	projected := projectMovies(ctx, movies)
	result := javdb.Page[ExploreMovie]{Items: make([]ExploreMovie, len(projected)), HasMore: feed.HasMore, GeneratedAt: feed.GeneratedAt}
	for i, movie := range projected {
		result.Items[i] = ExploreMovie{Movie: movie, Ranking: feed.Items[i].Ranking}
	}
	return result, nil
}

func (service *Service) ExploreActors(ctx context.Context, keyword string) ([]javdb.ActorSummary, error) {
	service.accountMu.RLock()
	defer service.accountMu.RUnlock()
	client, err := service.explorer()
	if err != nil {
		return nil, err
	}
	return client.SearchActors(ctx, keyword)
}

func (service *Service) ExploreLists(ctx context.Context, options javdb.ListOptions) (javdb.Page[javdb.Playlist], error) {
	service.accountMu.RLock()
	defer service.accountMu.RUnlock()
	client, err := service.explorer()
	if err != nil {
		return javdb.Page[javdb.Playlist]{}, err
	}
	return client.Playlists(ctx, options)
}

func (service *Service) ExploreList(ctx context.Context, id string) (javdb.Playlist, error) {
	service.accountMu.RLock()
	defer service.accountMu.RUnlock()
	client, err := service.explorer()
	if err != nil {
		return javdb.Playlist{}, err
	}
	return client.Playlist(ctx, id)
}

func (service *Service) ExploreReviews(ctx context.Context, id string, page, limit int) (javdb.Page[javdb.Review], error) {
	service.accountMu.RLock()
	defer service.accountMu.RUnlock()
	client, err := service.explorer()
	if err != nil {
		return javdb.Page[javdb.Review]{}, err
	}
	return client.Reviews(ctx, id, page, limit)
}
