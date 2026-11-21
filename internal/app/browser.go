package app

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ppxb/miyabi"
	"github.com/ppxb/miyabi/internal/api"
	"github.com/ppxb/miyabi/internal/browsehistory"
	"github.com/ppxb/miyabi/internal/catalogue"
	"github.com/ppxb/miyabi/internal/config"
	"github.com/ppxb/miyabi/internal/database"
	"github.com/ppxb/miyabi/internal/drive"
	"github.com/ppxb/miyabi/internal/network"
	"github.com/ppxb/miyabi/internal/transfer"
)

// BrowserApp has no library scanners, metadata providers, subscriptions, Emby
// clients or STRM relay. Existing library records remain untouched on upgrade.
type BrowserApp struct {
	store *database.Store
	drive *drive.Drive
	catalogue *catalogue.Service
	downloads *transfer.Service
	server *http.Server
	logger *slog.Logger
}

func NewBrowser(cfg *config.Config,logger *slog.Logger) (*BrowserApp,error) {
	if logger==nil { logger=slog.Default() }
	ctx:=context.Background()
	store,err:=database.Open(ctx,cfg.DataDir)
	if err!=nil{return nil,err}
	networkSvc,err:=network.New(ctx,store.Client)
	if err!=nil{_ =store.Close();return nil,err}
	d,err:=drive.New(ctx,store.Client)
	if err!=nil{_ =store.Close();return nil,err}
	catalogueSvc,err:=catalogue.NewBrowser(ctx,store.Client,networkSvc.ProxyManager())
	if err!=nil{d.Close();_ =store.Close();return nil,err}
	downloads:=transfer.New(store.Client,d)
	router:=api.NewRouter(api.Dependencies{
		Browser:true,Logger:logger,Access:api.NewAccessGateService(cfg.AccessPassword,cfg.JWTSecret),
		Catalogue:catalogueSvc,Explore:catalogueSvc,JavDB:catalogueSvc,Drive:d,Network:networkSvc,
		Downloads:downloads,History:browsehistory.New(store.Client),Frontend:miyabi.Frontend(),TrustedProxies:cfg.TrustedProxies,
	})
	return &BrowserApp{store:store,drive:d,catalogue:catalogueSvc,downloads:downloads,logger:logger,
		server:&http.Server{Addr:cfg.Listen,Handler:router,ReadHeaderTimeout:10*time.Second,IdleTimeout:60*time.Second}},nil
}

func (a *BrowserApp) Run(ctx context.Context) error {
	ctx,cancel:=context.WithCancel(ctx)
	defer cancel()
	a.server.BaseContext=func(net.Listener)context.Context{return ctx}
	var worker sync.WaitGroup
	worker.Add(1)
	go func(){
		defer worker.Done()
		ticker:=time.NewTicker(30*time.Second)
		defer ticker.Stop()
		for {
			if err:=a.downloads.Sync(ctx);err!=nil && ctx.Err()==nil{a.logger.ErrorContext(ctx,"download synchronization failed","error",err)}
			select{case <-ctx.Done():return;case <-ticker.C:}
		}
	}()
	serverError:=make(chan error,1)
	go func(){a.logger.Info("miyabi browser started","address",a.server.Addr);serverError<-a.server.ListenAndServe()}()
	var err error
	select{case <-ctx.Done():case err=<-serverError:}
	if errors.Is(err,http.ErrServerClosed){err=nil}
	cancel()
	shutdown,stop:=context.WithTimeout(context.Background(),30*time.Second)
	defer stop()
	if shutdownErr:=a.server.Shutdown(shutdown);shutdownErr!=nil{_ =a.server.Close();err=errors.Join(err,shutdownErr)}
	worker.Wait()
	return err
}

func(a *BrowserApp) Close() error {
	a.catalogue.Close()
	a.drive.Close()
	return a.store.Close()
}
