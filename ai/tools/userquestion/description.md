Ask the user a single question and wait for a typed or choice-based response.

Use this only when the agent is blocked on missing required information and cannot safely proceed without the user's answer.

Pass this JSON object directly as tool arguments; do not wrap it in an `input` property:
{
  "question": "Which environment should I deploy to?",
  "choices": ["prod", "staging", "dev"]
}

The tool returns the user's answer as plain text.
