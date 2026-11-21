package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/ppxb/miyabi/internal/transfer"
)

type DownloadManager interface {
	List(context.Context) ([]transfer.Task,error)
	Add(context.Context,transfer.Input) (transfer.Task,error)
	Retry(context.Context,int) (transfer.Task,error)
	Cancel(context.Context,int) (transfer.Task,error)
	Config(context.Context) (transfer.Config,error)
	SaveConfig(context.Context,transfer.Config) (transfer.Config,error)
}

func installDownloadRoutes(api,settings *gin.RouterGroup,s DownloadManager) {
	api.GET("/downloads",noStore(),func(c *gin.Context){ v,err:=s.List(c.Request.Context());respond(c,v,err) })
	api.POST("/downloads",func(c *gin.Context){
		input,ok:=bindJSON[transfer.Input](c);if !ok{return}
		v,err:=s.Add(c.Request.Context(),input);accepted(c,v,err)
	})
	control := func(retry bool) gin.HandlerFunc { return func(c *gin.Context){
		p,ok:=bindURI[struct{ID int `uri:"id" binding:"required,min=1"`}](c);if !ok{return}
		var v transfer.Task;var err error
		if retry{v,err=s.Retry(c.Request.Context(),p.ID)}else{v,err=s.Cancel(c.Request.Context(),p.ID)}
		accepted(c,v,err)
	} }
	api.POST("/downloads/:id/retry",control(true))
	api.POST("/downloads/:id/cancel",control(false))
	settings.GET("/downloads",func(c *gin.Context){v,err:=s.Config(c.Request.Context());respond(c,v,err)})
	settings.PUT("/downloads",func(c *gin.Context){
		input,ok:=bindJSON[transfer.Config](c);if !ok{return}
		v,err:=s.SaveConfig(c.Request.Context(),input);respond(c,v,err)
	})
}
