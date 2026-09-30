package datetime

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDateTimeStructuredInputType(t *testing.T) {
	inputType := (&DateTime{}).InputType()
	if inputType != reflect.TypeOf(dateTimeInputT{}) {
		t.Fatalf("InputType() = %v, want dateTimeInputT", inputType)
	}
	for field, wantTag := range map[string]string{
		"Amount":    "amount,omitempty",
		"Unit":      "unit,omitempty",
		"Direction": "direction,omitempty",
	} {
		fieldInfo, ok := inputType.FieldByName(field)
		if !ok || fieldInfo.Tag.Get("json") != wantTag {
			t.Errorf("input field %s JSON tag = %q, want %q", field, fieldInfo.Tag.Get("json"), wantTag)
		}
	}
}

func TestDateTimeAmountAcceptsJSONNumberAndNumericString(t *testing.T) {
	for _, input := range []string{
		`{"amount":2,"unit":"week","direction":"after"}`,
		`{"amount":"2","unit":"week","direction":"after"}`,
	} {
		var request dateTimeInputT
		decoder := json.NewDecoder(strings.NewReader(input))
		decoder.UseNumber()
		if err := decoder.Decode(&request); err != nil {
			t.Errorf("Unmarshal(%s): %v", input, err)
			continue
		}
		if request.Amount != 2 || request.Unit != "week" || request.Direction != "after" {
			t.Errorf("decoded request from %s = %+v", input, request)
		}
	}
}

func TestCalculateDateTime(t *testing.T) {
	zone := time.FixedZone("test", 5*60*60)
	now := time.Date(2026, time.September, 29, 16, 0, 0, 0, zone)
	tests := []struct {
		name    string
		input   dateTimeInputT
		want    time.Time
		wantErr string
	}{
		{
			name: "current date when no offset is supplied",
			want: now,
		},
		{
			name:  "days before",
			input: dateTimeInputT{Amount: 2, Unit: "days", Direction: "before"},
			want:  now.AddDate(0, 0, -2),
		},
		{
			name:  "weeks after",
			input: dateTimeInputT{Amount: 3, Unit: "week", Direction: "after"},
			want:  now.AddDate(0, 0, 21),
		},
		{
			name:  "months before preserve timezone",
			input: dateTimeInputT{Amount: 1, Unit: "month", Direction: "before"},
			want:  now.AddDate(0, -1, 0),
		},
		{
			name:    "invalid unit",
			input:   dateTimeInputT{Amount: 1, Unit: "fortnight", Direction: "after"},
			wantErr: "unit must be",
		},
		{
			name:    "missing direction",
			input:   dateTimeInputT{Amount: 1, Unit: "day"},
			wantErr: "direction must be",
		},
		{
			name:    "zero amount with offset fields",
			input:   dateTimeInputT{Unit: "day", Direction: "before"},
			wantErr: "amount must be",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := calculateDateTime(now, test.input)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("calculateDateTime() error = %v, want containing %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("calculateDateTime() error = %v", err)
			}
			if !got.Equal(test.want) || got.Location() != test.want.Location() {
				t.Fatalf("calculateDateTime() = %v (%v), want %v (%v)", got, got.Location(), test.want, test.want.Location())
			}
		})
	}
}
