package app

import (
	"bytes"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/ppxb/miyabi/internal/tasks"
)

func TestOrdinaryScanPreservesOutdatedPostersWithoutScraping(t *testing.T) {
	fixture := newPipelineFixture(t)
	ctx := t.Context()
	fixture.drive.addFile("101", "10", "ABP-123.mp4", 2<<30, []byte("video"))
	fixture.addCatalogueMovie(fixtureDetail())
	if _, err := fixture.library.StartRebuild(ctx); err != nil {
		t.Fatal(err)
	}
	fixture.runQueue(t)
	record := fixture.store.Client.Movie.Query().OnlyX(ctx)
	snapshot := *record.MetadataSnapshot
	snapshot.PosterVersion = 0
	record.Update().SetMetadataSnapshot(&snapshot).ExecX(ctx)
	posterPath := filepath.Join(fixture.embyDir, "ABP", "ABP-123", "poster.jpg")
	external := []byte("poster maintained by an external scraper")
	if err := os.WriteFile(posterPath, external, 0o644); err != nil {
		t.Fatal(err)
	}
	calls := maps.Clone(fixture.catalogue.calls)
	if _, err := fixture.library.StartScan(ctx); err != nil {
		t.Fatal(err)
	}
	jobs := fixture.runQueue(t)
	if len(jobs) != 1 || jobs[0].Type != tasks.KindScan {
		t.Fatalf("ordinary scan scheduled poster work: %+v", jobs)
	}
	if updated := fixture.store.Client.Movie.GetX(ctx, record.ID); updated.MetadataSnapshot.PosterVersion != 0 {
		t.Fatal("ordinary scan regenerated the poster")
	}
	if body, err := os.ReadFile(posterPath); err != nil || !bytes.Equal(body, external) {
		t.Fatalf("external poster changed: %v", err)
	}
	if !maps.Equal(calls, fixture.catalogue.calls) {
		t.Fatalf("ordinary scan contacted metadata or image sources: %v", fixture.catalogue.calls)
	}
}
