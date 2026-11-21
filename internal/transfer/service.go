package transfer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"slices"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"

	"github.com/ppxb/miyabi/internal/codeid"
	"github.com/ppxb/miyabi/internal/domain"
	"github.com/ppxb/miyabi/internal/domain/download"
	"github.com/ppxb/miyabi/internal/drive"
	"github.com/ppxb/miyabi/internal/ent"
	"github.com/ppxb/miyabi/internal/ent/offlinedownload"
	"github.com/ppxb/miyabi/internal/pan"
	"github.com/ppxb/miyabi/internal/syncx"
)

// Service handles only downloads explicitly selected by the user. Polling and
// retries read 115/LitePan state and never query the catalogue.
type Service struct {
	db *ent.Client
	drive Drive
	mu syncx.ContextLock
}

// Drive supplies source-bound file access without exposing 115 credentials.
type Drive interface {
	OpenFiles(context.Context) (drive.FileSession, error)
	StartWork() (func(), bool)
}

type Input struct {
	Code string `json:"code" binding:"required,max=100"`
	MovieID string `json:"movie_id" binding:"max=100"`
	URI string `json:"uri" binding:"required,max=16384"`
	AccountID string `json:"account_id" binding:"required_with=DirectoryID"`
	DirectoryID string `json:"directory_id" binding:"required_with=AccountID"`
}

type Task struct {
	ID int `json:"id"`
	Code string `json:"code"`
	MovieID string `json:"movie_id"`
	URI string `json:"uri"`
	Stage string `json:"stage"`
	Status string `json:"status"`
	Progress int `json:"progress"`
	Error *string `json:"error,omitempty"`
	Files []download.TransferFile `json:"files"`
	AccountID string `json:"account_id"`
	DirectoryID string `json:"directory_id"`
}

func New(db *ent.Client,d Drive) *Service { return &Service{db:db,drive:d} }

func project(r *ent.OfflineDownload) Task {
	t := r.Recovery.Transfer
	return Task{ID:r.ID,Code:r.Code,MovieID:r.JavdbID,URI:t.URI,Stage:t.Stage,Status:string(r.Status),Progress:r.Progress,Error:r.Error,Files:t.Files,AccountID:r.AccountID,DirectoryID:r.DirectoryID}
}

func (s *Service) List(ctx context.Context) ([]Task,error) {
	records,err := s.db.OfflineDownload.Query().Where(transferTask,offlinedownload.StatusNEQ(offlinedownload.StatusRunning)).Order(ent.Desc(offlinedownload.FieldID)).Limit(200).All(ctx)
	if err != nil { return nil,err }
	active,err:=s.db.OfflineDownload.Query().Where(transferTask,offlinedownload.StatusEQ(offlinedownload.StatusRunning)).All(ctx)
	if err!=nil{return nil,err}
	records=append(records,active...)
	slices.SortFunc(records,func(a,b *ent.OfflineDownload)int{return b.ID-a.ID})
	result := make([]Task,0)
	for _,r := range records {
		if r.Recovery != nil && r.Recovery.Transfer != nil { result = append(result,project(r)) }
	}
	return result,nil
}

func transferTask(selector *sql.Selector) {
	selector.Where(sqljson.HasKey(selector.C(offlinedownload.FieldRecovery),sqljson.Path("transfer")))
}

