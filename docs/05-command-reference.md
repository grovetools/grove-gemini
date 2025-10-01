# Command Reference

This document provides a reference for the `gemapi` command-line interface, covering all subcommands and their options.

## `gemapi request`

Sends a request to the Gemini API. It can use `.grove/rules` to generate and attach file-based context.

| Flag | Shorthand | Description |
| --- | --- | --- |
| `--model` | `-m` | The Gemini model to use for the request. |
| `--prompt` | `-p` | The prompt text provided as an argument. |
| `--file` | `-f` | The path to a file containing the prompt. |
| `--output` | `-o` | The path to a file to write the response to (defaults to stdout). |
| `--workdir` | `-w` | The working directory for the request (defaults to the current directory). |
| `--context` | | A list of additional context files to include. |
| `--regenerate` | | Forces the regeneration of context from `.grove/rules` before the request. |
| `--recache` | | Forces the recreation of the Gemini cache, ignoring any existing valid cache. |
| `--use-cache` | | Specifies a cache name (short hash) to use, bypassing automatic selection. |
| `--no-cache` | | Disables the use of context caching for this request. |
| `--cache-ttl` | | Sets a time-to-live duration for a new cache (e.g., `1h`, `30m`). |
| `--yes` | `-y` | Skips the confirmation prompt for potentially costly cache creation. |
| `--temperature` | | Sets the temperature for generation (0.0-2.0). |
| `--top-p` | | Sets the top-p value for nucleus sampling (0.0-1.0). |
| `--top-k` | | Sets the top-k value for sampling. |
| `--max-output-tokens` | | Sets the maximum number of tokens to generate in the response. |

**Examples**

```bash
# Make a request with an inline prompt
gemapi request -p "Explain this Go function."

# Use a prompt from a file and save the response to another file
gemapi request -f prompt.md -o response.md

# Force a rebuild of the cold context cache
gemapi request --recache "Analyze the latest version of the code."
```

## `gemapi cache`

Manages local records and remote state of Gemini API context caches.

### `gemapi cache tui`

Launches an interactive terminal user interface (TUI) for browsing, inspecting, and managing caches.

**Example**

```bash
gemapi cache tui
```

### `gemapi cache list`

Lists cached contexts, showing both local records and their status on Google's servers.

| Flag | Description |
| --- | --- |
| `--local-only` | Shows only information from local cache files, without querying the API. |
| `--api-only` | Shows only caches found on Google's API servers for the current project. |

**Example**

```bash
# List all caches with a combined local and remote view
gemapi cache list

# List only local cache records
gemapi cache list --local-only
```

### `gemapi cache inspect`

Shows detailed information about a specific cache, including its creation date, expiration, token count, and the files it contains.

**Example**

```bash
gemapi cache inspect <cache-name>
```

### `gemapi cache clear`

Deletes caches from Google's servers and updates local records. By default, it marks the local record as cleared but does not delete the file.

| Flag | Description |
| --- | --- |
| `--all` | Clears all caches for the current project. |
| `--with-local` | Also removes the local cache file, instead of just marking it as cleared. |
| `--preserve-local` | Deletes the remote cache but does not modify the local cache file at all. |

**Example**

```bash
# Clear a specific cache from the remote API and update the local record
gemapi cache clear <cache-name>

# Clear all caches and delete the local files
gemapi cache clear --all --with-local
```

### `gemapi cache prune`

Identifies expired local cache records, deletes them from Google's servers, and marks them as cleared locally.

| Flag | Description |
| --- | --- |
| `--remove-local` | Removes the local cache files for expired caches instead of marking them. |

**Example**

```bash
gemapi cache prune
```

## `gemapi query`

Provides a suite of commands to inspect Gemini API usage and costs from various sources.

### `gemapi query local`

Queries the detailed request logs stored on the local machine.

| Flag | Shorthand | Description |
| --- | --- | --- |
| `--hours` | `-H` | The number of hours to look back in the logs. |
| `--limit` | `-l` | The maximum number of log entries to display. |
| `--model` | `-m` | Filters the logs to a specific model name. |
| `--errors` | | Shows only requests that resulted in an error. |

**Example**

```bash
# View all requests from the last 12 hours
gemapi query local --hours 12
```

### `gemapi query metrics`

Fetches aggregate metrics, such as request counts and error rates, from Google Cloud Monitoring.

| Flag | Shorthand | Description |
| --- | --- | --- |
| `--project-id` | `-p` | The GCP project ID to query. |
| `--hours` | `-H` | The number of hours to look back for metrics. |
| `--debug` | | Enables debug output to help diagnose issues with metric filters. |

**Example**

```bash
gemapi query metrics --project-id my-gcp-project --hours 48
```

### `gemapi query tokens`

Fetches detailed token usage logs from Google Cloud Logging. This provides data on prompt tokens, completion tokens, and cache hits.

| Flag | Shorthand | Description |
| --- | --- | --- |
| `--project-id` | `-p` | The GCP project ID to query. |
| `--hours` | `-H` | The number of hours to look back for logs. |
| `--debug` | | Enables debug output for troubleshooting. |

**Example**

```bash
gemapi query tokens --project-id my-gcp-project
```

### `gemapi query billing`

Queries cost data directly from a BigQuery billing export. This requires a BigQuery billing export to be configured for your GCP billing account.

| Flag | Shorthand | Description |
| --- | --- | --- |
| `--project-id` | `-p` | The GCP project ID where BigQuery is located. |
| `--dataset-id` | `-d` | The BigQuery dataset ID containing the billing table (required). |
| `--table-id` | `-t` | The BigQuery table ID for the billing export (required). |
| `--days` | | The number of days to look back in the billing data. |

**Example**

```bash
gemapi query billing \
  --project-id my-gcp-project \
  --dataset-id my_billing_dataset \
  --table-id gcp_billing_export_v1_XXXX
```

## `gemapi count-tokens`

Counts the number of tokens in a given text using the Gemini API tokenizer. Input can be provided as an argument or via stdin.

| Flag | Shorthand | Description |
| --- | --- | --- |
| `--model` | `-m` | The model to use for token counting, as different models have different tokenization. |

**Examples**

```bash
# Count tokens from an argument
gemapi count-tokens "How many tokens is this string?"

# Count tokens from a file
cat my_file.txt | gemapi count-tokens
```

## `gemapi config`

Manages the local configuration for `gemapi`.

### `gemapi config get project`

Displays the resolution order and the currently configured default GCP project ID.

**Example**

```bash
gemapi config get project
```

### `gemapi config set project`

Sets a default GCP project ID in the local configuration file.

**Example**

```bash
gemapi config set project my-gcp-project-id
```

## `gemapi version`

Prints version information for the `gemapi` binary.

| Flag | Description |
| --- | --- |
| `--json` | Outputs the version information in JSON format. |

**Example**

```bash
gemapi version
```
