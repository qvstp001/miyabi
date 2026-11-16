package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/ppxb/miyabi/internal/catalogue"
	"github.com/ppxb/miyabi/internal/domain"
)

type discoverFiltersStub struct {
	CatalogueManager
	called bool
	search domain.SearchOptions
	browse domain.BrowseOptions
}

func (s *discoverFiltersStub) Search(_ context.Context, _ string, options domain.SearchOptions) ([]catalogue.Movie, error) {
	s.called = true
	s.search = options
	return []catalogue.Movie{}, nil
}

func (s *discoverFiltersStub) Browse(_ context.Context, options domain.BrowseOptions) ([]catalogue.Movie, error) {
	s.called = true
	s.browse = options
	return []catalogue.Movie{}, nil
}

func TestDiscoverSearchFilters(t *testing.T) {
	for _, tc := range []struct {
		query  string
		status int
		want   domain.SearchOptions
	}{
		{"?q=test&zone=fc2&sort=score&filter=subtitle&page=2&limit=40", http.StatusOK, domain.SearchOptions{Zone: domain.ZoneFC2, Sort: "score", FilterBy: "subtitle", Page: 2, Limit: 40}},
		{"?q=test", http.StatusOK, domain.SearchOptions{Page: 1, Limit: 20}},
		{"?q=test&zone=invalid", http.StatusBadRequest, domain.SearchOptions{}},
		{"?q=test&sort=invalid", http.StatusBadRequest, domain.SearchOptions{}},
		{"?q=test&filter=invalid", http.StatusBadRequest, domain.SearchOptions{}},
	} {
		t.Run(tc.query, func(t *testing.T) {
			stub := &discoverFiltersStub{}
			router := NewRouter(Dependencies{Access: NewAccessGateService("", ""), Catalogue: stub, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/discover/search"+tc.query, nil))
			if response.Code != tc.status || stub.called != (tc.status == http.StatusOK) || !reflect.DeepEqual(stub.search, tc.want) {
				t.Fatalf("status=%d called=%t options=%+v body=%s", response.Code, stub.called, stub.search, response.Body)
			}
		})
	}
}

func TestDiscoverCombinedBrowseFilters(t *testing.T) {
	stub := &discoverFiltersStub{}
	router := NewRouter(Dependencies{Access: NewAccessGateService("", ""), Catalogue: stub, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	query := "?zone=fc2&tag_id=one&tag_id=two&main=m&main=s&year=2026&month=9&sort=score&order=asc&limit=60"
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/discover/movies"+query, nil))
	want := domain.BrowseOptions{Zone: domain.ZoneFC2, TagIDs: []string{"one", "two"}, Main: []string{"m", "s"}, Year: "2026", Month: "9", Sort: "score", Order: "asc", Page: 1, Limit: 60}
	if response.Code != http.StatusOK || !reflect.DeepEqual(stub.browse, want) {
		t.Fatalf("status=%d options=%+v body=%s", response.Code, stub.browse, response.Body)
	}
	stub.called = false
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/discover/movies?month=9", nil))
	if response.Code != http.StatusBadRequest || stub.called {
		t.Fatalf("month without year: status=%d called=%t", response.Code, stub.called)
	}
}
