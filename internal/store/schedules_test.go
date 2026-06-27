package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/t0mer/raptor/internal/crypto"
	"github.com/t0mer/raptor/internal/models"
)

func TestScheduleCRUDWithEncryptedNotify(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	key, _ := crypto.LoadOrCreateKey(filepath.Join(t.TempDir(), "k"))
	c, _ := crypto.New(key)
	s.SetCipher(c)

	sc := &models.Schedule{
		UUID:      uuid.NewString(),
		Name:      "uptime",
		Cron:      "*/5 * * * *",
		TargetURL: "https://example.com/health",
		Method:    "GET",
		Keyword:   "ok",
		NotifyURL: "slack://token@channel",
		Enabled:   true,
		SSLDays:   14,
	}
	if err := s.CreateSchedule(ctx, sc); err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}

	// Verify the notify URL is stored encrypted (not plaintext).
	var stored string
	s.DB().QueryRowContext(ctx, `SELECT notify_url FROM schedules WHERE uuid = ?`, sc.UUID).Scan(&stored)
	if stored == "slack://token@channel" || stored == "" {
		t.Errorf("notify URL not encrypted at rest: %q", stored)
	}

	got, err := s.GetSchedule(ctx, sc.UUID)
	if err != nil {
		t.Fatalf("GetSchedule: %v", err)
	}
	if got.NotifyURL != "slack://token@channel" {
		t.Errorf("decrypted notify URL = %q", got.NotifyURL)
	}
	if got.Keyword != "ok" || !got.Enabled {
		t.Errorf("round-trip mismatch: %+v", got)
	}

	enabled, _ := s.ListEnabledSchedules(ctx)
	if len(enabled) != 1 {
		t.Errorf("ListEnabledSchedules = %d, want 1", len(enabled))
	}

	if err := s.RecordScheduleResult(ctx, sc.UUID, time.Now(), time.Now().Add(5*time.Minute), models.ScheduleAlert, "down"); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetSchedule(ctx, sc.UUID)
	if got.LastStatus != models.ScheduleAlert || got.LastRun == nil {
		t.Errorf("result not recorded: %+v", got)
	}

	run := &models.ScheduleRun{ID: uuid.NewString(), ScheduleID: sc.UUID, Status: "alert", StatusCode: 500, Message: "down"}
	if err := s.CreateScheduleRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	runs, _ := s.ListScheduleRuns(ctx, sc.UUID, 10)
	if len(runs) != 1 || runs[0].StatusCode != 500 {
		t.Errorf("ListScheduleRuns = %+v", runs)
	}

	if err := s.DeleteSchedule(ctx, sc.UUID); err != nil {
		t.Fatal(err)
	}
	runs, _ = s.ListScheduleRuns(ctx, sc.UUID, 10)
	if len(runs) != 0 {
		t.Error("schedule_runs not cascade-deleted")
	}
}
