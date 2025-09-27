# Command Reference Documentation

You are creating a detailed command reference for the `gemapi` CLI. Analyze the `cmd/` directory to document all commands, subcommands, and flags.

## Task
Document the following commands and their subcommands:
- `gemapi request`: The core command. Detail all flags like `-m`, `-p`, `-f`, `--regenerate`, `--recache`, `--use-cache`, and the generation parameters (`--temperature`, etc.).
- `gemapi cache`: The cache management suite.
  - `list`: Explain the different views (`--local-only`, `--api-only`).
  - `inspect`: How to view details of a single cache.
  - `clear`: How to remove caches from GCP and locally.
  - `prune`: How to clean up expired caches.
  - `tui`: How to launch the interactive TUI.
- `gemapi query`: The observability suite.
  - `local`: Querying local request logs.
  - `metrics`: Querying Cloud Monitoring.
  - `tokens`: Querying Cloud Logging for token usage.
  - `billing`: Querying BigQuery for cost data. Mention the setup prerequisites.
- `gemapi count-tokens`: The token counting utility.
- `gemapi config`: The configuration command (`set project`, `get project`).
- `gemapi version`: How to check the version.

## Output Format
- Use a clear hierarchical structure with H2 (`##`) for top-level commands and H3 (`###`) for subcommands.
- For each command/subcommand, provide a brief description.
- Use tables or bullet points to list all flags and their descriptions.
- Include simple `bash` examples for each command.