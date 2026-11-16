package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ppxb/miyabi/internal/catalogue"
	"github.com/ppxb/miyabi/internal/domain"
	"github.com/ppxb/miyabi/internal/javdb"
)

type exploreStub struct {
	CatalogueExplorer
	calls   int
	options javdb.FeedOptions
	err     error
}

func (s *exploreStub) ExploreFeed(_ context.Context, options javdb.FeedOptions) (javdb.Page[catalogue.ExploreMovie], error) {
	s.calls++
	s.options = options
	return javdb.Page[catalogue.ExploreMovie]{Items: []catalogue.ExploreMovie{}}, s.err
}

func TestExploreRoutesValidateFiltersAndPreserveLoginErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		query  string
		status int
		err    error
	}{
		{"ranking", "?kind=ranking&zone=fc2&period=monthly&page=3&limit=40", http.StatusOK, nil},
		{"private login", "?kind=personal&status=watched", http.StatusUnauthorized, javdb.ErrLoginRequired},
		{"missing list ID", "?kind=list", http.StatusBadRequest, nil},
		{"invalid period", "?period=yearly", http.StatusBadRequest, nil},
		{"unsupported zone", "?zone=anime", http.StatusBadRequest, nil},
		{"invalid status", "?kind=personal&status=viewed", http.StatusBadRequest, nil},
		{"invalid page", "?page=0", http.StatusBadRequest, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &exploreStub{err: tc.err}
			router := NewRouter(Dependencies{Access: NewAccessGateService("", ""), Explore: stub, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/discover/feed"+tc.query, nil))
			if response.Code != tc.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body)
			}
			if tc.status == http.StatusBadRequest && stub.calls != 0 {
				t.Fatal("invalid request reached the upstream client")
			}
			if tc.name == "ranking" && (stub.options.Zone != domain.ZoneFC2 || stub.options.Period != "monthly" || stub.options.Page != 3 || stub.options.Limit != 40) {
				t.Fatalf("options=%+v", stub.options)
			}
			if tc.err != nil && !strings.Contains(response.Body.String(), "JAVDB_LOGIN_REQUIRED") {
				t.Fatalf("login error lost its code: %s", response.Body)
			}
			if tc.status == http.StatusOK && !strings.Contains(response.Header().Get("Cache-Control"), "no-store") {
				t.Fatal("account-capable feed response must not be stored by HTTP caches")
			}
		})
	}
}
