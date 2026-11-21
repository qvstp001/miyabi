package transfer

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/ppxb/miyabi/internal/domain"
	"github.com/ppxb/miyabi/internal/domain/download"
	"github.com/ppxb/miyabi/internal/drive"
	"github.com/ppxb/miyabi/internal/pan"
)

func TestPlanFilesMatchesSubtitlesAndOrdersParts(t *testing.T){
	files:=[]locatedFile{
		{File:pan.File{ID:"10",Name:"part10.MKV",Size:domain.MinVideoSize},location:"part10.MKV"},
		{File:pan.File{ID:"2",Name:"part2.mp4",Size:domain.MinVideoSize},location:"part2.mp4"},
		{File:pan.File{ID:"sub",Name:"part2.zh.srt"},location:"part2.zh.srt"},
		{File:pan.File{ID:"ad",Name:"advert.txt"},location:"advert.txt"},
		{File:pan.File{ID:"sample",Name:"sample.mp4",Size:1024},location:"sample.mp4"},
	}
	plan,err:=planFiles("ABC-123",files)
	want:=[]download.TransferFile{{ID:"2",Name:"ABC-123-cd1.mp4",OriginalName:"part2.mp4"},{ID:"10",Name:"ABC-123-cd2.mkv",OriginalName:"part10.MKV"},{ID:"sub",Name:"ABC-123-cd1.zh.srt",OriginalName:"part2.zh.srt"}}
	if err!=nil || !reflect.DeepEqual(plan,want){t.Fatalf("plan=%+v err=%v",plan,err)}
	if _,err:=planFiles("ABC-123",files[3:]);err==nil{t.Fatal("non-video download accepted")}
}

func TestPlanFilesAssignsSubtitleToLongestMatchingVideo(t *testing.T){
	files:=[]locatedFile{
		{File:pan.File{ID:"a",Name:"movie.mp4",Size:domain.MinVideoSize},location:"movie.mp4"},
		{File:pan.File{ID:"b",Name:"movie.zh.mp4",Size:domain.MinVideoSize},location:"movie.zh.mp4"},
		{File:pan.File{ID:"s",Name:"movie.zh.srt"},location:"movie.zh.srt"},
	}
	plan,err:=planFiles("ABC-123",files)
	if err!=nil || len(plan)!=3 || plan[2].Name!="ABC-123-cd2.srt"{t.Fatalf("plan=%+v err=%v",plan,err)}
}

func TestPlanFilesRefusesMultiMoviePacks(t *testing.T){
	files:=[]locatedFile{{File:pan.File{ID:"foreign",Name:"XYZ-456.mp4",Size:domain.MinVideoSize},location:"XYZ-456.mp4"}}
	if _,err:=planFiles("ABC-123",files);err==nil{t.Fatal("another movie would have been relabeled as a part")}
}

type fileFixture struct{
	drive.Session
	files map[string]pan.FileInfo
	renames,moves,deletes int
	failMove bool
	unconfirmedMove bool
}

func newFilesFixture()*fileFixture{
	return &fileFixture{files:map[string]pan.FileInfo{
		"folder":{File:pan.File{ID:"folder",Name:"download",ParentID:"root",IsDirectory:true},Path:[]pan.Directory{{ID:"root"}}},
		"video":{File:pan.File{ID:"video",Name:"original.mp4",ParentID:"folder",Size:domain.MinVideoSize},Path:[]pan.Directory{{ID:"root"},{ID:"folder"}}},
	}}
}

func(f *fileFixture)Source()domain.LibrarySource{return domain.LibrarySource{AccountID:"account",Directory:domain.LibraryDirectory{ID:"root"}}}
func(f *fileFixture)Info(_ context.Context,id string)(pan.FileInfo,error){v,ok:=f.files[id];if !ok{return pan.FileInfo{},pan.ErrNotFound};return v,nil}
func(f *fileFixture)List(_ context.Context,id string,_ int)(pan.FilePage,error){
	page:=pan.FilePage{Path:[]pan.Directory{{ID:"root"},{ID:id}}}
	for _,v:=range f.files{if v.ParentID==id{page.Files=append(page.Files,v.File)}}
	page.Total=len(page.Files)
	return page,nil
}
func(f *fileFixture)Rename(_ context.Context,id,name string)error{v:=f.files[id];v.Name=name;f.files[id]=v;f.renames++;return nil}
func(f *fileFixture)Move(_ context.Context,id string)error{if f.failMove{f.failMove=false;return fmt.Errorf("temporary move error")};f.moves++;if f.unconfirmedMove{return nil};v:=f.files[id];v.ParentID="root";v.Path=[]pan.Directory{{ID:"root"}};f.files[id]=v;return nil}
func(f *fileFixture)DeleteEmpty(_ context.Context,id string)error{if _,ok:=f.files[id];!ok{return pan.ErrNotFound};for _,v:=range f.files{if v.ParentID==id{return nil}};delete(f.files,id);f.deletes++;return nil}

