package app

import (
	"testing"

	"github.com/ppxb/miyabi/internal/domain"
	"github.com/ppxb/miyabi/internal/ent/task"
	"github.com/ppxb/miyabi/internal/tasks"
)

func TestTaskViewsHideAndRejectLegacyScrapingWorkflows(t *testing.T) {
	views, fix := snapshotFixture(t)
	ctx := t.Context()
	var hidden []int
	for _, input := range []domain.ScanPayload{
		{Source: fix.Payload.Source, Rebuild: true},
		{Source: fix.Payload.Source, MovieID: 42, Code: "IPX-123"},
	} {
		body, err := tasks.EncodePayload(input)
		if err != nil {
			t.Fatal(err)
		}
		row := fix.DB.Task.Create().SetType(tasks.KindScan.String()).SetPayload(body).SetStatus(task.StatusFailed).SaveX(ctx)
		hidden = append(hidden, row.ID)
	}
	if err := fix.Tasks.RetireScrapingTasks(ctx); err != nil {
		t.Fatal(err)
	}
	items, err := views.List(ctx)
	if err != nil || len(items) != 1 || items[0].ID != fix.Queued.ID {
		t.Fatalf("legacy scraping remains visible: %+v, %v", items, err)
	}
	for _, id := range hidden {
		if _, err := views.Retry(ctx, id); !domain.IsKind(err, domain.KindInvalid) {
			t.Fatalf("legacy workflow %d accepted a retry: %v", id, err)
		}
	}
}
