package transfer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/ppxb/miyabi/internal/database"
	"github.com/ppxb/miyabi/internal/domain/download"
	"github.com/ppxb/miyabi/internal/ent/offlinedownload"
)

func TestLitePanFailureRetriesDeliveryWithoutReorganizing(t *testing.T){
	var calls atomic.Int32
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		count:=calls.Add(1)
		if r.URL.Path!="/api/open/automation/events" || r.Header.Get("Authorization")!="Bearer fixture-key"{t.Errorf("unexpected webhook %s",r.URL.Path)}
		var event map[string]string
		if err:=json.NewDecoder(r.Body).Decode(&event);err!=nil{t.Error(err)}
		if event["event"]!=EventName || event["source"]!="miyabi" || event["path"]!="/115/movies"{t.Errorf("event=%v",event)}
		if count==1{w.WriteHeader(http.StatusServiceUnavailable);return}
		_,_=w.Write([]byte(`{"success":true,"data":{"matched":1}}`))
	}))
	defer server.Close()
	store,err:=database.Open(t.Context(),t.TempDir())
	if err!=nil{t.Fatal(err)}
	defer store.Close()
	s:=New(store.Client,nil)
	config:=Config{LitePanURL:server.URL,APIKey:"fixture-key",Path:"/115/movies",AccountID:"account",DirectoryID:"root"}
	if err:=database.SaveSetting(t.Context(),store.Client,configKey,config);err!=nil{t.Fatal(err)}
	f:=newFilesFixture()
	r,err:=store.Client.OfflineDownload.Create().SetHash("fixture").SetCode("ABC-123").SetAccountID("account").SetDirectoryID("root").SetRecovery(&download.Recovery{Transfer:&download.Transfer{Stage:"organizing",Files:[]download.TransferFile{{ID:"video",Name:"ABC-123.mp4"}},Directories:[]string{"folder"}}}).Save(t.Context())
	if err!=nil{t.Fatal(err)}
	if err:=s.advance(t.Context(),f,r,nil);err==nil{t.Fatal("expected delivery failure")}
	saved:=store.Client.OfflineDownload.GetX(t.Context(),r.ID)
	if saved.Recovery.Transfer.Stage!="pushing"{t.Fatal("delivery checkpoint was not saved")}
	if err:=s.advance(t.Context(),f,saved,nil);err!=nil{t.Fatal(err)}
	saved=store.Client.OfflineDownload.GetX(t.Context(),r.ID)
	if calls.Load()!=2 || f.renames!=1 || f.moves!=1 || saved.Status!=offlinedownload.StatusDone{t.Fatalf("unexpected retry: calls=%d files=%+v task=%+v",calls.Load(),f,saved)}
	public,err:=s.Config(t.Context())
	if err!=nil || public.APIKey!="" || !public.HasAPIKey{t.Fatalf("secret exposed: %+v %v",public,err)}
}

func TestLitePanUnmatchedWebhookIsNotSuccess(t *testing.T){
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){_,_=w.Write([]byte(`{"success":true,"data":{"matched":0}}`))}))
	defer server.Close()
	if err:=pushLitePan(t.Context(),Config{LitePanURL:server.URL,APIKey:"key",Path:"/movies"});err==nil{t.Fatal("unmatched event treated as STRM delivery")}
}

func TestBrowserNeverResumesLegacyDownloads(t *testing.T){
	store,err:=database.Open(t.Context(),t.TempDir())
	if err!=nil{t.Fatal(err)}
	defer store.Close()
	legacy:=store.Client.OfflineDownload.Create().SetHash("legacy").SaveX(t.Context())
	s:=New(store.Client,nil)
	if err:=s.Sync(t.Context());err!=nil{t.Fatal(err)}
	list,err:=s.List(t.Context())
	if err!=nil || len(list)!=0{t.Fatalf("legacy job exposed: %+v %v",list,err)}
	if current:=store.Client.OfflineDownload.GetX(t.Context(),legacy.ID);current.Status!=legacy.Status{t.Fatal("legacy task was modified")}
}

func TestLitePanRejectsInvalidAcknowledgements(t *testing.T) {
	for _,test:=range []struct{name string;status int;body string}{
		{"unauthorized",http.StatusUnauthorized,`{"success":false}`},
		{"error envelope",http.StatusOK,`{"success":false,"data":{"matched":1}}`},
		{"invalid matched count",http.StatusOK,`{"success":true,"data":{"matched":-1}}`},
		{"malformed reply",http.StatusOK,`not json`},
	}{
		t.Run(test.name,func(t *testing.T){
			server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.WriteHeader(test.status);_,_=w.Write([]byte(test.body))}))
			defer server.Close()
			if err:=pushLitePan(t.Context(),Config{LitePanURL:server.URL,APIKey:"secret",Path:"/movies"});err==nil{t.Fatal("invalid acknowledgement was accepted")}
		})
	}
}

func TestLitePanDoesNotForwardAPIKeyOnRedirect(t *testing.T) {
	var forwarded atomic.Int32
	target:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){forwarded.Add(1)}))
	defer target.Close()
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){http.Redirect(w,r,target.URL,http.StatusTemporaryRedirect)}))
	defer server.Close()
	if err:=pushLitePan(t.Context(),Config{LitePanURL:server.URL,APIKey:"secret",Path:"/movies"});err==nil{t.Fatal("redirect was accepted")}
	if forwarded.Load()!=0{t.Fatal("redirect forwarded the API key")}
}
