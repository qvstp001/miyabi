package transfer

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/ppxb/miyabi/internal/codeid"
	"github.com/ppxb/miyabi/internal/domain"
	"github.com/ppxb/miyabi/internal/domain/download"
	"github.com/ppxb/miyabi/internal/drive"
	"github.com/ppxb/miyabi/internal/pan"
)

type locatedFile struct { pan.File; location string }

func makePlan(ctx context.Context,sess drive.Session,root,code string) ([]download.TransferFile,[]string,error) {
	if root == sess.Source().Directory.ID { return nil,nil,fmt.Errorf("115 未提供独立的下载资源位置，不能整理整个目标目录") }
	var files []locatedFile
	var dirs []string
	seen := make(map[string]bool)
	var walk func(string,string) error
	walk = func(id,relative string) error {
		if id == sess.Source().Directory.ID { return fmt.Errorf("下载目录遍历返回目标根目录，已停止整理") }
		if seen[id] { return fmt.Errorf("115 目录包含重复文件 ID") }
		seen[id] = true
		info,err := drive.SourceInfo(ctx,sess,id)
		if err != nil { return err }
		if !info.IsDirectory { files = append(files,locatedFile{File:info.File,location:path.Join(relative,info.Name)}); return nil }
		children,err := drive.DirectoryEntries(ctx,sess,id)
		if err != nil { return err }
		for _,f := range children {
			if f.ParentID == "" { f.ParentID = id }
			if f.ParentID != id { return fmt.Errorf("115 返回了其他目录的文件，已停止整理") }
			if f.IsDirectory { if err := walk(f.ID,path.Join(relative,f.Name)); err != nil { return err }
			} else { files = append(files,locatedFile{File:f,location:path.Join(relative,f.Name)}) }
		}
		dirs = append(dirs,id) // Children precede parents for empty-folder cleanup.
		return nil
	}
	if err := walk(root,""); err != nil { return nil,nil,err }
	plan,err := planFiles(code,files)
	return plan,dirs,err
}

func planFiles(code string,files []locatedFile) ([]download.TransferFile,error) {
	var videos []locatedFile
	for _,f := range files {
		if domain.IsVideo(f.Name) && f.Size >= domain.MinVideoSize {
			if found,ok:=codeid.Parse(f.Name);ok && !codeid.IsEquivalent(found,code) && !slices.Contains([]string{"PART","CD","DISC","VIDEO","MOVIE"},codeid.Prefix(found)) {
				return nil,fmt.Errorf("下载结果含其他番号 %s，已停止自动整理，请手动核对",found)
			}
			videos = append(videos,f)
		}
	}
	if len(videos) == 0 { return nil,fmt.Errorf("下载结果中没有大于等于 100 MiB 的视频，未移动或删除文件") }
	slices.SortFunc(videos,func(a,b locatedFile) int {
		if c:=naturalCompare(a.location,b.location);c!=0{return c}
		return strings.Compare(a.ID,b.ID)
	})
	result := make([]download.TransferFile,0,len(videos))
	names := make(map[string]bool)
	ids := make(map[string]bool)
	add := func(file locatedFile,name string) error {
		if file.ID == "" || ids[file.ID] { return fmt.Errorf("115 返回重复或缺失的文件 ID，已停止整理") }
		key := strings.ToLower(name)
		if names[key] { return fmt.Errorf("多个下载文件将重命名为 %s，请手动整理",name) }
		names[key] = true
		ids[file.ID] = true
		result = append(result,download.TransferFile{ID:file.ID,Name:name,OriginalName:file.Name,ParentID:file.ParentID})
		return nil
	}
	stems:=make([]string,len(videos))
	for i,v := range videos {
		stem := code
		if len(videos) > 1 { stem = fmt.Sprintf("%s-cd%d",code,i+1) }
		stems[i]=stem
		if err := add(v,stem+strings.ToLower(path.Ext(v.Name))); err != nil { return nil,err }
	}
	for _,f:=range files {
		if !isSubtitle(f.Name){continue}
		sub:=strings.ToLower(strings.TrimSuffix(f.location,path.Ext(f.location)))
		owner,suffix,ambiguous:=subtitleOwner(sub,videos,false)
		if owner<0 { owner,suffix,ambiguous=subtitleOwner(sub,videos,true) }
		if ambiguous { return nil,fmt.Errorf("字幕 %s 对应多个视频，请手动核对",f.Name) }
		if owner>=0 {
			if err:=add(f,stems[owner]+suffix+strings.ToLower(path.Ext(f.Name)));err!=nil{return nil,err}
		}
	}
	return result,nil
}