func (s *Service) Add(ctx context.Context,input Input) (Task,error) {
	resource,err := ParseResource(input.URI)
	if err != nil { return Task{},domain.E(domain.KindInvalid,err.Error(),nil) }
	code := codeid.Normalize(input.Code)
	if !codeid.Valid(code) || strings.ContainsAny(code,"/\\\x00\r\n:*?\"<>|") {
		return Task{},domain.E(domain.KindInvalid,"番号不能用作文件名",nil)
	}
	if err := s.mu.Lock(ctx); err != nil { return Task{}, err }
	defer s.mu.Unlock()
	cfg,err := s.config(ctx)
	if err != nil { return Task{},err }
	if err := validateConfig(cfg); err != nil { return Task{},domain.E(domain.KindInvalid,err.Error(),nil) }
	sess,err := s.drive.OpenFiles(ctx)
	if err != nil { return Task{},err }
	source := sess.Source()
	if (input.AccountID != "" || input.DirectoryID != "") && (input.AccountID != source.AccountID || input.DirectoryID != source.Directory.ID) { return Task{},drive.ErrSourceChanged }
	if cfg.AccountID!=source.AccountID || cfg.DirectoryID!=source.Directory.ID { return Task{},domain.E(domain.KindInvalid,"下载目录已变更，请重新保存对应的 LitePan 路径",nil) }
	previous,err := s.db.OfflineDownload.Query().Where(
		offlinedownload.AccountIDEQ(source.AccountID),offlinedownload.HashEQ(resource.Key),
		offlinedownload.StatusNEQ(offlinedownload.StatusCancelled)).Order(ent.Desc(offlinedownload.FieldID)).All(ctx)
	if err != nil { return Task{},err }
	for _,r := range previous {
		if r.Recovery == nil || r.Recovery.Transfer == nil { continue }
		if !codeid.IsEquivalent(r.Code,code) { return Task{},domain.E(domain.KindConflict,"该资源已关联到影片 "+r.Code+"，请先核对下载任务",nil) }
		if r.DirectoryID != source.Directory.ID { return Task{},domain.E(domain.KindConflict,"该资源已有其他目录的下载任务",nil) }
		return project(r),nil
	}
	work, cancel := context.WithTimeout(context.WithoutCancel(ctx),2*time.Minute)
	defer cancel()
	done,ok := s.drive.StartWork()
	if !ok { return Task{},context.Canceled }
	defer done()
	var r *ent.OfflineDownload
	err = sess.Commit(work,func(tx *ent.Tx) error {
		var err error
		r,err = tx.OfflineDownload.Create().SetCode(code).SetJavdbID(input.MovieID).SetHash(resource.Key).
			SetAccountID(source.AccountID).SetDirectoryID(source.Directory.ID).
			SetRecovery(&download.Recovery{Transfer:&download.Transfer{URI:resource.URI,Stage:"submitting"}}).Save(work)
		return err
	})
	if err != nil { return Task{},err }
	hash,err := sess.AddOffline(work,resource.URI)
	if err != nil { return s.fail(work,r,fmt.Errorf("115 提交未确认，请先检查 115 任务再重试：%w",err)) }
	r.InfoHash,r.Recovery.Transfer.Stage = hash,"downloading"
	if err := s.save(work,r); err != nil { return Task{},err }
	return project(r),nil
}

func (s *Service) save(ctx context.Context,r *ent.OfflineDownload) error {
	err := s.db.OfflineDownload.UpdateOneID(r.ID).SetInfoHash(r.InfoHash).SetFileID(r.FileID).
		SetRecovery(r.Recovery).SetStatus(r.Status).SetProgress(r.Progress).ClearError().Exec(ctx)
	if err == nil { r.Error = nil }
	return err
}

func (s *Service) fail(ctx context.Context,r *ent.OfflineDownload,err error) (Task,error) {
	message := err.Error()
	r.Status,r.Error = offlinedownload.StatusFailed,&message
	if saveErr := s.db.OfflineDownload.UpdateOneID(r.ID).SetStatus(r.Status).SetRecovery(r.Recovery).
		SetInfoHash(r.InfoHash).SetFileID(r.FileID).SetProgress(r.Progress).SetError(message).Exec(ctx); saveErr != nil { return Task{},saveErr }
	return project(r),nil
}

func (s *Service) record(ctx context.Context,id int) (*ent.OfflineDownload,error) {
	r,err := s.db.OfflineDownload.Get(ctx,id)
	if err != nil { return nil,err }
	if r.Recovery == nil || r.Recovery.Transfer == nil { return nil,domain.E(domain.KindNotFound,"下载任务不存在",nil) }
	return r,nil
}

func (s *Service) Retry(ctx context.Context,id int) (Task,error) {
	if err := s.mu.Lock(ctx); err != nil { return Task{}, err }
	defer s.mu.Unlock()
	r,err := s.record(ctx,id)
	if err != nil { return Task{},err }
	if r.Status != offlinedownload.StatusFailed { return project(r),nil }
	if r.Recovery.Transfer.Stage == "done" {
		r.Status,r.Progress = offlinedownload.StatusDone,100
		if err := s.save(ctx,r); err != nil { return Task{},err }
		return project(r),nil
	}
	sess,err := s.drive.OpenFiles(ctx)
	if err != nil { return Task{},err }
	if err := matchSource(sess,r); err != nil { return Task{},err }
	r.Status,r.Error = offlinedownload.StatusRunning,nil
	if err := s.save(ctx,r); err != nil { return Task{},err }
	return project(r),nil
}

