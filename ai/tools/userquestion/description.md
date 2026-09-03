Ask the user a single question and wait for a typed or choice-based response.

Use this only when the agent is blocked on missing required information and cannot safely proceed without the user's answer.

Input JSON:
{
  "question": "Which environment should I deploy to?",
  "choices": ["prod", "staging", "dev"]
}

The tool returns the user's answer as plain text.