// Prefer a sibling video. A separate Subs folder is supported only when the
// basename identifies a unique video, never by directory length or list order.
func subtitleOwner(stem string,videos []locatedFile,basename bool) (int,string,bool) {
	if basename { stem=path.Base(stem) }
	owner,length,ambiguous:=-1,0,false
	for i,v:=range videos {
		original:=strings.ToLower(strings.TrimSuffix(v.location,path.Ext(v.location)))
		if basename { original=path.Base(original) }
		if stem!=original && !strings.HasPrefix(stem,original+"."){continue}
		if len(original)>length { owner,length,ambiguous=i,len(original),false } else if len(original)==length { ambiguous=true }
	}
	if owner<0 { return -1,"",false }
	return owner,stem[length:],ambiguous
}

func isSubtitle(name string) bool {
	if domain.IsSubtitle(name) { return true }
	switch strings.ToLower(path.Ext(name)) {
	case ".sub", ".idx", ".sup", ".smi": return true
	default: return false
	}
}

func naturalCompare(a,b string) int {
	a,b = strings.ToLower(a),strings.ToLower(b)
	for len(a)>0 && len(b)>0 {
		if a[0]>='0' && a[0]<='9' && b[0]>='0' && b[0]<='9' {
			i,j := 0,0
			for i<len(a) && a[i]>='0' && a[i]<='9' { i++ }
			for j<len(b) && b[j]>='0' && b[j]<='9' { j++ }
			x,y := strings.TrimLeft(a[:i],"0"),strings.TrimLeft(b[:j],"0")
			if len(x)!=len(y) { if len(x)<len(y) { return -1 }; return 1 }
			if c := strings.Compare(x,y); c!=0 { return c }
			a,b = a[i:],b[j:]
			continue
		}
		if a[0]!=b[0] { if a[0]<b[0] { return -1 }; return 1 }
		a,b = a[1:],b[1:]
	}
	return strings.Compare(a,b)
}

func organize(ctx context.Context,sess drive.FileSession,t *download.Transfer) error {
	destination := sess.Source().Directory.ID
	entries,err := drive.DirectoryEntries(ctx,sess,destination)
	if err != nil { return err }
	// Preflight every collision before the first mutation. Never replace an existing file.
	for _,f := range t.Files {
		for _,entry := range entries {
			if strings.EqualFold(entry.Name,f.Name) && entry.ID != f.ID { return fmt.Errorf("目标目录已存在 %s，未覆盖",f.Name) }
		}
		info,err := plannedFile(ctx,sess,f)
		if err != nil { return err }
		if info.Name != f.Name {
			peers,err := drive.DirectoryEntries(ctx,sess,info.ParentID)
			if err != nil { return err }
			for _,peer := range peers { if strings.EqualFold(peer.Name,f.Name) && peer.ID != f.ID { return fmt.Errorf("下载文件夹已有同名文件 %s",f.Name) } }
		}
	}
	for _,f := range t.Files {
		info,err := plannedFile(ctx,sess,f)
		if err != nil { return err }
		if info.Name!=f.Name {
			// Also prevent a collision within the original download folder.
			peers,err := drive.DirectoryEntries(ctx,sess,info.ParentID)
			if err != nil { return err }
			for _,peer := range peers { if strings.EqualFold(peer.Name,f.Name) && peer.ID!=f.ID { return fmt.Errorf("下载文件夹已有同名文件 %s",f.Name) } }
			if err := sess.Rename(ctx,f.ID,f.Name); err != nil { return err }
		}
		if info.ParentID!=destination { if err := sess.Move(ctx,f.ID); err != nil { return err } }
	}
	if err := verifyFiles(ctx,sess,t.Files); err != nil { return err }
	for _,id := range t.Directories {
		if err := sess.DeleteEmpty(ctx,id); err != nil && !errors.Is(err,pan.ErrNotFound) { return err }
	}
	return nil
}

func plannedFile(ctx context.Context,sess drive.Session,file download.TransferFile) (pan.FileInfo,error) {
	info,err := drive.SourceInfo(ctx,sess,file.ID)
	if err != nil { return pan.FileInfo{},err }
	if info.IsDirectory { return pan.FileInfo{},fmt.Errorf("计划中的视频或字幕已变成目录") }
	if file.ParentID != "" && info.ParentID != file.ParentID && info.ParentID != sess.Source().Directory.ID {
		return pan.FileInfo{},fmt.Errorf("文件 %s 已被移动到其他目录，请手动核对",file.Name)
	}
	if file.OriginalName != "" && info.Name != file.OriginalName && info.Name != file.Name {
		return pan.FileInfo{},fmt.Errorf("文件 %s 已被手动改名，请核对后重试",file.Name)
	}
	return info,nil
}

func verifyFiles(ctx context.Context,sess drive.Session,files []download.TransferFile) error {
	if len(files) == 0 { return fmt.Errorf("整理计划没有可推送的文件") }
	for _,file := range files {
		info,err := drive.SourceInfo(ctx,sess,file.ID)
		if err != nil { return err }
		if info.IsDirectory || info.Name != file.Name || info.ParentID != sess.Source().Directory.ID {
			return fmt.Errorf("文件 %s 尚未在目标目录就位，请核对后重试",file.Name)
		}
	}
	return nil
}
