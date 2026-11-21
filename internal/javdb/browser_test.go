package javdb

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestBrowserDoesNotScrapeOrFailOverOnRejection(t *testing.T){
	for _,status:=range []int{http.StatusForbidden,http.StatusTooManyRequests}{
		calls:=0
		client:=clientWithTransport(retryTransport(func()error{calls++;return &HTTPError{StatusCode:status,RetryAfter:time.Second}}))
		client.options.APIOnly=true
		client.webFallback.Store(true) // A legacy gate must never enable HTML in browser mode.
		_,err:=client.MovieDetail(t.Context(),"movie")
		if err==nil || calls!=1{t.Fatalf("status=%d calls=%d err=%v",status,calls,err)}
		if status==http.StatusForbidden && !errors.Is(err,ErrLoginRequired){t.Fatalf("missing login hint: %v",err)}
	}
}
