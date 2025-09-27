This document provides a detailed reference for all `gemapi` commands, subcommands, and their associated flags.

## `gemapi request`

The `gemapi request` command is the primary interface for making requests to the Gemini API. It integrates with `grove-context` to automatically manage and include codebase context.

**Description**

This command assembles context based on `.grove/rules`, handles caching of large "cold context" files, and sends the final prompt to the Gemini API.

**Flags**

*   `-p, --prompt <string>`: The prompt text to send to the API.
*   `-f, --file <path>`: Path to a file containing the prompt text.
*   `-m, --model <string>`: The Gemini model to use (default: `gemini-2.0-flash`).
*   `-o, --output <path>`: Write the API response to a file instead of standard output.
*   `-w, --workdir <path>`: The working directory for the project context (defaults to the current directory).
*   `--context <path>`: Path to an additional context file to include in the request. Can be specified multiple times.
*   `--regenerate`: Force regeneration of the context from `.grove/rules` before making the request.
*   `--recache`: Force recreation of the Gemini cache for the cold context, ignoring any existing valid cache.
*   `--use-cache <name>`: Use a specific, existing cache by its name (short hash), bypassing automatic cache selection and validation.
*   `--no-cache`: Disable the use of the Gemini Caching API for this request.
*   `--cache-ttl <duration>`: Specify a custom Time-To-Live for a newly created cache (e.g., `1h`, `30m`).
*   `-y, --yes`: Skip the confirmation prompt when creating a new, potentially costly cache.
*   `--temperature <float>`: Controls randomness (0.0-2.0). A value of -1 uses the model's default.
*   `--top-p <float>`: Nucleus sampling (0.0-1.0). A value of -1 uses the model's default.
*   `--top-k <int>`: Top-k sampling. A value of -1 uses the model's default.
*   `--max-output-tokens <int>`: The maximum number of tokens to generate in the response. A value of -1 uses the model's default.

**Examples**

```bash
# Basic request with a text prompt
gemapi request "Explain this project's main entrypoint."

# Request using a prompt file and saving the response to another file
gemapi request -f prompt.md -o response.md -m gemini-1.5-pro-latest

# Force a cache rebuild and set a custom temperature
gemapi request --recache --temperature 0.5 "Analyze the latest version of the code."
```

## `gemapi cache`

The `gemapi cache` command provides a suite of tools for managing local and remote Gemini caches.

### `gemapi cache list`

**Description**

Lists cached contexts, showing both local records and their status on Google's servers.

**Flags**

*   `--local-only`: Display only information from local cache files, without querying the API.
*   `--api-only`: Display only caches found on Google's servers, ignoring local records.

**Example**

```bash
# Show a combined view of local and remote caches
gemapi cache list
```

### `gemapi cache inspect`

**Description**

Shows detailed information about a specific cache, including its creation date, expiration, usage statistics, and the files it contains.

**Usage**

```bash
gemapi cache inspect <cache-name>
```

**Example**

```bash
# Inspect a cache with a specific name
gemapi cache inspect 53f364cda78e82a8
```

### `gemapi cache clear`

**Description**

Deletes caches from Google's servers and updates or removes local tracking files.

**Flags**

*   `--all`: Clear all caches for the current project.
*   `--with-local`: Also remove the local cache tracking file.
*   `--preserve-local`: Delete the remote cache but do not modify the local tracking file.

**Example**

```bash
# Clear a specific cache by name
gemapi cache clear 53f364cda78e82a8

# Clear all caches and remove their local files
gemapi cache clear --all --with-local
```

### `gemapi cache prune`

**Description**

Cleans up expired caches by deleting them from Google's API and updating their local status.

**Flags**

*   `--remove-local`: Also remove the local cache files for expired caches instead of just marking them as expired.

**Example**

```bash
# Prune all expired caches
gemapi cache prune
```

### `gemapi cache tui`

**Description**

Launches a terminal-based user interface for interactively viewing, inspecting, deleting, and analyzing caches.

**Example**

