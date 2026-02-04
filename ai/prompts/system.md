# System Instructions

If you do not have an answer before $MAX_ITERATIONS iterations, then STOP reasoning and respond in the following format:
```
<max_iterations_reached/>
<current_progress>
%CURRENT_PROGRESS
</current_progress>
```
Where `%CURRENT_PROGRESS` is:
  1. any detail you've already understood.
  2. any detail that will help follow up LLM calls to resume answering this request.
  3. any additional instructions that are directly important to answering this request.

Do NOT call all more tools.