package agent

import "testing"

func TestModelDisallowsStopParameter(t *testing.T) {
	tests := []struct {
		name  string
		agent *Agent
		want  bool
	}{
		{
			name:  "nil agent",
			agent: nil,
			want:  false,
		},
		{
			name: "openai gpt-5",
			agent: &Agent{
				serviceName: LLM_OPENAI,
				modelName:   "gpt-5.5",
			},
			want: true,
		},
		{
			name: "openai gpt-5 mini mixed case",
			agent: &Agent{
				serviceName: LLM_OPENAI,
				modelName:   " GPT-5.4-Mini ",
			},
			want: true,
		},
		{
			name: "openai gpt-4",
			agent: &Agent{
				serviceName: LLM_OPENAI,
				modelName:   "gpt-4.1",
			},
			want: false,
		},
		{
			name: "anthropic model",
			agent: &Agent{
				serviceName: LLM_ANTHROPIC,
				modelName:   "claude-sonnet-4-5",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modelDisallowsStopParameter(tt.agent)
			if got != tt.want {
				t.Fatalf("modelDisallowsStopParameter() = %v, want %v", got, tt.want)
			}
		})
	}
}
