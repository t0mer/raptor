// Package actions implements Raptor's Custom Actions engine: an ordered chain of
// actions that run per captured request, sharing variables, able to gate
// execution, build the response, extract data and call out over HTTP or run a
// script. Action types register themselves with the package registry.
package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/t0mer/raptor/internal/models"
)

// Response is the HTTP response the action chain builds for a captured request.
type Response struct {
	Status      int
	Content     string
	ContentType string
	Headers     map[string]string
	// Set reports whether an action explicitly produced/overrode the response.
	Set bool
}

// ExecContext is the mutable state threaded through an action chain for a single
// request. Variables set by one action are visible to later actions.
type ExecContext struct {
	Request  *models.Request
	Vars     map[string]string
	Response *Response

	// Control flags.
	DontSave bool // do not persist this request
	Stopped  bool // halt the chain

	skipNext  bool
	out       *strings.Builder
	engineRef *Engine
}

// Logf appends a line to the current action's captured output.
func (ec *ExecContext) Logf(format string, args ...any) {
	if ec.out == nil {
		return
	}
	fmt.Fprintf(ec.out, format, args...)
	ec.out.WriteByte('\n')
}

// SetVar stores a variable.
func (ec *ExecContext) SetVar(name, value string) { ec.Vars[name] = value }

// GetVar resolves a variable name, including request.* accessors.
func (ec *ExecContext) GetVar(name string) string { return resolveVar(ec, name) }

// Interp replaces $name$ references in s with their resolved values.
func (ec *ExecContext) Interp(s string) string { return interpolate(ec, s) }

// runner is the executable form of an action, built from its parameters.
type runner interface {
	run(ctx context.Context, ec *ExecContext) error
}

// factory builds a runner from a stored action definition.
type factory func(a *models.Action) (runner, error)

var registry = map[string]factory{}

// register adds an action type to the registry (called from init()).
func register(typ string, f factory) { registry[typ] = f }

// KnownTypes returns the registered action type names.
func KnownTypes() []string {
	out := make([]string, 0, len(registry))
	for t := range registry {
		out = append(out, t)
	}
	return out
}

// param helpers for reading typed values out of an action's parameters map.

func strParam(a *models.Action, key string) string {
	if v, ok := a.Parameters[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func intParam(a *models.Action, key string, def int) int {
	if v, ok := a.Parameters[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return def
}

func boolParam(a *models.Action, key string) bool {
	if v, ok := a.Parameters[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
