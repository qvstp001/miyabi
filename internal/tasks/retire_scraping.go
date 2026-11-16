package tasks

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/ppxb/miyabi/internal/ent"
	"github.com/ppxb/miyabi/internal/ent/task"
)

// RetireScrapingTasks runs before workers start. Preserve saved metadata and
// ordinary scans, but never resume legacy automatic or manual scraping jobs.
func (s *Service) RetireScrapingTasks(ctx context.Context) error {
	return ent.WithTx(ctx, s.queue.database, func(tx *ent.Tx) error {
		_, err := tx.Task.Update().Where(task.StatusNEQ(task.StatusDone),
			task.Or(task.TypeEQ(KindScrape.String()), task.And(task.TypeEQ(KindScan.String()), func(selector *sql.Selector) {
				selector.Where(sql.Or(
					sqljson.ValueEQ(task.FieldPayload, true, sqljson.Path("rebuild")),
					sqljson.HasKey(task.FieldPayload, sqljson.Path("movie_id")),
				))
			})),
		).SetStatus(task.StatusDone).SetProgress(100).ClearError().ClearRetryAt().Save(ctx)
		return err
	})
}