func (s *Service) Cancel(ctx context.Context,id int) (Task,error) {
	if err := s.mu.Lock(ctx); err != nil { return Task{}, err }
	defer s.mu.Unlock()
	r,err := s.record(ctx,id)
	if err != nil { return Task{},err }
	if r.Status == offlinedownload.StatusDone || r.Status == offlinedownload.StatusCancelled { return project(r),nil }
	stage := r.Recovery.Transfer.Stage
	if stage == "done" {
		r.Status,r.Progress = offlinedownload.StatusDone,100
		if err := s.save(ctx,r); err != nil { return Task{},err }
		return project(r),nil
	}
	if r.InfoHash != "" && (needsRemote(stage) || stage == "cancelling") {
		sess,err := s.drive.OpenFiles(ctx)
		if err != nil { return Task{},err }
		if err := matchSource(sess,r); err != nil { return Task{},err }
		r.Recovery.Transfer.Stage = "cancelling"
		r.Status = offlinedownload.StatusRunning
		if err := s.save(ctx,r); err != nil { return Task{},err }
		if err := cancelRemote(ctx,sess,r); err != nil {
			if ctx.Err() != nil { return Task{},err }
			state,saveErr := s.fail(ctx,r,err)
			if saveErr != nil { return Task{},saveErr }
			return state,domain.E(domain.KindUpstream,"115 取消结果未确认，请重试当前阶段",err)
		}
		if err := s.save(ctx,r); err != nil { return Task{},err }
		return project(r),nil
	}
	r.Status = offlinedownload.StatusCancelled
	if err := s.save(ctx,r); err != nil { return Task{},err }
	return project(r),nil
}

func cancelRemote(ctx context.Context,sess drive.Session,r *ent.OfflineDownload) error {
	if r.InfoHash == "" { return fmt.Errorf("115 任务标识缺失，无法取消远程任务") }
	if err := sess.RemoveOffline(ctx,r.InfoHash); err != nil && !errors.Is(err,pan.ErrNotFound) { return err }
	r.Status = offlinedownload.StatusCancelled
	return nil
}

func needsRemote(stage string) bool {
	return stage == "submitting" || stage == "downloading" || stage == "locating"
}

func matchSource(sess drive.Session,r *ent.OfflineDownload) error {
	source := sess.Source()
	if source.AccountID != r.AccountID || source.Directory.ID != r.DirectoryID { return domain.E(domain.KindConflict,"请切回该任务原来的 115 账号和下载目录",nil) }
	return nil
}

func (s *Service) Sync(ctx context.Context) error {
	records,err := s.db.OfflineDownload.Query().Where(transferTask,offlinedownload.StatusEQ(offlinedownload.StatusRunning)).All(ctx)
	if err != nil { return err }
	var jobs []*ent.OfflineDownload
	needRemote := false
	var failures []error
	for _,r := range records {
		if r.Recovery == nil || r.Recovery.Transfer == nil { continue }
		if r.Recovery.Transfer.Stage == "done" {
			if err := s.syncRecord(ctx,nil,r.ID,nil,nil); err != nil { failures = append(failures,err) }
			continue
		}
		jobs = append(jobs,r)
		needRemote = needRemote || needsRemote(r.Recovery.Transfer.Stage)
	}
	if len(jobs) == 0 { return errors.Join(failures...) }
	sess,err := s.drive.OpenFiles(ctx)
	if err != nil { return errors.Join(append(failures,err)...) }
	remote := make(map[string]pan.OfflineTask)
	var remoteErr error
	if needRemote {
		remoteErr = drive.WalkOfflinePages(ctx,func(page int)(pan.OfflinePage,error){return sess.OfflineTasks(ctx,page)},func(page pan.OfflinePage)(bool,error){
			for _,task := range page.Tasks {
				remote[strings.ToLower(task.Hash)] = task
				if resource,err := ParseResource(task.URL); err==nil { remote[resource.Key] = task }
			}
			return true,nil
		})
		if remoteErr != nil { failures = append(failures,remoteErr) }
	}
	for _,r := range jobs {
		if err := ctx.Err(); err != nil { return err }
		if err := s.syncRecord(ctx,sess,r.ID,remote,remoteErr); err != nil { failures = append(failures,err) }
	}
	return errors.Join(failures...)
}

