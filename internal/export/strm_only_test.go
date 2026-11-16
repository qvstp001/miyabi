package export

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ppxb/miyabi/internal/pan"
)

func TestWriteSTRMPreservesExternalSidecarsAndIsIdempotent(t *testing.T) {
	cfg := Config{EmbyDir: t.TempDir(), PublicURL: "http://localhost", STRMToken: "token"}
	dir := EmbyMovieDir(cfg.EmbyDir, "ABP-123")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	external := []byte("maintained by another scraper")
	for _, name := range []string{"ABP-123.nfo", "poster.jpg", "fanart.jpg", "ABP-123.zh.srt"} {
		if err := os.WriteFile(filepath.Join(dir, name), external, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	videos := []pan.File{{ID: "two", Name: "ABP-123-CD2.mp4"}, {ID: "one", Name: "ABP-123-CD1.mp4"}}
	changed, err := WriteSTRM(cfg, "ABP-123", videos)
	if err != nil || !changed {
		t.Fatalf("first export: changed=%t err=%v", changed, err)
	}
	for i, id := range []string{"one", "two"} {
		name := []string{"ABP-123-cd1.strm", "ABP-123-cd2.strm"}[i]
		file := filepath.Join(dir, name)
		if body, err := os.ReadFile(file); err != nil || !bytes.Equal(body, STRMContent(cfg.PublicURL, id, cfg.STRMToken)) {
			t.Fatalf("wrong part order: %s %v", body, err)
		}
		stamp := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		if err := os.Chtimes(file, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	if changed, err := WriteSTRM(cfg, "ABP-123", videos); err != nil || changed {
		t.Fatalf("unchanged export: changed=%t err=%v", changed, err)
	}
	for _, name := range []string{"ABP-123.nfo", "poster.jpg", "fanart.jpg", "ABP-123.zh.srt"} {
		if body, err := os.ReadFile(filepath.Join(dir, name)); err != nil || !bytes.Equal(body, external) {
			t.Fatalf("external sidecar overwritten: %s %v", name, err)
		}
	}
}

func TestWriteSTRMDoesNotCreateMetadataForNewMovies(t *testing.T) {
	cfg := Config{EmbyDir: t.TempDir(), PublicURL: "http://localhost"}
	if _, err := WriteSTRM(cfg, "ABP-123", []pan.File{{ID: "video", Name: "ABP-123.mp4"}}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(EmbyMovieDir(cfg.EmbyDir, "ABP-123"))
	if err != nil || len(entries) != 1 || entries[0].Name() != "ABP-123.strm" {
		t.Fatalf("unexpected generated sidecars: %v %v", entries, err)
	}
}

func TestWriteSTRMUpdatesPartsAndAddressesWithoutRemovingOtherStreams(t *testing.T) {
	cfg := Config{EmbyDir: t.TempDir(), PublicURL: "http://old-host", STRMToken: "old-token"}
	dir := EmbyMovieDir(cfg.EmbyDir, "ABP-123")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	external := []byte("https://external.example/video")
	for _, name := range []string{"ABP-1234.strm", "ABP-123-trailer.strm"} {
		if err := os.WriteFile(filepath.Join(dir, name), external, 0644); err != nil {
			t.Fatal(err)
		}
	}
	videos := []pan.File{{ID: "ten", Name: "ABP-123-CD10.mp4"}, {ID: "two", Name: "ABP-123-CD2.mp4"}, {ID: "one", Name: "ABP-123-CD1.mp4"}}
	if _, err := WriteSTRM(cfg, "ABP-123", videos); err != nil {
		t.Fatal(err)
	}
	for i, id := range []string{"one", "two", "ten"} {
		name := []string{"ABP-123-cd1.strm", "ABP-123-cd2.strm", "ABP-123-cd3.strm"}[i]
		if body, err := os.ReadFile(filepath.Join(dir, name)); err != nil || !bytes.Equal(body, STRMContent(cfg.PublicURL, id, cfg.STRMToken)) {
			t.Fatalf("wrong part ordering: %s, %v", body, err)
		}
	}
	cfg.PublicURL, cfg.STRMToken = "http://new-host", "new token & value"
	if _, err := WriteSTRM(cfg, "ABP-123", []pan.File{{ID: "one", Name: "ABP-123.mp4"}}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ABP-123-cd1.strm", "ABP-123-cd2.strm", "ABP-123-cd3.strm"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("obsolete part remains: %s", name)
		}
	}
	want := []byte("http://new-host/api/strm/play/one?token=new+token+%26+value\n")
	if body, err := os.ReadFile(filepath.Join(dir, "ABP-123.strm")); err != nil || !bytes.Equal(body, want) {
		t.Fatalf("address or token was not refreshed: %s, %v", body, err)
	}
	for _, name := range []string{"ABP-1234.strm", "ABP-123-trailer.strm"} {
		if body, err := os.ReadFile(filepath.Join(dir, name)); err != nil || !bytes.Equal(body, external) {
			t.Fatalf("another stream was removed: %s", name)
		}
	}
}
