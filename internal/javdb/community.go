package javdb

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/ppxb/miyabi/internal/domain"
)

type wirePlaylist struct {
	ID               wireID `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	MoviesCount      int    `json:"movies_count"`
	ViewsCount       int    `json:"views_count"`
	CollectionsCount int    `json:"collections_count"`
}

func (w wirePlaylist) playlist() Playlist {
	return Playlist{ID: string(w.ID), Name: w.Name, Description: w.Description, MoviesCount: w.MoviesCount, ViewsCount: w.ViewsCount, CollectionsCount: w.CollectionsCount}
}

func (c *Client) Playlists(ctx context.Context, o ListOptions) (Page[Playlist], error) {
	if o.Page < 1 || o.Limit < 1 || o.Limit > 100 {
		return Page[Playlist]{}, domain.E(domain.KindInvalid, "片单页码无效", nil)
	}
	params := url.Values{"page": {strconv.Itoa(o.Page)}, "limit": {strconv.Itoa(o.Limit)}}
	path := "/api/v2/search"
	private := false
	switch {
	case o.MovieID != "":
		path = "/api/v1/lists/related"
		params.Set("movie_id", o.MovieID)
	case o.Mine:
		path, private = "/api/v1/lists", true
		params.Set("sort_by", "created")
	default:
		keyword := strings.TrimSpace(o.Query)
		if keyword == "" {
			return Page[Playlist]{}, domain.E(domain.KindInvalid, "请输入片单关键词", nil)
		}
		params.Set("q", keyword)
		params.Set("type", "list")
		private = true // Search shares the account requirement of movie search.
	}
	var data struct {
		Lists       []*wirePlaylist `json:"lists"`
		CurrentPage *int            `json:"current_page"`
	}
	var err error
	if private {
		err = c.getAccountJSON(ctx, path, params, &data)
	} else {
		err = c.getJSONWithAuth(ctx, path, params, &data)
	}
	if err != nil {
		return Page[Playlist]{}, err
	}
	result := Page[Playlist]{Items: make([]Playlist, 0, len(data.Lists))}
	// Typed search without pagination metadata behaves as a lookup, like actor search.
	result.HasMore = len(data.Lists) >= o.Limit && (o.Mine || o.MovieID != "" || data.CurrentPage != nil)
	for _, item := range data.Lists {
		if item != nil && item.ID != "" {
			result.Items = append(result.Items, item.playlist())
		}
	}
	return result, nil
}

func (c *Client) Playlist(ctx context.Context, id string) (Playlist, error) {
	if strings.TrimSpace(id) == "" {
		return Playlist{}, domain.E(domain.KindInvalid, "片单 ID 不能为空", nil)
	}
	var data struct {
		List *wirePlaylist `json:"list"`
	}
	if err := c.getJSONWithAuth(ctx, "/api/v1/lists/"+url.PathEscape(id), nil, &data); err != nil {
		return Playlist{}, err
	}
	if data.List == nil || data.List.ID == "" {
		return Playlist{}, domain.E(domain.KindUpstream, "JavDB 未返回有效的片单信息", nil)
	}
	return data.List.playlist(), nil
}

func (c *Client) Reviews(ctx context.Context, id string, page, limit int) (Page[Review], error) {
	if strings.TrimSpace(id) == "" || page < 1 || limit < 1 || limit > 100 {
		return Page[Review]{}, domain.E(domain.KindInvalid, "评论参数无效", nil)
	}
	var data struct {
		Reviews []*struct {
			ID         wireID     `json:"id"`
			Username   string     `json:"username"`
			Content    string     `json:"content"`
			Score      wireRating `json:"score"`
			LikesCount int        `json:"likes_count"`
			CreatedAt  string     `json:"created_at"`
		} `json:"reviews"`
	}
	params := url.Values{"page": {strconv.Itoa(page)}, "limit": {strconv.Itoa(limit)}}
	if err := c.getJSONWithAuth(ctx, "/api/v1/movies/"+url.PathEscape(id)+"/reviews", params, &data); err != nil {
		return Page[Review]{}, err
	}
	result := Page[Review]{Items: make([]Review, 0, len(data.Reviews)), HasMore: len(data.Reviews) >= limit}
	for _, item := range data.Reviews {
		if item != nil && item.ID != "" {
			result.Items = append(result.Items, Review{ID: string(item.ID), Username: item.Username, Content: item.Content, Score: float64(item.Score), LikesCount: item.LikesCount, CreatedAt: item.CreatedAt})
		}
	}
	return result, nil
}
