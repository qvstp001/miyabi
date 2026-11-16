package scan

import (
	"context"

	"github.com/ppxb/miyabi/internal/codeid"
	"github.com/ppxb/miyabi/internal/ent"
	"github.com/ppxb/miyabi/internal/ent/movie"
	"github.com/ppxb/miyabi/internal/library/scrape"
	"github.com/ppxb/miyabi/internal/nfo"
)

// Import sidecars without discarding a known JavDB identity when an external
// tool only supplies a catalogue number. Source files are never rewritten.
func importMovieNFO(ctx context.Context, tx *ent.Tx, record *ent.Movie, doc nfo.Movie) (*ent.Movie, error) {
	if doc.JavDBID() == "" && record.JavdbID != nil && codeid.IsEquivalent(doc.Code, record.Code) {
		doc.IDs = append(append([]nfo.UniqueID(nil), doc.IDs...), nfo.UniqueID{Type: "javdb", Value: *record.JavdbID})
	}
	// Ordinary Kodi NFOs often use genre without tag.
	names := make(map[string]bool, len(doc.Tags))
	for _, tag := range doc.Tags {
		names[tag.Name] = true
	}
	doc.Tags = append([]nfo.Tag(nil), doc.Tags...)
	for _, genre := range doc.Genres {
		if genre != "" && !names[genre] {
			doc.Tags = append(doc.Tags, nfo.Tag{Name: genre})
			names[genre] = true
		}
	}
	if err := scrape.SaveMovieMetadata(ctx, tx, record.ID, doc); err != nil {
		return nil, err
	}
	return tx.Movie.UpdateOneID(record.ID).SetScrapeStatus(movie.ScrapeStatusDone).Save(ctx)
}
