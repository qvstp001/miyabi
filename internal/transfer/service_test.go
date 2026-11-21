package transfer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ppxb/miyabi/internal/database"
	"github.com/ppxb/miyabi/internal/domain"
	"github.com/ppxb/miyabi/internal/domain/download"
	"github.com/ppxb/miyabi/internal/drive"
	"github.com/ppxb/miyabi/internal/ent"
	"github.com/ppxb/miyabi/internal/ent/offlinedownload"
	"github.com/ppxb/miyabi/internal/pan"
)

type transferSession struct {
	*fileFixture
	db *ent.Client
	remote []pan.OfflineTask
	listErr error
	list func(context.Context,int) (pan.OfflinePage,error)
	add func(context.Context,string) (string,error)
	remove func(context.Context,string) error
	addCalls,removeCalls,listCalls int
	guardErr error
}

func (s *transferSession) Commit(ctx context.Context,fn func(*ent.Tx) error) error { return ent.WithTx(ctx,s.db,fn) }
func (s *transferSession) WithSource(_ context.Context,fn func() error) error { if s.guardErr!=nil{return s.guardErr};return fn() }
func (s *transferSession) AddOffline(ctx context.Context,uri string)(string,error){s.addCalls++;return s.add(ctx,uri)}
func (s *transferSession) RemoveOffline(ctx context.Context,hash string)error{s.removeCalls++;if s.remove!=nil{return s.remove(ctx,hash)};return nil}
func (s *transferSession) OfflineTasks(ctx context.Context,page int)(pan.OfflinePage,error){s.listCalls++;if s.list!=nil{return s.list(ctx,page)};return pan.OfflinePage{PageCount:1,Tasks:s.remote},s.listErr}

type transferDrive struct {
	session drive.FileSession
	openErr error
	starts,finishes int
}

func (d *transferDrive) OpenFiles(context.Context)(drive.FileSession,error){return d.session,d.openErr}
func (d *transferDrive) StartWork()(func(),bool){d.starts++;return func(){d.finishes++},true}

func transferFixture(t *testing.T) (*Service,*transferSession,*transferDrive,*atomic.Int32) {
	t.Helper()
	store,err:=database.Open(t.Context(),t.TempDir())
	if err!=nil{t.Fatal(err)}
	t.Cleanup(func(){_ = store.Close()})
	sess:=&transferSession{fileFixture:newFilesFixture(),db:store.Client}
	d:=&transferDrive{session:sess}
	s:=New(store.Client,d)
	pushes:=new(atomic.Int32)
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		pushes.Add(1)
		_,_=w.Write([]byte(`{"success":true,"data":{"matched":1}}`))
	}))
	t.Cleanup(server.Close)
	cfg:=Config{LitePanURL:server.URL,Path:"/115/movies",APIKey:"fixture-key",AccountID:"account",DirectoryID:"root"}
	if err:=database.SaveSetting(t.Context(),s.db,configKey,cfg);err!=nil{t.Fatal(err)}
	return s,sess,d,pushes
}

func createTransfer(t *testing.T,s *Service,stage string) *ent.OfflineDownload {
	t.Helper()
	return s.db.OfflineDownload.Create().SetCode("ABC-123").SetHash(strings.Repeat("a",40)).
		SetInfoHash(strings.Repeat("b",40)).SetAccountID("account").SetDirectoryID("root").
		SetRecovery(&download.Recovery{Transfer:&download.Transfer{URI:"magnet:?xt=urn:btih:"+strings.Repeat("a",40),Stage:stage}}).SaveX(t.Context())
}

