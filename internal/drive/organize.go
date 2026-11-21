package drive

import (
	"context"
	"fmt"
	"slices"

	"github.com/ppxb/miyabi/internal/pan"
)

// FileClient is separate from read/playback clients so existing readers need no
// write permissions. The browser download worker is its only caller.
type FileClient interface {
	Rename(context.Context, string, string, string) error
	Move(context.Context, string, string, string) error
	Delete(context.Context, string, string) error
}

var _ FileClient = (*pan.Client)(nil)

type FileSession interface {
	Session
	Rename(context.Context, string, string) error
	Move(context.Context, string) error
	DeleteEmpty(context.Context, string) error
}

func (d *Drive) OpenFiles(ctx context.Context) (FileSession, error) {
	session, err := d.Open(ctx)
	if err != nil { return nil, err }
	if _, ok := d.client.(FileClient); !ok { return nil, fmt.Errorf("115 client does not support file organization") }
	return session.(*sourceSession), nil
}

func (s *sourceSession) mutate(ctx context.Context, fn func(FileClient, string) error) error {
	state, err := s.drive.sourceState(s.source, s.version)
	if err != nil { return err }
	_, err = withPanSourceToken(ctx, s.drive, state, func(token string) (struct{}, error) {
		return struct{}{}, s.WithSource(ctx,func() error { return fn(s.drive.client.(FileClient),token) })
	})
	return err
}

func (s *sourceSession) Rename(ctx context.Context, id, name string) error {
	if _, err := SourceInfo(ctx, s, id); err != nil { return err }
	return s.mutate(ctx, func(c FileClient, token string) error { return c.Rename(ctx, token, id, name) })
}

func (s *sourceSession) Move(ctx context.Context, id string) error {
	if _, err := SourceInfo(ctx, s, id); err != nil { return err }
	return s.mutate(ctx, func(c FileClient, token string) error { return c.Move(ctx, token, id, s.source.Directory.ID) })
}

func (s *sourceSession) DeleteEmpty(ctx context.Context, id string) error {
	if id == s.source.Directory.ID { return fmt.Errorf("cannot remove the download destination") }
	// Keep source changes excluded through the emptiness check and deletion.
	// Direct client reads let the outer mutation refresh an expired token after
	// releasing the source lock, rather than refreshing recursively under it.
	return s.mutate(ctx,func(c FileClient,token string)error{
		info,err:=s.drive.client.Info(ctx,token,id)
		if err!=nil{return err}
		if !WithinSource(info,s.source){return ErrSourceChanged}
		if !info.IsDirectory{return fmt.Errorf("cannot remove a file as an empty directory")}
		page,err:=s.drive.client.List(ctx,token,id,0,1)
		if err!=nil{return err}
		if !slices.ContainsFunc(page.Path,func(d pan.Directory)bool{return d.ID==s.source.Directory.ID}){return ErrSourceChanged}
		if len(page.Files)>0{return nil}
		if page.Total!=0 || page.HasMore{return ErrDirectoryIncomplete}
		return c.Delete(ctx,token,id)
	})
}
