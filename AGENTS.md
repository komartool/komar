# Rules for AI agents

1. NEVER write comments in code by yourself. Add comments only if the user asks for them directly.
2. Before you finish any task that changed code or build files, run `make lint` from the repo root and fix every reported problem. A task is not done while `make lint` fails. Never weaken a linter config, disable a check or add a suppression to make it pass, unless the user asks for it directly.
