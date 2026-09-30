package datetime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

type DateTime struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&DateTime{})
}

func (t DateTime) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &DateTime{agent: agent, enabled: true}, nil
}

func (t *DateTime) Enabled() bool { return t.enabled }
func (t *DateTime) Toggle()       { t.enabled = !t.enabled }

func (t *DateTime) Name() string { return "dateTime" }
func (t *DateTime) Path() string { return "internal" }
func (t *DateTime) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "alwaysAllow", Subagents: "allow"}
}
func (t *DateTime) Description() string {
	return `Returns the current date, time, and timezone, or a relative date/time. Input is an optional JSON object with amount, unit, and direction. Omit all fields for the current date/time. For a relative value, provide a positive integer amount, a unit (second, minute, hour, day, week, month, or year), and a direction (before or after), for example {"amount":2,"unit":"week","direction":"after"}. Quoted integer amounts such as "2" are also accepted. Calendar units preserve the local time and timezone.`
}

type dateTimeInputT struct {
	Amount    dateTimeAmount `json:"amount,omitempty"`
	Unit      string         `json:"unit,omitempty"`
	Direction string         `json:"direction,omitempty"`
}

type dateTimeAmount int

func (a *dateTimeAmount) UnmarshalJSON(data []byte) error {
	var numeric int
	if err := json.Unmarshal(data, &numeric); err == nil {
		*a = dateTimeAmount(numeric)
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("amount must be an integer")
	}
	numeric, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return fmt.Errorf("amount must be an integer")
	}
	*a = dateTimeAmount(numeric)
	return nil
}

func (t *DateTime) InputType() reflect.Type { return reflect.TypeFor[dateTimeInputT]() }

func (t *DateTime) Call(ctx context.Context, input string) (response string, err error) {
	log.Println("[debug] ai tool: dateTime")

	return time.Now().String(), nil
}

func (t *DateTime) CallStructured(_ context.Context, input any) (string, error) {
	request, ok := input.(*dateTimeInputT)
	if !ok || request == nil {
		return "", fmt.Errorf("dateTime received an invalid structured input %T", input)
	}
	value, err := calculateDateTime(time.Now(), *request)
	if err != nil {
		return "ERROR: " + err.Error(), nil
	}
	return value.String(), nil
}

func calculateDateTime(now time.Time, request dateTimeInputT) (time.Time, error) {
	amount := int(request.Amount)
	unit := strings.ToLower(strings.TrimSpace(request.Unit))
	unit = strings.TrimSuffix(unit, "s")
	direction := strings.ToLower(strings.TrimSpace(request.Direction))
	if request.Amount == 0 && unit == "" && direction == "" {
		return now, nil
	}
	if amount <= 0 {
		return time.Time{}, fmt.Errorf("amount must be a positive integer")
	}
	if amount > 100000 {
		return time.Time{}, fmt.Errorf("amount cannot exceed 100000")
	}
	if direction != "before" && direction != "after" {
		return time.Time{}, fmt.Errorf("direction must be 'before' or 'after'")
	}

	sign := 1
	if direction == "before" {
		sign = -1
	}

	switch unit {
	case "second":
		return now.Add(time.Duration(sign*amount) * time.Second), nil
	case "minute":
		return now.Add(time.Duration(sign*amount) * time.Minute), nil
	case "hour":
		return now.Add(time.Duration(sign*amount) * time.Hour), nil
	case "day":
		return now.AddDate(0, 0, sign*amount), nil
	case "week":
		return now.AddDate(0, 0, sign*amount*7), nil
	case "month":
		return now.AddDate(0, sign*amount, 0), nil
	case "year":
		return now.AddDate(sign*amount, 0, 0), nil
	default:
		return time.Time{}, fmt.Errorf("unit must be second, minute, hour, day, week, month, or year")
	}
}