func TestDownloadPipelineForMagnetAndED2K(t *testing.T) {
	for _,uri:=range []string{
		"magnet:?xt=urn:btih:"+strings.Repeat("a",40),
		"ed2k://|file|original.mp4|123456789|"+strings.Repeat("c",32)+"|/",
	}{
		t.Run(uri[:6],func(t *testing.T){
			s,sess,d,pushes:=transferFixture(t)
			for id,name:=range map[string]string{"srt":"ORIGINAL.zh.SRT","idx":"original.IDX","sub":"original.SUB","sup":"original.sup","smi":"original.smi"}{
				sess.files[id]=pan.FileInfo{File:pan.File{ID:id,Name:name,ParentID:"folder"},Path:[]pan.Directory{{ID:"root"},{ID:"folder"}}}
			}
			sess.files["subs"]=pan.FileInfo{File:pan.File{ID:"subs",Name:"Subs",ParentID:"folder",IsDirectory:true},Path:[]pan.Directory{{ID:"root"},{ID:"folder"}}}
			subtitle:=sess.files["srt"]
			subtitle.ParentID="subs"
			subtitle.Path=[]pan.Directory{{ID:"root"},{ID:"folder"},{ID:"subs"}}
			sess.files["srt"]=subtitle
			sess.files["ad"]=pan.FileInfo{File:pan.File{ID:"ad",Name:"advert.jpg",ParentID:"folder"}}
			sess.add=func(_ context.Context,submitted string)(string,error){
				if submitted!=uri{t.Errorf("link changed: %q",submitted)}
				sess.remote=[]pan.OfflineTask{{Hash:strings.Repeat("b",40),URL:submitted,Status:2,FileID:"folder",DirectoryID:"root"}}
				return strings.Repeat("b",40),nil
			}
			task,err:=s.Add(t.Context(),Input{Code:"ABC-123",MovieID:"movie",URI:uri})
			if err!=nil{t.Fatal(err)}
			if _,err:=s.Add(t.Context(),Input{Code:"ABC-123",MovieID:"movie",URI:uri});err!=nil{t.Fatal(err)}
			if err:=s.Sync(t.Context());err!=nil{t.Fatal(err)}
			saved:=s.db.OfflineDownload.GetX(t.Context(),task.ID)
			if saved.Status!=offlinedownload.StatusDone || saved.InfoHash!=strings.Repeat("b",40) || pushes.Load()!=1{t.Fatalf("pipeline failed: %+v pushes=%d",saved,pushes.Load())}
			for id,name:=range map[string]string{"video":"ABC-123.mp4","srt":"ABC-123.zh.srt","idx":"ABC-123.idx","sub":"ABC-123.sub","sup":"ABC-123.sup","smi":"ABC-123.smi"}{
				f:=sess.files[id]
				if f.Name!=name || f.ParentID!="root"{t.Errorf("file %s: %+v",id,f)}
			}
			if sess.deletes!=1 || sess.files["ad"].ParentID!="folder"{t.Fatal("extra files or their folder were removed")}
			if _,ok:=sess.files["folder"];!ok{t.Fatal("non-empty wrapper was removed")}
			if _,ok:=sess.files["subs"];ok{t.Fatal("empty subtitle folder was retained")}
			if err:=s.Sync(t.Context());err!=nil{t.Fatal(err)}
			if sess.addCalls!=1 || pushes.Load()!=1 || d.starts!=1 || d.finishes!=1{t.Fatal("duplicate submission or replay")}
		})
	}
}

func TestOfflineListFailureDoesNotBlockOrganizationAndPush(t *testing.T) {
	s,sess,_,pushes:=transferFixture(t)
	pending:=createTransfer(t,s,"downloading")
	s.db.OfflineDownload.UpdateOneID(pending.ID).SetProgress(42).ExecX(t.Context())
	ready:=createTransfer(t,s,"organizing")
	ready.Recovery.Transfer.Files=[]download.TransferFile{{ID:"video",Name:"ABC-123.mp4"}}
	ready.Recovery.Transfer.Directories=[]string{"folder"}
	if err:=s.save(t.Context(),ready);err!=nil{t.Fatal(err)}
	sess.listErr=errors.New("offline endpoint unavailable")
	if err:=s.Sync(t.Context());!errors.Is(err,sess.listErr){t.Fatalf("missing endpoint error: %v",err)}
	if r:=s.db.OfflineDownload.GetX(t.Context(),pending.ID);r.Status!=offlinedownload.StatusRunning || r.Progress!=42{t.Fatalf("pending download regressed: %+v",r)}
	if r:=s.db.OfflineDownload.GetX(t.Context(),ready.ID);r.Status!=offlinedownload.StatusDone || pushes.Load()!=1{t.Fatalf("ready job was blocked: %+v",r)}
}

func TestCompletionWithoutLocationIsBoundedAndResumable(t *testing.T) {
	s,sess,_,pushes:=transferFixture(t)
	r:=createTransfer(t,s,"downloading")
	remote:=map[string]pan.OfflineTask{r.InfoHash:{Hash:r.InfoHash,Status:2,FileID:"0"}}
	if err:=s.advance(t.Context(),sess,r,remote);err!=nil{t.Fatal(err)}
	r=s.db.OfflineDownload.GetX(t.Context(),r.ID)
	if r.Progress!=100 || r.Recovery.Transfer.Stage!="locating" || r.Recovery.Transfer.CompletedAt.IsZero(){t.Fatalf("bad location checkpoint: %+v",r)}
	r.Recovery.Transfer.CompletedAt=time.Now().Add(-6*time.Minute)
	if err:=s.advance(t.Context(),sess,r,remote);err==nil{t.Fatal("missing location waited forever")}
	remote[r.InfoHash]=pan.OfflineTask{Hash:r.InfoHash,Status:2,FileID:"folder"}
	if err:=s.advance(t.Context(),sess,r,remote);err!=nil{t.Fatal(err)}
	if pushes.Load()!=1{t.Fatal("late location was not processed")}
}

