package actions

import (
	"context"

	"github.com/google/uuid"

	"github.com/t0mer/raptor/internal/models"
	"github.com/t0mer/raptor/internal/store"
)

// Service ties the engine to the store: it loads a token's action chain, runs it
// for a request, and persists the per-action run log.
type Service struct {
	engine *Engine
	store  *store.Store
}

// NewService builds a Service.
func NewService(engine *Engine, st *store.Store) *Service {
	return &Service{engine: engine, store: st}
}

// Execute loads and runs the token's action chain against req, returning the
// resulting context and per-action results. Nothing is persisted.
func (s *Service) Execute(ctx context.Context, req *models.Request, tok *models.Token) (*ExecContext, []RunResult, error) {
	acts, err := s.store.ListActions(ctx, tok.UUID)
	if err != nil {
		return nil, nil, err
	}
	ec := s.engine.NewContext(req, tok)
	results := s.engine.Execute(ctx, acts, ec)
	return ec, results, nil
}

// ExecuteDefs runs an explicit set of action definitions (used by test-action),
// persisting nothing.
func (s *Service) ExecuteDefs(ctx context.Context, defs []*models.Action, req *models.Request, tok *models.Token) (*ExecContext, []RunResult) {
	ec := s.engine.NewContext(req, tok)
	return ec, s.engine.Execute(ctx, defs, ec)
}

// SaveRuns persists the action run log for a stored request.
func (s *Service) SaveRuns(ctx context.Context, requestID string, results []RunResult) error {
	for _, r := range results {
		if r.Skipped {
			continue
		}
		run := &models.ActionRun{
			ID:         uuid.NewString(),
			RequestID:  requestID,
			ActionID:   r.Action.UUID,
			ActionType: r.Action.Type,
			ActionName: r.Action.Name,
			Position:   r.Action.Position,
			Output:     r.Output,
			Error:      errString(r.Err),
		}
		if err := s.store.CreateActionRun(ctx, run); err != nil {
			return err
		}
	}
	return nil
}

// Summarise reduces results to output/error maps keyed by action name (or type),
// for storage on the request itself.
func Summarise(results []RunResult) (output, errs map[string]any) {
	output = map[string]any{}
	errs = map[string]any{}
	for _, r := range results {
		if r.Skipped {
			continue
		}
		key := r.Action.Name
		if key == "" {
			key = r.Action.Type
		}
		if r.Output != "" {
			output[key] = r.Output
		}
		if r.Err != nil {
			errs[key] = r.Err.Error()
		}
	}
	return output, errs
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
