package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/ppxb/miyabi/internal/catalogue"
	"github.com/ppxb/miyabi/internal/domain"
	"github.com/ppxb/miyabi/internal/javdb"
)

type CatalogueExplorer interface {
	ExploreFeed(context.Context, javdb.FeedOptions) (javdb.Page[catalogue.ExploreMovie], error)
	ExploreActors(context.Context, string) ([]javdb.ActorSummary, error)
	ExploreLists(context.Context, javdb.ListOptions) (javdb.Page[javdb.Playlist], error)
	ExploreList(context.Context, string) (javdb.Playlist, error)
	ExploreReviews(context.Context, string, int, int) (javdb.Page[javdb.Review], error)
}

type exploreFeedQuery struct {
	Kind   string `form:"kind,default=ranking" binding:"oneof=ranking top250 list personal"`
	Zone   string `form:"zone,default=censored" binding:"oneof=censored uncensored western fc2"`
	Period string `form:"period,default=weekly" binding:"oneof=daily weekly monthly"`
	ID     string `form:"id" binding:"required_if=Kind list"`
	Status string `form:"status,default=want_watch" binding:"oneof=want_watch watched"`
	Sort   string `form:"sort,default=release" binding:"oneof=hit release score update want_watch_count watched_count"`
	Order  string `form:"order,default=desc" binding:"oneof=asc desc"`
	Page   int    `form:"page,default=1" binding:"min=1,max=100000000"`
	Limit  int    `form:"limit,default=20" binding:"min=1,max=100"`
}

type exploreListQuery struct {
	Mode    string `form:"mode,default=search" binding:"oneof=search mine related"`
	Query   string `form:"q" binding:"required_if=Mode search"`
	MovieID string `form:"movie_id" binding:"required_if=Mode related"`
	Page    int    `form:"page,default=1" binding:"min=1,max=100000000"`
	Limit   int    `form:"limit,default=20" binding:"min=1,max=100"`
}

type explorePageQuery struct {
	Page  int `form:"page,default=1" binding:"min=1,max=100000000"`
	Limit int `form:"limit,default=20" binding:"min=1,max=100"`
}

func installExploreRoutes(router *gin.RouterGroup, manager CatalogueExplorer) {
	router.GET("/discover/feed", noStore(), func(c *gin.Context) {
		query, ok := bindQuery[exploreFeedQuery](c)
		if !ok {
			return
		}
		feed, err := manager.ExploreFeed(c.Request.Context(), javdb.FeedOptions{
			Kind: query.Kind, Zone: domain.Zone(query.Zone), Period: query.Period,
			ID: query.ID, Status: query.Status, Sort: query.Sort, Order: query.Order,
			Page: query.Page, Limit: query.Limit,
		})
		respond(c, feed, err)
	})
	router.GET("/discover/actors", noStore(), func(c *gin.Context) {
		query, ok := bindQuery[struct {
			Query string `form:"q" binding:"required"`
		}](c)
		if !ok {
			return
		}
		actors, err := manager.ExploreActors(c.Request.Context(), query.Query)
		respond(c, actors, err)
	})
	router.GET("/discover/lists", noStore(), func(c *gin.Context) {
		query, ok := bindQuery[exploreListQuery](c)
		if !ok {
			return
		}
		options := javdb.ListOptions{Page: query.Page, Limit: query.Limit}
		switch query.Mode {
		case "search":
			options.Query = query.Query
		case "mine":
			options.Mine = true
		case "related":
			options.MovieID = query.MovieID
		}
		lists, err := manager.ExploreLists(c.Request.Context(), options)
		respond(c, lists, err)
	})
	router.GET("/discover/lists/:id", noStore(), func(c *gin.Context) {
		id, ok := bindURI[movieURI](c)
		if !ok {
			return
		}
		list, err := manager.ExploreList(c.Request.Context(), id.ID)
		respond(c, list, err)
	})
	router.GET("/discover/movies/:id/reviews", noStore(), func(c *gin.Context) {
		id, ok := bindURI[movieURI](c)
		if !ok {
			return
		}
		query, ok := bindQuery[explorePageQuery](c)
		if !ok {
			return
		}
		reviews, err := manager.ExploreReviews(c.Request.Context(), id.ID, query.Page, query.Limit)
		respond(c, reviews, err)
	})
}