func TestUnknownProgressDoesNotResetDownload(t *testing.T) {
	s,sess,_,_:=transferFixture(t)
	r:=createTransfer(t,s,"downloading")
	r.Progress=73
	if err:=s.advance(t.Context(),sess,r,map[string]pan.OfflineTask{r.InfoHash:{Hash:r.InfoHash,Status:1,ProgressUnknown:true}});err!=nil{t.Fatal(err)}
	if r.Progress!=73{t.Fatal("missing upstream percentage reset progress")}
}

func TestCancelSurvivesLostResponseWithoutDeletingSourceFiles(t *testing.T) {
	s,sess,_,_:=transferFixture(t)
	r:=createTransfer(t,s,"downloading")
	ctx,cancel:=context.WithCancel(t.Context())
	defer cancel()
	sess.remove=func(context.Context,string)error{cancel();return context.Canceled}
	if _,err:=s.Cancel(ctx,r.ID);!errors.Is(err,context.Canceled){t.Fatalf("cancel error=%v",err)}
	r=s.db.OfflineDownload.GetX(t.Context(),r.ID)
	if r.Recovery.Transfer.Stage!="cancelling" || r.Status!=offlinedownload.StatusRunning{t.Fatal("cancel intent was lost")}
	sess.remove=func(context.Context,string)error{return pan.ErrNotFound}
	if err:=s.Sync(t.Context());err!=nil{t.Fatal(err)}
	if r:=s.db.OfflineDownload.GetX(t.Context(),r.ID);r.Status!=offlinedownload.StatusCancelled{t.Fatal("cancel was not resumed")}
	if sess.addCalls!=0 || sess.moves!=0 || sess.deletes!=0{t.Fatal("cancel mutated source content or redownloaded")}
}

func TestPushAcknowledgementSurvivesCompletionWriteFailure(t *testing.T) {
	s,sess,d,pushes:=transferFixture(t)
	r:=createTransfer(t,s,"pushing")
	v:=sess.files["video"]
	v.Name,v.ParentID="ABC-123.mp4","root"
	sess.files["video"]=v
	r.Recovery.Transfer.Files=[]download.TransferFile{{ID:"video",Name:"ABC-123.mp4"}}
	if err:=s.save(t.Context(),r);err!=nil{t.Fatal(err)}
	failOnce:=true
	s.db.OfflineDownload.Use(func(next ent.Mutator)ent.Mutator{return ent.MutateFunc(func(ctx context.Context,m ent.Mutation)(ent.Value,error){
		mutation:=m.(*ent.OfflineDownloadMutation)
		if status,ok:=mutation.Status();ok && status==offlinedownload.StatusDone && failOnce{failOnce=false;return nil,fmt.Errorf("injected completion write failure")}
		return next.Mutate(ctx,m)
	})})
	if err:=s.Sync(t.Context());err!=nil{t.Fatal(err)}
	saved:=s.db.OfflineDownload.GetX(t.Context(),r.ID)
	if saved.Status!=offlinedownload.StatusFailed || saved.Recovery.Transfer.Stage!="done" || pushes.Load()!=1{t.Fatal("acknowledged delivery checkpoint was lost")}
	d.openErr=errors.New("115 unavailable after delivery")
	if task,err:=s.Retry(t.Context(),r.ID);err!=nil || task.Status!="done"{t.Fatalf("completion retry: %+v %v",task,err)}
	if pushes.Load()!=1 || sess.moves!=0 || sess.renames!=0{t.Fatal("completion retry replayed remote effects")}
}

func TestPushRechecksFilesAfterOrganization(t *testing.T) {
	s,sess,_,pushes:=transferFixture(t)
	r:=createTransfer(t,s,"pushing")
	r.Recovery.Transfer.Files=[]download.TransferFile{{ID:"video",Name:"ABC-123.mp4"}}
	if err:=s.advance(t.Context(),sess,r,nil);err==nil{t.Fatal("push ignored a file outside the destination")}
	delete(sess.files,"video")
	if err:=s.advance(t.Context(),sess,r,nil);err==nil{t.Fatal("push ignored a removed file")}
	if pushes.Load()!=0{t.Fatal("LitePan was notified before files were ready")}
}

