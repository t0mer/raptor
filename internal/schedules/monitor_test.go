package schedules

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/t0mer/raptor/internal/actions"
	"github.com/t0mer/raptor/internal/models"
	"github.com/t0mer/raptor/internal/store"
)

func newRunner(t *testing.T) (*Runner, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	svc := actions.NewService(actions.New(actions.WithSSRFLists(nil, nil, true)), st)
	return New(st, svc), st
}

func mkSchedule(t *testing.T, st *store.Store, sc *models.Schedule) {
	t.Helper()
	sc.UUID = uuid.NewString()
	sc.Cron = "*/5 * * * *"
	sc.Enabled = true
	if err := st.CreateSchedule(context.Background(), sc); err != nil {
		t.Fatalf("create schedule: %v", err)
	}
}

func TestHTTPCheckOK(t *testing.T) {
	r, st := newRunner(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("all systems ok"))
	}))
	defer srv.Close()

	sc := &models.Schedule{Name: "uptime", TargetURL: srv.URL, Method: "GET", Keyword: "ok"}
	mkSchedule(t, st, sc)
	run := r.Run(context.Background(), sc)
	if run.Status != models.ScheduleOK {
		t.Errorf("status = %s (%s), want ok", run.Status, run.Message)
	}
	// next_run is computed and persisted.
	got, _ := st.GetSchedule(context.Background(), sc.UUID)
	if got.NextRun == nil || got.LastStatus != models.ScheduleOK {
		t.Errorf("schedule result not recorded: %+v", got)
	}
}

func TestHTTPCheckKeywordAlert(t *testing.T) {
	r, st := newRunner(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("service degraded"))
	}))
	defer srv.Close()

	sc := &models.Schedule{TargetURL: srv.URL, Keyword: "ok"}
	mkSchedule(t, st, sc)
	run := r.Run(context.Background(), sc)
	if run.Status != models.ScheduleAlert {
		t.Errorf("status = %s, want alert", run.Status)
	}
}

func TestHTTPCheckStatusAlert(t *testing.T) {
	r, st := newRunner(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	sc := &models.Schedule{TargetURL: srv.URL}
	mkSchedule(t, st, sc)
	run := r.Run(context.Background(), sc)
	if run.Status != models.ScheduleAlert || run.StatusCode != 500 {
		t.Errorf("status = %s code = %d, want alert/500", run.Status, run.StatusCode)
	}
}

func TestHTTPCheckUnreachable(t *testing.T) {
	r, st := newRunner(t)
	sc := &models.Schedule{TargetURL: "http://127.0.0.1:1/"}
	mkSchedule(t, st, sc)
	run := r.Run(context.Background(), sc)
	if run.Status != models.ScheduleAlert {
		t.Errorf("status = %s, want alert (unreachable)", run.Status)
	}
}

func TestRunActionsSchedule(t *testing.T) {
	r, st := newRunner(t)
	tok := &models.Token{UUID: uuid.NewString(), Premium: true, Actions: true}
	if err := st.CreateToken(context.Background(), tok); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateAction(context.Background(), &models.Action{
		UUID: uuid.NewString(), TokenID: tok.UUID, Type: "set_variable",
		Parameters: map[string]any{"name": "x", "value": "1"},
	}); err != nil {
		t.Fatal(err)
	}
	sc := &models.Schedule{Name: "cron actions", TokenID: tok.UUID, RunActions: true}
	mkSchedule(t, st, sc)
	run := r.Run(context.Background(), sc)
	if run.Status != models.ScheduleOK {
		t.Errorf("status = %s (%s), want ok", run.Status, run.Message)
	}
}
