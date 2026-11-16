package javdb

import (
	"errors"
	"testing"

	"github.com/ppxb/miyabi/internal/domain"
)

func TestRankingSnapshotUsesLocalPagination(t *testing.T) {
	transport := &fixtureTransport{responses: map[string][]byte{
		"/api/v1/rankings": []byte(`{"success":1,"data":{"movies":[{"id":"one","number":"AA-001"},{"id":"two","number":"AA-002"},{"id":"three","number":"AA-003"}]}}`),
	}}
	client := clientWithTransport(transport)
	page, err := client.Feed(t.Context(), FeedOptions{Kind: "ranking", Zone: domain.ZoneFC2, Period: "monthly", Page: 2, Limit: 2})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != "three" || page.Items[0].Ranking != 3 || page.HasMore {
		t.Fatalf("feed=%+v err=%v", page, err)
	}
	params := transport.calls[0].params
	if params.Get("type") != "3" || params.Get("period") != "monthly" || params.Has("page") || params.Has("limit") {
		t.Fatalf("ranking query=%v", params)
	}
}

func TestExplorePrivateResourcesRequireConfiguredAccount(t *testing.T) {
	transport := &fixtureTransport{}
	client := clientWithTransport(transport)
	client.SetCredentials("", "")
	for _, kind := range []string{"top250", "personal"} {
		_, err := client.Feed(t.Context(), FeedOptions{Kind: kind, Status: "watched", Page: 1, Limit: 20})
		if !errors.Is(err, ErrLoginRequired) {
			t.Fatalf("%s error=%v", kind, err)
		}
	}
	if _, err := client.SearchActors(t.Context(), "actor"); !errors.Is(err, ErrLoginRequired) {
		t.Fatalf("actor search error=%v", err)
	}
	if len(transport.calls) != 0 {
		t.Fatal("private endpoints were requested anonymously")
	}
}

func TestExploreFeedRequestsMatchTheirCapabilities(t *testing.T) {
	for _, tc := range []struct {
		options FeedOptions
		path    string
		key     string
		value   string
	}{
		{FeedOptions{Kind: "top250", Page: 2, Limit: 40}, "/api/v1/movies/top", "type", "all"},
		{FeedOptions{Kind: "list", ID: "list-1", Sort: "score", Order: "asc", Page: 2, Limit: 40}, "/api/v1/movies/tags", "filter_by", "0:l:list-1:"},
		{FeedOptions{Kind: "personal", Status: "watched", Page: 2, Limit: 40}, "/api/v2/users/review_movies", "status", "watched"},
	} {
		transport := &fixtureTransport{responses: map[string][]byte{tc.path: []byte(`{"success":1,"data":{"movies":[],"current_page":2}}`)}}
		client := clientWithTransport(transport)
		if _, err := client.Feed(t.Context(), tc.options); err != nil {
			t.Fatal(err)
		}
		params := transport.calls[0].params
		if params.Get(tc.key) != tc.value || params.Get("page") != "2" || params.Get("limit") != "40" {
			t.Fatalf("%s params=%v", tc.path, params)
		}
	}
}

func TestReviewsKeepExactIDsAndSkipNullWithoutTruncatingPagination(t *testing.T) {
	transport := &fixtureTransport{responses: map[string][]byte{
		"/api/v1/movies/movie-1/reviews": []byte(`{"success":1,"data":{"reviews":[{"id":9007199254740993,"username":"Alice","score":"4.5","content":"Plain text"},null,{"id":"two","score":null}]}}`),
	}}
	page, err := clientWithTransport(transport).Reviews(t.Context(), "movie-1", 3, 3)
	if err != nil || len(page.Items) != 2 || page.Items[0].ID != "9007199254740993" || page.Items[0].Score != 4.5 || !page.HasMore {
		t.Fatalf("reviews=%+v err=%v", page, err)
	}
	if len(transport.calls) != 1 || transport.calls[0].params.Get("page") != "3" {
		t.Fatalf("calls=%+v", transport.calls)
	}
}

func TestPlaylistLookupDoesNotInventSearchPages(t *testing.T) {
	transport := &fixtureTransport{responses: map[string][]byte{
		"/api/v2/search": []byte(`{"success":1,"data":{"lists":[{"id":123,"name":"A list"}]}}`),
		"/api/v1/lists/123": []byte(`{"success":1,"data":{"list":{"id":123,"name":"A list","movies_count":7}}}`),
	}}
	client := clientWithTransport(transport)
	page, err := client.Playlists(t.Context(), ListOptions{Query: "keyword", Page: 1, Limit: 1})
	if err != nil || len(page.Items) != 1 || page.HasMore || page.Items[0].ID != "123" {
		t.Fatalf("lists=%+v err=%v", page, err)
	}
	if transport.calls[0].params.Get("type") != "list" {
		t.Fatalf("search params=%v", transport.calls[0].params)
	}
	list, err := client.Playlist(t.Context(), "123")
	if err != nil || list.MoviesCount != 7 {
		t.Fatalf("list=%+v err=%v", list, err)
	}
}

func TestExploreRejectsInvalidMasksAndUnsupportedRankingZones(t *testing.T) {
	for _, options := range []FeedOptions{
		{Kind: "ranking", Zone: domain.ZoneAnime, Period: "weekly", Page: 1, Limit: 20},
		{Kind: "ranking", Zone: domain.ZoneCensored, Period: "yearly", Page: 1, Limit: 20},
		{Kind: "list", ID: "id:m:tag", Page: 1, Limit: 20},
		{Kind: "personal", Status: "invalid", Page: 1, Limit: 20},
	} {
		if _, _, err := buildFeedParams(options); err == nil {
			t.Fatalf("accepted options=%+v", options)
		}
	}
}

func TestTop250StopsBeforeRequestingPastTheLastRank(t *testing.T) {
	transport := &fixtureTransport{}
	page, err := clientWithTransport(transport).Feed(t.Context(), FeedOptions{Kind: "top250", Page: 14, Limit: 20})
	if err != nil || len(page.Items) != 0 || page.HasMore || len(transport.calls) != 0 {
		t.Fatalf("page=%+v calls=%d err=%v", page, len(transport.calls), err)
	}
}
