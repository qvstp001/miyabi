package tasks

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ppxb/miyabi/internal/database"
	"github.com/ppxb/miyabi/internal/ent/task"
)

func TestRetireScrapingTasksPreservesScansAndCheckpoints(t *testing.T) {
	ctx := t.Context()
	store, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	service := NewService(store.Client, NewRegistry())
	for _, tc := range []struct {
		kind   Kind
		body   string
		retire bool
	}{
		{KindScrape, `{"checkpoint":"saved"}`, true},
		{KindScrape, `{"rebuild":true,"document":{"title":"saved"}}`, true},
		{KindScan, `{"rebuild":true,"source":{"account_id":"100"}}`, true},
		{KindScan, `{"movie_id":42,"source":{"account_id":"100"}}`, true},
		{KindScan, `{"rebuild":false,"scan":{"stage":"scanning"}}`, false},
		{KindScan, `{"offline_task_id":7,"target_id":"10"}`, false},
		{KindSubscriptionBatch, `{"checkpoint":"saved"}`, false},
	} {
		for _, status := range []task.Status{task.StatusQueued, task.StatusRunning, task.StatusFailed} {
			record := store.Client.Task.Create().SetType(tc.kind.String()).SetPayload(json.RawMessage(tc.body)).
				SetStatus(status).SetError("old failure").SetRetryAt(time.Now().Add(time.Minute)).SaveX(ctx)
			for range 2 {
				if err := service.RetireScrapingTasks(ctx); err != nil {
					t.Fatal(err)
				}
			}
			got := store.Client.Task.GetX(ctx, record.ID)
			if string(got.Payload) != tc.body {
				t.Fatal("retirement changed a checkpoint")
			}
			if tc.retire {
				if got.Status != task.StatusDone || got.Error != nil || got.RetryAt != nil || got.Progress != 100 {
					t.Fatalf("scraping task remains active: %+v", got)
				}
			} else if got.Status != status || got.Error == nil || got.RetryAt == nil {
				t.Fatalf("unrelated work changed: %+v", got)
			}
		}
	}
}
