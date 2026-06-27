package actions

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/t0mer/raptor/internal/models"
)

func init() { register("conditions", newConditions) }

// conditions — evaluate input <operator> value and, if true, perform action
// ("stop" halts the chain, "skip" skips the next action). Doubles as a
// header-based auth gate.
type conditions struct {
	input    string
	operator string
	value    string
	action   string
}

func newConditions(a *models.Action) (runner, error) {
	op := strings.ToLower(strParam(a, "operator"))
	if op == "" {
		op = "equals"
	}
	action := strings.ToLower(strParam(a, "action"))
	if action == "" {
		action = "stop"
	}
	return &conditions{
		input:    strParam(a, "input"),
		operator: op,
		value:    strParam(a, "value"),
		action:   action,
	}, nil
}

func (c *conditions) run(_ context.Context, ec *ExecContext) error {
	in := ec.Interp(c.input)
	val := ec.Interp(c.value)
	met := evalCondition(in, c.operator, val)
	ec.Logf("condition %q %s %q => %v", in, c.operator, val, met)
	if !met {
		return nil
	}
	switch c.action {
	case "stop":
		ec.Stopped = true
	case "skip":
		ec.skipNext = true
	}
	return nil
}

func evalCondition(input, op, value string) bool {
	switch op {
	case "equals":
		return input == value
	case "not_equals":
		return input != value
	case "contains":
		return strings.Contains(input, value)
	case "not_contains":
		return !strings.Contains(input, value)
	case "matches":
		re, err := regexp.Compile(value)
		return err == nil && re.MatchString(input)
	case "exists":
		return input != ""
	case "not_exists":
		return input == ""
	case "gt", "lt":
		a, err1 := strconv.ParseFloat(input, 64)
		b, err2 := strconv.ParseFloat(value, 64)
		if err1 != nil || err2 != nil {
			return false
		}
		if op == "gt" {
			return a > b
		}
		return a < b
	default:
		return false
	}
}
