package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBrowserRetiresMediaServerRoutes(t *testing.T){
	router:=NewRouter(Dependencies{Browser:true,Access:NewAccessGateService("","")})
	for _,route:=range []struct{method,path string}{
		{"GET","/api/library/movies"},{"POST","/api/library/scan"},{"GET","/api/subscriptions"},
		{"GET","/api/settings/emby"},{"GET","/api/settings/subscription"},{"GET","/api/strm/play/1"},
		{"GET","/api/tasks"},{"GET","/api/tasks/events"},{"GET","/api/offline/tasks"},
	}{
		response:=httptest.NewRecorder()
		router.ServeHTTP(response,httptest.NewRequest(route.method,route.path,nil))
		if response.Code!=http.StatusNotFound{t.Errorf("%s %s: %d",route.method,route.path,response.Code)}
	}
}