func TestOrganizeResumesAfterRenameAndKeepsExtras(t *testing.T){
	f:=newFilesFixture()
	f.files["ad"]=pan.FileInfo{File:pan.File{ID:"ad",Name:"advert.jpg",ParentID:"folder"}}
	f.failMove=true
	checkpoint:=&download.Transfer{Files:[]download.TransferFile{{ID:"video",Name:"ABC-123.mp4"}},Directories:[]string{"folder"}}
	if err:=organize(t.Context(),f,checkpoint);err==nil{t.Fatal("expected injected failure")}
	if err:=organize(t.Context(),f,checkpoint);err!=nil{t.Fatal(err)}
	if f.renames!=1 || f.moves!=1 || f.deletes!=0 || f.files["ad"].Name!="advert.jpg"{t.Fatalf("unexpected mutations: %+v",f)}
	delete(f.files,"ad")
	if err:=organize(t.Context(),f,checkpoint);err!=nil{t.Fatal(err)}
	if f.deletes!=1{t.Fatal("empty wrapper was not removed")}
}

func TestOrganizeStopsBeforeCollision(t *testing.T){
	f:=newFilesFixture()
	f.files["existing"]=pan.FileInfo{File:pan.File{ID:"existing",Name:"abc-123.MP4",ParentID:"root"}}
	err:=organize(t.Context(),f,&download.Transfer{Files:[]download.TransferFile{{ID:"video",Name:"ABC-123.mp4"}}})
	if err==nil || f.renames!=0 || f.moves!=0 || f.deletes!=0{t.Fatalf("collision mutated files: %+v %v",f,err)}
}

func TestOrganizePreflightsEveryOriginalFolder(t *testing.T) {
	f := newFilesFixture()
	f.files["second"] = pan.FileInfo{File: pan.File{ID: "second", Name: "part2.mp4", ParentID: "folder"}, Path: []pan.Directory{{ID: "root"}, {ID: "folder"}}}
	f.files["conflict"] = pan.FileInfo{File: pan.File{ID: "conflict", Name: "ABC-123-cd2.mp4", ParentID: "folder"}}
	plan := &download.Transfer{Files: []download.TransferFile{
		{ID: "video", Name: "ABC-123-cd1.mp4", OriginalName: "original.mp4", ParentID: "folder"},
		{ID: "second", Name: "ABC-123-cd2.mp4", OriginalName: "part2.mp4", ParentID: "folder"},
	}}
	if err := organize(t.Context(), f, plan); err == nil { t.Fatal("second file collision was ignored") }
	if f.renames != 0 || f.moves != 0 || f.deletes != 0 { t.Fatalf("preflight mutated files: %+v", f) }
}

func TestOrganizeDoesNotUndoUserChanges(t *testing.T) {
	for _, change := range []string{"name", "parent"} {
		t.Run(change, func(t *testing.T) {
			f := newFilesFixture()
			v := f.files["video"]
			if change == "name" { v.Name = "my-custom-name.mp4" } else { v.ParentID = "other-folder" }
			f.files["video"] = v
			plan := &download.Transfer{Files: []download.TransferFile{{ID: "video", Name: "ABC-123.mp4", OriginalName: "original.mp4", ParentID: "folder"}}}
			if err := organize(t.Context(), f, plan); err == nil { t.Fatal("manual change was overwritten") }
			if f.renames != 0 || f.moves != 0 || f.deletes != 0 { t.Fatal("unexpected mutation") }
		})
	}
}

func TestOrganizeDetectsDuplicateDestinationNames(t *testing.T) {
	f := newFilesFixture()
	v := f.files["video"]
	v.ParentID, v.Name = "root", "ABC-123.mp4"
	f.files["video"] = v
	f.files["duplicate"] = pan.FileInfo{File: pan.File{ID: "duplicate", Name: "abc-123.MP4", ParentID: "root"}}
	if err := organize(t.Context(), f, &download.Transfer{Files: []download.TransferFile{{ID: "video", Name: "ABC-123.mp4"}}}); err == nil { t.Fatal("duplicate name hidden by matching file ID") }
}

func TestSubtitleInSeparateDirectoryUsesUniqueBasename(t *testing.T) {
	files:=[]locatedFile{
		{File:pan.File{ID:"v",Name:"original.mp4",Size:domain.MinVideoSize},location:"Video/original.mp4"},
		{File:pan.File{ID:"s",Name:"original.zh.IDX"},location:"Subs/original.zh.IDX"},
		{File:pan.File{ID:"b",Name:"original.zh.SUB"},location:"Subs/original.zh.SUB"},
	}
	plan,err:=planFiles("ABC-123",files)
	if err!=nil || len(plan)!=3 || plan[1].Name!="ABC-123.zh.idx" || plan[2].Name!="ABC-123.zh.sub"{t.Fatalf("separate subtitles: %+v %v",plan,err)}
	files=append(files,locatedFile{File:pan.File{ID:"other",Name:"original.mkv",Size:domain.MinVideoSize},location:"Other/original.mkv"})
	if _,err:=planFiles("ABC-123",files);err==nil{t.Fatal("ambiguous subtitle was assigned to an arbitrary video")}
}
