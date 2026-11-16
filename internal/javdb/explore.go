package javdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ppxb/miyabi/internal/domain"
)

// Page preserves pagination independently of the number of valid decoded items.
type Page[T any] struct {
	Items       []T    `json:"items"`
	HasMore     bool   `json:"has_more"`
	GeneratedAt string `json:"generated_at,omitempty"`
}

type FeedOptions struct {
	Kind   string
	Zone   domain.Zone
	Period string
	ID     string
	Status string
	Sort   string
	Order  string
	Page   int
	Limit  int
}

type FeedMovie struct {
	domain.Movie
	Ranking int `json:"ranking,omitempty"`
}

type ActorSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	NameZHT     string `json:"name_zht"`
	Avatar      string `json:"avatar"`
	VideosCount int    `json:"videos_count"`
}

type Playlist struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	MoviesCount      int    `json:"movies_count"`
	ViewsCount       int    `json:"views_count"`
	CollectionsCount int    `json:"collections_count"`
}

type Review struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
	LikesCount int     `json:"likes_count"`
	CreatedAt  string  `json:"created_at"`
}

type ListOptions struct {
	Query   string
	MovieID string
	Mine    bool
	Page    int
	Limit   int
}

// Feed reads one page; ranking endpoints return a finite snapshot and are sliced locally.
func (c *Client) Feed(ctx context.Context, options FeedOptions) (Page[FeedMovie], error) {
	path, params, err := buildFeedParams(options)
	if err != nil {
		return Page[FeedMovie]{}, domain.E(domain.KindInvalid, err.Error(), err)
	}
	if options.Kind == "top250" && (options.Page-1)*options.Limit >= 250 {
		return Page[FeedMovie]{Items: []FeedMovie{}}, nil
	}
	var data struct {
		Movies []struct {
			wireMovie
			Ranking int `json:"ranking"`
		} `json:"movies"`
		GeneratedAt string `json:"generated_at"`
		CurrentPage *int   `json:"current_page"`
	}
	if options.Kind == "top250" || options.Kind == "personal" {
		err = c.getAccountJSON(ctx, path, params, &data)
	} else {
		err = c.getJSONWithAuth(ctx, path, params, &data)
	}
	if err != nil {
		return Page[FeedMovie]{}, err
	}
	if options.Kind != "ranking" && data.CurrentPage != nil && *data.CurrentPage != options.Page {
		return Page[FeedMovie]{}, domain.E(domain.KindUpstream, "JavDB 未返回请求的页码，请返回上一页重试", nil)
	}
	result := Page[FeedMovie]{Items: make([]FeedMovie, 0, len(data.Movies)), GeneratedAt: data.GeneratedAt}
	rows := data.Movies
	if options.Kind == "ranking" {
		start := min((options.Page-1)*options.Limit, len(rows))
		end := min(start+options.Limit, len(rows))
		result.HasMore = end < len(rows)
		rows = rows[start:end]
	} else {
		result.HasMore = len(rows) >= options.Limit
		if options.Kind == "top250" {
			rows = rows[:min(len(rows), 250-(options.Page-1)*options.Limit)]
			result.HasMore = result.HasMore && options.Page*options.Limit < 250
		}
	}
	for index, row := range rows {
		movie, err := movieFromWire(ctx, row.wireMovie)
		if err != nil {
			return Page[FeedMovie]{}, domain.E(domain.KindUpstream, "JavDB 影片数据无效", err)
		}
		if options.Kind == "ranking" && row.Ranking == 0 {
			row.Ranking = (options.Page-1)*options.Limit + index + 1
		}
		result.Items = append(result.Items, FeedMovie{Movie: movie, Ranking: row.Ranking})
	}
	return result, nil
}

func buildFeedParams(o FeedOptions) (string, url.Values, error) {
	if o.Page < 1 || o.Limit < 1 || o.Limit > 100 {
		return "", nil, errors.New("页码或每页数量无效")
	}
	p := url.Values{"page": {strconv.Itoa(o.Page)}, "limit": {strconv.Itoa(o.Limit)}}
	switch o.Kind {
	case "ranking":
		zone, ok := zoneCodes[o.Zone]
		if !ok || o.Zone == domain.ZoneAnime {
			return "", nil, errors.New("排行榜分区无效")
		}
		if o.Period != "daily" && o.Period != "weekly" && o.Period != "monthly" {
			return "", nil, errors.New("排行榜周期无效")
		}
		// Ranking snapshots do not honor page/limit; do not pretend they are paginated.
		return "/api/v1/rankings", url.Values{"type": {strconv.Itoa(zone)}, "period": {o.Period}}, nil
	case "top250":
		p.Set("type", "all")
		p.Set("type_value", "all")
		p.Set("start_rank", "1")
		p.Set("ignore_watched", "false")
		return "/api/v1/movies/top", p, nil
	case "list":
		if strings.TrimSpace(o.ID) == "" || strings.ContainsAny(o.ID, ":,") {
			return "", nil, errors.New("片单 ID 无效")
		}
		p.Set("filter_by", "0:l:"+o.ID+":")
		p.Set("sort_by", o.Sort)
		p.Set("order_by", o.Order)
		return "/api/v1/movies/tags", p, nil
	case "personal":
		if o.Status != "want_watch" && o.Status != "watched" {
			return "", nil, errors.New("个人列表类型无效")
		}
		p.Set("status", o.Status)
		p.Set("sort_by", "create")
		p.Set("order_by", "desc")
		return "/api/v2/users/review_movies", p, nil
	default:
		return "", nil, errors.New("内容类型无效")
	}
}

// getAccountJSON never issues an anonymous request for account-private resources.
func (c *Client) getAccountJSON(ctx context.Context, path string, params url.Values, out any) error {
	username, password := c.Credentials()
	if username == "" || password == "" {
		return ErrLoginRequired
	}
	err := c.getJSONWithAuth(ctx, path, params, out)
	if requiresWebFallback(err) {
		return ErrLoginRequired
	}
	return err
}

// SearchActors is an unpaginated typed lookup; the upstream can ignore page/limit.
func (c *Client) SearchActors(ctx context.Context, keyword string) ([]ActorSummary, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, domain.E(domain.KindInvalid, "请输入演员名称", nil)
	}
	var data struct {
		Actors []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			NameZHT     string `json:"name_zht"`
			Avatar      string `json:"avatar_url"`
			VideosCount int    `json:"videos_count"`
		} `json:"actors"`
	}
	if err := c.getAccountJSON(ctx, "/api/v2/search", url.Values{"q": {keyword}, "type": {"actor"}, "page": {"1"}}, &data); err != nil {
		return nil, err
	}
	actors := make([]ActorSummary, 0, len(data.Actors))
	for _, actor := range data.Actors {
		if actor.ID != "" {
			actors = append(actors, ActorSummary{ID: actor.ID, Name: actor.Name, NameZHT: actor.NameZHT, Avatar: actor.Avatar, VideosCount: actor.VideosCount})
		}
	}
	return actors, nil
}

// wireID preserves numeric review and list IDs without float conversion.
type wireID string

func (id *wireID) UnmarshalJSON(raw []byte) error {
	if string(raw) == "null" {
		return nil
	}
	if len(raw) > 0 && raw[0] == '"' {
		return json.Unmarshal(raw, (*string)(id))
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err != nil {
		return fmt.Errorf("decode JavDB ID: %w", err)
	}
	*id = wireID(number.String())
	return nil
}