func TestUnconfirmedMoveDoesNotCleanFoldersOrPush(t *testing.T) {
	s,sess,_,pushes:=transferFixture(t)
	r:=createTransfer(t,s,"organizing")
	r.Recovery.Transfer.Files=[]download.TransferFile{{ID:"video",Name:"ABC-123.mp4",OriginalName:"original.mp4",ParentID:"folder"}}
	r.Recovery.Transfer.Directories=[]string{"folder"}
	if err:=s.save(t.Context(),r);err!=nil{t.Fatal(err)}
	sess.unconfirmedMove=true
	if err:=s.Sync(t.Context());err!=nil{t.Fatal(err)}
	if saved:=s.db.OfflineDownload.GetX(t.Context(),r.ID);saved.Status!=offlinedownload.StatusFailed{t.Fatal("unconfirmed move was accepted")}
	if sess.deletes!=0 || pushes.Load()!=0{t.Fatal("folders were cleaned or LitePan notified too soon")}
	sess.unconfirmedMove=false
	if _,err:=s.Retry(t.Context(),r.ID);err!=nil{t.Fatal(err)}
	if err:=s.Sync(t.Context());err!=nil{t.Fatal(err)}
	if sess.renames!=1 || sess.deletes!=1 || pushes.Load()!=1{t.Fatal("partial move retry did not resume safely")}
}

type sourceOverrideSession struct {
	drive.FileSession
	source domain.LibrarySource
}

func (s sourceOverrideSession) Source() domain.LibrarySource { return s.source }

func TestSourceChangePreventsFileMutationsAndPush(t *testing.T) {
	s,sess,_,pushes:=transferFixture(t)
	r:=createTransfer(t,s,"organizing")
	r.Recovery.Transfer.Files=[]download.TransferFile{{ID:"video",Name:"ABC-123.mp4"}}
	wrong:=sourceOverrideSession{FileSession:sess,source:domain.LibrarySource{AccountID:"other",Directory:domain.LibraryDirectory{ID:"other-root"}}}
	if err:=s.advance(t.Context(),wrong,r,nil);err==nil{t.Fatal("changed account/directory was accepted")}
	if sess.moves!=0 || sess.renames!=0 || sess.deletes!=0 || pushes.Load()!=0{t.Fatal("source change allowed side effects")}
}

func TestSettingsAreNotReboundAfterSourceGuardFails(t *testing.T) {
	s,sess,_,_:=transferFixture(t)
	cfg,err:=s.config(t.Context())
	if err!=nil{t.Fatal(err)}
	cfg.Path="/115/other"
	sess.guardErr=drive.ErrSourceChanged
	if _,err:=s.SaveConfig(t.Context(),cfg);!errors.Is(err,drive.ErrSourceChanged){t.Fatalf("guard ignored: %v",err)}
	stored,err:=s.config(t.Context())
	if err!=nil || stored.Path!="/115/movies"{t.Fatalf("path binding changed: %+v %v",stored,err)}
}

func TestLateOfflineSnapshotDoesNotResumeCancelledTask(t *testing.T) {
	s,sess,_,pushes:=transferFixture(t)
	r:=createTransfer(t,s,"downloading")
	sess.list=func(context.Context,int)(pan.OfflinePage,error){
		if _,err:=s.Cancel(t.Context(),r.ID);err!=nil{t.Fatal(err)}
		return pan.OfflinePage{PageCount:1,Tasks:[]pan.OfflineTask{{Hash:r.InfoHash,Status:2,FileID:"folder"}}},nil
	}
	if err:=s.Sync(t.Context());err!=nil{t.Fatal(err)}
	if saved:=s.db.OfflineDownload.GetX(t.Context(),r.ID);saved.Status!=offlinedownload.StatusCancelled{t.Fatal("late snapshot regressed cancellation")}
	if sess.moves!=0 || sess.renames!=0 || pushes.Load()!=0{t.Fatal("cancelled download was organized")}
}

func TestDownloadSubmissionRejectsStaleDirectorySelection(t *testing.T) {
	s,sess,_,_:=transferFixture(t)
	_,err:=s.Add(t.Context(),Input{Code:"ABC-123",URI:"magnet:?xt=urn:btih:"+strings.Repeat("a",40),AccountID:"account",DirectoryID:"old-root"})
	if !errors.Is(err,drive.ErrSourceChanged) || sess.addCalls!=0{t.Fatalf("stale target submitted: calls=%d err=%v",sess.addCalls,err)}
}

func TestDownloadSubmissionFinishesRecordingAfterCallerLeaves(t *testing.T) {
	s,sess,d,_:=transferFixture(t)
	ctx,cancel:=context.WithCancel(t.Context())
	defer cancel()
	sess.add=func(work context.Context,_ string)(string,error){
		cancel()
		if work.Err()!=nil{t.Fatal("submission still uses the departed caller's context")}
		return strings.Repeat("b",40),nil
	}
	task,err:=s.Add(ctx,Input{Code:"ABC-123",URI:"magnet:?xt=urn:btih:"+strings.Repeat("a",40)})
	if err!=nil{t.Fatal(err)}
	if saved:=s.db.OfflineDownload.GetX(t.Context(),task.ID);saved.InfoHash!=strings.Repeat("b",40) || saved.Recovery.Transfer.Stage!="downloading"{t.Fatal("remote submission was lost")}
	if d.starts!=1 || d.finishes!=1{t.Fatal("submission lifecycle was not tracked")}
}