```bash
# Launch the interactive TUI
gemapi cache tui
```

## `gemapi query`

The `gemapi query` command is an observability suite for inspecting API usage, costs, and performance from various data sources.

### `gemapi query local`

**Description**

Queries and displays detailed logs of recent `gemapi` requests made from the local machine. The logs are stored locally and include token counts, cost, latency, and git context.

**Flags**

*   `-H, --hours <int>`: The number of hours to look back (default: 24).
*   `-l, --limit <int>`: The maximum number of requests to display (default: 100).
*   `-m, --model <string>`: Filter requests by model name.
*   `--errors`: Show only failed requests.

**Example**

```bash
# View all requests from the last 3 days
gemapi query local --hours 72

# View only failed requests for the flash model
gemapi query local --errors --model flash
```

### `gemapi query metrics`

**Description**

Fetches request counts and error rates from Google Cloud Monitoring for the Gemini API.

**Flags**

*   `-p, --project-id <string>`: The GCP project ID to query.
*   `-H, --hours <int>`: The number of hours to look back (default: 24).
*   `--debug`: Enable debug output for troubleshooting.

**Example**

```bash
# Query metrics for the last 48 hours
gemapi query metrics --project-id your-gcp-project --hours 48
```

### `gemapi query tokens`

**Description**

Fetches detailed token usage information from Google Cloud Logging, providing a breakdown of prompt tokens, completion tokens, and cache hits.

**Flags**

*   `-p, --project-id <string>`: The GCP project ID to query.
*   `-H, --hours <int>`: The number of hours to look back (default: 24).
*   `--debug`: Enable debug output for troubleshooting.

**Example**

```bash
# Query token usage for the last day
gemapi query tokens --project-id your-gcp-project
```

### `gemapi query billing`

**Description**

Queries cost data directly from a BigQuery billing export table. This provides the most accurate, SKU-level cost information.

**Prerequisite:** You must have a "Detailed usage cost" export to BigQuery configured for your Google Cloud billing account.

**Flags**

*   `-p, --project-id <string>`: The GCP project ID where the BigQuery dataset resides.
*   `-d, --dataset-id <string>`: **(Required)** The BigQuery dataset ID containing the billing export.
*   `-t, --table-id <string>`: **(Required)** The BigQuery table ID for the billing export.
*   `--days <int>`: The number of days to look back (default: 7).

**Example**

```bash
# Query billing data for the last 30 days
gemapi query billing \
  --project-id your-gcp-project \
  --dataset-id your_billing_dataset \
  --table-id gcp_billing_export_v1_XXXXXX_XXXXXX_XXXXXX \
  --days 30
```

## `gemapi count-tokens`

**Description**

Counts the number of tokens in a given text using the Gemini API. This is useful for estimating costs and checking if a prompt fits within a model's context window. Input can be provided as an argument or via standard input.

**Flags**

*   `-m, --model <string>`: The model to use for token counting (default: `gemini-1.5-flash-latest`).

**Examples**

```bash
# Count tokens from a string argument
gemapi count-tokens "How many tokens is this string?"

# Count tokens from a file
cat my_file.txt | gemapi count-tokens -m gemini-1.5-pro-latest
```

## `gemapi config`

**Description**

Manages local configuration settings for `gemapi`.

### `gemapi config set project`

**Description**

Sets the default GCP project ID to be used by the `query` commands. This avoids needing to pass the `--project-id` flag on every call.

**Usage**

```bash
gemapi config set project <PROJECT_ID>
```

**Example**

```bash
gemapi config set project your-gcp-project-12345
```

### `gemapi config get project`

**Description**

Displays the current default GCP project ID and shows the order of precedence used to resolve it (flag, environment variable, saved config).

**Example**

```bash
gemapi config get project
```

## `gemapi version`

**Description**

Prints the version information for the `gemapi` binary, including the version number, commit hash, and build date.

**Flags**

*   `--json`: Output the version information in JSON format.

**Example**

```bash
# Print version information
gemapi version
```