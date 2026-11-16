package library

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ppxb/miyabi/internal/database"
	"github.com/ppxb/miyabi/internal/domain"
	"github.com/ppxb/miyabi/internal/export"
	"github.com/ppxb/miyabi/internal/tasks"
)

func TestPublicSyncImportsLocalNFOWithout115AndRefreshesLaterChanges(t *testing.T) {
	ctx, root := t.Context(), t.TempDir()
	store, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	queue := tasks.NewService(store.Client, tasks.NewRegistry())
	lib := New(store.Client, nil, queue, nil, Options{ExportManager: export.NewManager(export.Config{EmbyDir: root})})
	queue.Registry().Register(tasks.NewHandler(tasks.KindScan, lib.Scan, lib.Finished))
	if err := os.WriteFile(filepath.Join(root, "TEST-001.strm"), []byte("https://external.example/video"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"initial title", "changed after startup"} {
		if err := os.WriteFile(filepath.Join(root, "TEST-001.nfo"), []byte("<movie><title>" + title + "</title><num>TEST-001</num></movie>"), 0600); err != nil {
			t.Fatal(err)
		}
		info, err := lib.StartScan(ctx)
		if err != nil || info.Source.AccountID != domain.LocalAccountID {
			t.Fatalf("local sync=%+v, %v", info, err)
		}
		job, err := queue.Queue().Claim(ctx, []tasks.Kind{tasks.KindScan})
		if err != nil || job == nil {
			t.Fatalf("claim=%+v, %v", job, err)
		}
		if err := lib.Scan(ctx, *job); err != nil {
			t.Fatal(err)
		}
		if err := queue.Queue().Finish(ctx, job.ID, nil); err != nil {
			t.Fatal(err)
		}
		page, err := lib.Movies(ctx, 1, 20)
		if err != nil || page.Source != nil || page.LocalSource == nil || page.LocalSource.AccountID != domain.LocalAccountID || len(page.Movies) != 1 || page.Movies[0].Title != title {
			t.Fatalf("library=%+v, %v", page, err)
		}
	}
}
