package scan

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ppxb/miyabi/internal/ent/movie"
	"github.com/ppxb/miyabi/internal/ent/task"
	"github.com/ppxb/miyabi/internal/export"
	"github.com/ppxb/miyabi/internal/nfo"
	"github.com/ppxb/miyabi/internal/pan"
)

func TestLocalNFOUpdatesFailedLegacyMetadataAndPreservesKnownIdentity(t *testing.T) {
	scanner := localScannerFixture(t)
	ctx, root := t.Context(), t.TempDir()
	film := scanner.db.Movie.Create().SetCode("TEST-001").SetJavdbID("known-javdb").SetScrapeStatus(movie.ScrapeStatusFailed).SaveX(ctx)
	writeLocalFile(t, root, "TEST-001.mp4", []byte("video"))
	writeLocalFile(t, root, "poster.jpg", testJPEG(t))
	for _, title := range []string{"external title", "updated external title"} {
		body, err := nfo.Encode(nfo.Movie{Code: film.Code, Title: title, Genres: []string{"Drama"}, Actors: []nfo.Actor{{Name: "Actor"}}})
		if err != nil {
			t.Fatal(err)
		}
		writeLocalFile(t, root, "TEST-001.nfo", body)
		result, err := scanner.Scan(ctx, root)
		if err != nil || result.NFORead != 1 || result.Movies != 1 {
			t.Fatalf("import=%+v, %v", result, err)
		}
		got := scanner.db.Movie.Query().Where(movie.IDEQ(film.ID)).WithActors().WithTags().OnlyX(ctx)
		if got.Title != title || got.JavdbID == nil || *got.JavdbID != "known-javdb" || got.Metadata.JavDBID() != "known-javdb" || got.ScrapeStatus != movie.ScrapeStatusDone {
			t.Fatalf("external NFO was not applied: %+v", got)
		}
		if len(got.Edges.Actors) != 1 || len(got.Edges.Tags) != 1 || got.Edges.Tags[0].Name != "Drama" || got.Poster == nil {
			t.Fatal("NFO entities or local artwork were lost")
		}
		if _, err := scanner.images.ReadURL(*got.Poster); err != nil {
			t.Fatal(err)
		}
		if saved, err := os.ReadFile(filepath.Join(root, "TEST-001.nfo")); err != nil || !bytes.Equal(saved, body) {
			t.Fatal("import rewrote the external NFO")
		}
	}
	if scanner.db.File.Query().CountX(ctx) != 1 || scanner.db.Movie.Query().CountX(ctx) != 1 {
		t.Fatal("rescan duplicated the movie or video")
	}
}

func TestCloudScanImportsExistingNFOWithoutScraping(t *testing.T) {
	run := reconcileFixture(t, 0, 1)
	ctx := t.Context()
	film := run.scanner.db.Movie.Query().OnlyX(ctx)
	film.Update().SetJavdbID("known-javdb").ExecX(ctx)
	body, err := nfo.Encode(nfo.Movie{Code: film.Code, Title: "external cloud title", Genres: []string{"Drama"}, Actors: []nfo.Actor{{Name: "Actor"}}})
	if err != nil {
		t.Fatal(err)
	}
	session := &nfoStubSession{bodies: map[string][]byte{"nfo-pick": body}}
	videos := []Video{
		{File: pan.File{ID: "0", Name: film.Code + "-CD1.mp4", ParentID: "folder", Size: 1 << 30}, Code: film.Code},
		{File: pan.File{ID: "part2", Name: film.Code + "-CD2.mp4", ParentID: "folder", Size: 1 << 30}, Code: film.Code},
	}
	if err := ResolveNFOCodes(ctx, session, []pan.File{{ID: "nfo", Name: film.Code + ".nfo", PickCode: "nfo-pick"}}, videos); err != nil {
		t.Fatal(err)
	}
	if session.reads != 1 {
		t.Fatalf("read NFO %d times for one movie", session.reads)
	}
	if err := ProcessScanPage(ctx, run.scanner.db, run.taskID, "/Movies", videos, run.payload, nil, run.scanner.tasksSvc); err != nil {
		t.Fatal(err)
	}
	if err := run.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	got := run.scanner.db.Movie.Query().Where(movie.IDEQ(film.ID)).WithActors().WithTags().OnlyX(ctx)
	if got.Title != "external cloud title" || got.JavdbID == nil || *got.JavdbID != "known-javdb" || len(got.Edges.Actors) != 1 || len(got.Edges.Tags) != 1 {
		t.Fatalf("NFO was discarded: %+v", got)
	}
	if run.scanner.db.Task.Query().Where(task.TypeEQ("scrape")).CountX(ctx) != 0 {
		t.Fatal("NFO import scheduled scraping")
	}
	dir := export.EmbyMovieDir(run.scanner.exportMgr.Config().EmbyDir, film.Code)
	for _, name := range []string{film.Code + "-cd1.strm", film.Code + "-cd2.strm"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMalformedOptionalNFODoesNotBlockFilenameIndexing(t *testing.T) {
	for _, body := range []string{"invalid XML", "<movie><num>TEST-001</num><premiered>invalid</premiered></movie>"} {
		session := &nfoStubSession{bodies: map[string][]byte{"nfo-pick": []byte(body)}}
		videos := []Video{{File: pan.File{ID: "video", Name: "TEST-001.mp4", Size: 1 << 30}, Code: "TEST-001"}}
		if err := ResolveNFOCodes(t.Context(), session, []pan.File{{ID: "nfo", Name: "TEST-001.nfo", PickCode: "nfo-pick"}}, videos); err != nil {
			t.Fatal(err)
		}
		if videos[0].Code != "TEST-001" || videos[0].Document != nil {
			t.Fatal("malformed optional NFO changed video identification")
		}
	}
}
