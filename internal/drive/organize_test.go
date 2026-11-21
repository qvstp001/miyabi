package drive

import (
	"context"
	"testing"

	"github.com/ppxb/miyabi/internal/pan"
)

type organizingClient struct {
	*stubClient
	deleted []string
}

func (*organizingClient) Rename(context.Context,string,string,string) error { return nil }
func (*organizingClient) Move(context.Context,string,string,string) error { return nil }
func (c *organizingClient) Delete(_ context.Context,_ string,id string) error { c.deleted=append(c.deleted,id);return nil }

func TestDeleteEmptyOnlyRemovesVerifiedEmptyDirectories(t *testing.T) {
	for _,kind:=range []string{"empty","non-empty","file","outside","incomplete","root"}{
		t.Run(kind,func(t *testing.T){
			d,stub:=mountedTestDrive(t)
			client:=&organizingClient{stubClient:stub}
			d.client=client
			stub.info=func(context.Context,string,string)(pan.FileInfo,error){
				info:=pan.FileInfo{File:pan.File{ID:"folder",ParentID:testSource.Directory.ID,IsDirectory:kind!="file"},Path:[]pan.Directory{{ID:testSource.Directory.ID}}}
				if kind=="outside"{info.Path=[]pan.Directory{{ID:"other-root"}}}
				return info,nil
			}
			stub.list=func(context.Context,string,string,int,int)(pan.FilePage,error){
				page:=pan.FilePage{Path:[]pan.Directory{{ID:testSource.Directory.ID},{ID:"folder"}}}
				if kind=="non-empty"{page.Files=[]pan.File{{ID:"keep",ParentID:"folder",Name:"keep.txt"}};page.Total=1}
				if kind=="incomplete"{page.Total=1;page.HasMore=true}
				return page,nil
			}
			sess,err:=d.OpenFiles(t.Context())
			if err!=nil{t.Fatal(err)}
			id:="folder"
			if kind=="root"{id=testSource.Directory.ID}
			err=sess.DeleteEmpty(t.Context(),id)
			if kind=="empty"{
				if err!=nil || len(client.deleted)!=1 || client.deleted[0]!="folder"{t.Fatalf("empty directory not removed: %v %v",client.deleted,err)}
			}else{
				if len(client.deleted)!=0{t.Fatalf("unsafe deletion for %s",kind)}
				if kind!="non-empty" && err==nil{t.Fatalf("invalid %s was accepted",kind)}
			}
		})
	}
}