func (s *Service) syncRecord(ctx context.Context,sess drive.FileSession,id int,remote map[string]pan.OfflineTask,remoteErr error) error {
	if err := s.mu.Lock(ctx); err != nil { return err }
	defer s.mu.Unlock()
	// Reads of the offline list may overlap a user cancellation or retry.
	r,err := s.record(ctx,id)
	if err != nil { return err }
	if r.Status != offlinedownload.StatusRunning { return nil }
	if remoteErr != nil && needsRemote(r.Recovery.Transfer.Stage) { return nil }
	if err := s.advance(ctx,sess,r,remote); err != nil {
		if ctx.Err() != nil { return ctx.Err() }
		_,saveErr := s.fail(ctx,r,err)
		return saveErr
	}
	return nil
}

func (s *Service) advance(ctx context.Context,sess drive.FileSession,r *ent.OfflineDownload,remote map[string]pan.OfflineTask) error {
	t := r.Recovery.Transfer
	if t.Stage == "done" {
		r.Status,r.Progress = offlinedownload.StatusDone,100
		return s.save(ctx,r)
	}
	if err := matchSource(sess,r); err != nil { return err }
	if t.Stage == "cancelling" {
		if err := cancelRemote(ctx,sess,r); err != nil { return err }
		return s.save(ctx,r)
	}
	if needsRemote(t.Stage) {
		hash := r.InfoHash
		if hash == "" { resource,err := ParseResource(t.URI); if err != nil { return err }; hash = resource.Hash }
		task,found := remote[strings.ToLower(hash)]
		if !found { task,found = remote[r.Hash] }
		if !found {
			if !t.CompletedAt.IsZero() && time.Since(t.CompletedAt)<5*time.Minute { return nil }
			if !t.CompletedAt.IsZero() { return fmt.Errorf("115 下载已完成但任务记录已移除，无法确认文件位置，请在 115 中核对") }
			if time.Since(r.CreatedAt)<2*time.Minute { return nil }
			return fmt.Errorf("115 未找到任务；提交结果不明确时不会重复提交，请在 115 中确认")
		}
		if task.DirectoryID != "" && task.DirectoryID != r.DirectoryID { return fmt.Errorf("115 任务目录与指定下载目录不一致") }
		if task.Hash != "" { r.InfoHash = task.Hash }
		if !task.ProgressUnknown { r.Progress = task.Progress }
		switch task.Status {
		case 0,1:
			if !t.CompletedAt.IsZero() { return fmt.Errorf("115 已完成的任务状态发生变化，请在 115 中核对") }
			t.Stage = "downloading"
			return s.save(ctx,r)
		case 2:
			if t.CompletedAt.IsZero() { t.CompletedAt = time.Now().UTC() }
			r.Progress,t.Stage = 100,"locating"
			if task.FileID == "" || task.FileID == "0" {
				if time.Since(t.CompletedAt)>=5*time.Minute { return fmt.Errorf("115 已完成下载但未提供文件位置，请在 115 中确认结果后重试") }
				return s.save(ctx,r)
			}
			r.FileID,r.Progress,t.Stage = task.FileID,100,"planning"
			if err := s.save(ctx,r); err != nil { return err }
		case -1: return fmt.Errorf("115 下载失败，请手动选择其他资源")
		default: return fmt.Errorf("115 返回未知任务状态 %d",task.Status)
		}
	}
	if t.Stage == "planning" {
		files,dirs,err := makePlan(ctx,sess,r.FileID,r.Code)
		if err != nil { return err }
		t.Files,t.Directories,t.Stage = files,dirs,"organizing"
		if err := s.save(ctx,r); err != nil { return err }
	}
	if t.Stage == "organizing" {
		if err := organize(ctx,sess,t); err != nil { return err }
		t.Stage = "pushing"
		if err := s.save(ctx,r); err != nil { return err }
	}
	if t.Stage == "pushing" {
		if err := verifyFiles(ctx,sess,t.Files); err != nil { return err }
		cfg,err := s.config(ctx)
		if err != nil { return err }
		if cfg.AccountID!=r.AccountID || cfg.DirectoryID!=r.DirectoryID { return fmt.Errorf("请重新保存该下载目录对应的 LitePan 路径") }
		if err := pushLitePan(ctx,cfg); err != nil { return err }
		t.Stage,r.Status = "done",offlinedownload.StatusDone
		return s.save(ctx,r)
	}
	return fmt.Errorf("未知下载处理阶段 %q",t.Stage)
}
