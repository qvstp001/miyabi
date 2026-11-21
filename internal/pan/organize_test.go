package pan

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFileMutationRequestsAndRejections(t *testing.T) {
	for _,test:=range []struct{
		name,path string
		form map[string]string
		call func(*Client)error
	}{
		{name:"rename",path:"/open/ufile/update",form:map[string]string{"file_id":"video","file_name":"ABC-123.zh.srt"},call:func(c *Client)error{return c.Rename(context.Background(),"fixture-token","video","ABC-123.zh.srt")}},
		{name:"move",path:"/open/ufile/move",form:map[string]string{"file_ids":"video","to_cid":"root"},call:func(c *Client)error{return c.Move(context.Background(),"fixture-token","video","root")}},
		{name:"delete empty directory",path:"/open/ufile/delete",form:map[string]string{"file_ids":"folder"},call:func(c *Client)error{return c.Delete(context.Background(),"fixture-token","folder")}},
	}{
		for _,rejected:=range []bool{false,true}{
			t.Run(test.name+map[bool]string{false:" accepted",true:" rejected"}[rejected],func(t *testing.T){
				client:=New()
				defer client.Close()
				client.http.SetTransport(offlineRoundTrip(func(r *http.Request)(*http.Response,error){
					if r.Method!=http.MethodPost || r.URL.Path!=test.path || r.Header.Get("Authorization")!="Bearer fixture-token"{t.Errorf("unexpected mutation request %s %s",r.Method,r.URL.Path)}
					if err:=r.ParseForm();err!=nil{t.Fatal(err)}
					for key,value:=range test.form{if r.FormValue(key)!=value{t.Errorf("%s=%q",key,r.FormValue(key))}}
					body:=`{"state":true,"code":0}`
					if rejected{body=`{"state":false,"code":500001,"message":"fixture rejection"}`}
					return &http.Response{StatusCode:http.StatusOK,Header:http.Header{"Content-Type":{"application/json"}},Body:io.NopCloser(strings.NewReader(body)),Request:r},nil
				}))
				if err:=test.call(client);(err!=nil)!=rejected{t.Fatalf("rejected=%t err=%v",rejected,err)}
			})
		}
	}
}
