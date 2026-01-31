# Command Reference

This document provides a reference for the `grove-gemini` command-line interface, covering all subcommands and their options.

## `grove-gemini request`

Sends a request to the Gemini API. It can use `.grove/rules` to generate and attach file-based context.

| Flag                | Shorthand | Description                                                              |
| ------------------- | --------- | ------------------------------------------------------------------------ |
| `--model`           | `-m`      | The Gemini model to use for the request.                                 |
| `--prompt`          | `-p`      | The prompt text provided as an argument.                                 |
| `--file`            | `-f`      | The path to a file containing the prompt.                                |
| `--output`          | `-o`      | The path to a file to write the response to (defaults to stdout).        |
| `--workdir`         | `-w`      | The working directory for the request (defaults to the current directory). |
| `--context`         |           | A list of additional context files to include.                           |
| `--regenerate`      |           | Forces regeneration of context from `.grove/rules` before the request.   |
| `--recache`         |           | Forces recreation of the Gemini cache, ignoring any existing valid cache.  |
| `--use-cache`       |           | Specifies a cache name (short hash) to use, bypassing automatic selection. |
| `--no-cache`        |           | Disables the use of context caching for this request.                    |
| `--cache-ttl`       |           | Sets a time-to-live duration for a new cache (e.g., `1h`, `30m`).        |
| `--yes`             | `-y`      | Skips the confirmation prompt for potentially costly cache creation.     |
| `--temperature`     |           | Sets the temperature for generation (0.0-2.0).                           |
| `--top-p`           |           | Sets the top-p value for nucleus sampling (0.0-1.0).                     |
| `--top-k`           |           | Sets the top-k value for sampling.                                       |
| `--max-output-tokens` |           | Sets the maximum number of tokens to generate in the response.           |

**Examples**

```bash
# Make a request with an inline prompt
grove-gemini request -p "Explain this Go function."

# Use a prompt from a file and save the response to another file
grove-gemini request -f prompt.md -o response.md

# Force a rebuild of the cold context cache
grove-gemini request --recache "Analyze the latest version of the code."
```

## `grove-gemini cache`

Manages local records and remote state of Gemini API context caches.

### `grove-gemini cache tui`

Launches an interactive terminal user interface (TUI) for browsing, inspecting, and managing caches.

**Example**

```bash
grove-gemini cache tui
```

### `grove-gemini cache list`

Lists cached contexts, showing both local records and their status on Google's servers.

| Flag           | Description                                                            |
| -------------- | ---------------------------------------------------------------------- |
| `--local-only` | Shows only information from local cache files, without querying the API. |
| `--api-only`   | Shows only caches found on Google's API servers for the current project. |

**Example**

```bash
# List all caches with a combined local and remote view
grove-gemini cache list

# List only local cache records
grove-gemini cache list --local-only
```

### `grove-gemini cache inspect`

Shows detailed information about a specific cache, including its creation date, expiration, token count, and the files it contains.

**Example**

```bash
grove-gemini cache inspect <cache-name>
```

### `grove-gemini cache clear`

Deletes caches from Google's servers and updates local records. By default, it marks the local record as cleared but does not delete the file.

| Flag               | Description                                                                    |
| ------------------ | ------------------------------------------------------------------------------ |
| `--all`            | Clears all caches for the current project.                                     |
| `--with-local`     | Also removes the local cache file, instead of just marking it as cleared.      |
| `--preserve-local` | Deletes the remote cache but does not modify the local cache file at all.      |

**Example**

```bash
# Clear a specific cache from the remote API and update the local record
grove-gemini cache clear <cache-name>

# Clear all caches and delete the local files
grove-gemini cache clear --all --with-local
```

### `grove-gemini cache prune`

Identifies expired local cache records, deletes them from Google's servers, and marks them as cleared locally.

| Flag             | Description                                                            |
| ---------------- | ---------------------------------------------------------------------- |
| `--remove-local` | Removes the local cache files for expired caches instead of marking them. |

**Example**

```bash
grove-gemini cache prune
```

## `grove-gemini query`

Provides a suite of commands to inspect Gemini API usage and costs from various sources.

### `grove-gemini query local`

Queries the detailed request logs stored on the local machine, displaying token usage, costs, and performance metrics with a summary.

| Flag      | Shorthand | Description                                       |
| --------- | --------- | ------------------------------------------------- |
| `--hours` | `-H`      | The number of hours to look back in the logs.     |
| `--limit` | `-l`      | The maximum number of log entries to display.     |
| `--model` | `-m`      | Filters the logs to a specific model name.        |
| `--errors`  |           | Shows only requests that resulted in an error.    |

**Example**

```bash
# View all requests from the last 12 hours
grove-gemini query local --hours 12
```

### `grove-gemini query requests`

Displays a table of individual Gemini API requests from local logs with details like timestamp, method, tokens, latency, and status.

| Flag      | Shorthand | Description                                       |
| --------- | --------- | ------------------------------------------------- |
| `--hours` | `-H`      | The number of hours to look back in the logs.     |
| `--limit` | `-l`      | The maximum number of requests to display.        |
| `--model` | `-m`      | Filters the requests to a specific model name.    |
| `--errors`  |           | Shows only requests that resulted in an error.    |

**Example**

```bash
# View the last 20 requests
grove-gemini query requests --limit 20
```

### `grove-gemini query metrics`

Fetches aggregate metrics, such as request counts and error rates, from Google Cloud Monitoring.

| Flag           | Shorthand | Description                                                        |
| -------------- | --------- | ------------------------------------------------------------------ |
| `--project-id` | `-p`      | The GCP project ID to query.                                       |
| `--hours`      | `-H`      | The number of hours to look back for metrics.                      |
| `--debug`      |           | Enables debug output to help diagnose issues with metric filters. |

**Example**

```bash
grove-gemini query metrics --project-id my-gcp-project --hours 48
```

### `grove-gemini query tokens`

Fetches detailed token usage logs from Google Cloud Logging. This provides data on prompt tokens, completion tokens, and cache hits.

| Flag           | Shorthand | Description                              |
| -------------- | --------- | ---------------------------------------- |
| `--project-id` | `-p`      | The GCP project ID to query.             |
| `--hours`      | `-H`      | The number of hours to look back for logs. |
| `--debug`      |           | Enables debug output for troubleshooting. |

**Example**

```bash
grove-gemini query tokens --project-id my-gcp-project
```

### `grove-gemini query billing`

Queries cost data directly from a BigQuery billing export. This requires a BigQuery "Detailed usage cost" export to be configured for your GCP billing account.

| Flag           | Shorthand | Description                                                        |
| -------------- | --------- | ------------------------------------------------------------------ |
| `--project-id` | `-p`      | The GCP project ID where BigQuery is located.                      |
| `--dataset-id` | `-d`      | The BigQuery dataset ID containing the billing table (required).   |
| `--table-id`   | `-t`      | The BigQuery table ID for the billing export (required).           |
| `--days`       |           | The number of days to look back in the billing data.               |

**Example**

```bash
grove-gemini query billing \
  --project-id my-gcp-project \
  --dataset-id my_billing_dataset \
  --table-id gcp_billing_export_v1_XXXX
```

### `grove-gemini query explore`

Explores Cloud Logging to find available logs for the Gemini API service, which can help discover resource types and payload structures.

| Flag           | Shorthand | Description                                    |
| -------------- | --------- | ---------------------------------------------- |
| `--project-id` | `-p`      | The GCP project ID to query.                   |
| `--hours`      | `-H`      | The number of hours to look back for logs.     |
| `--limit`      | `-l`      | The maximum number of log entries to examine.  |

**Example**

```bash
grove-gemini query explore --project-id my-gcp-project --hours 2
```

## `grove-gemini count-tokens`

Counts the number of tokens in a given text using the Gemini API tokenizer. Input can be provided as an argument or via stdin.

| Flag      | Shorthand | Description                                                                    |
| --------- | --------- | ------------------------------------------------------------------------------ |
| `--model` | `-m`      | The model to use for token counting, as different models have different tokenization. |

**Examples**

```bash
# Count tokens from an argument
grove-gemini count-tokens "How many tokens is this string?"

# Count tokens from a file
cat my_file.txt | grove-gemini count-tokens
```

## `grove-gemini config`

Manages the local configuration for `grove-gemini`.

### `grove-gemini config get project`

Displays the resolution order and the currently configured default GCP project ID.

**Example**

```bash
grove-gemini config get project
```

### `grove-gemini config set project`

Sets a default GCP project ID in the local configuration file.

**Example**

```bash
grove-gemini config set project my-gcp-project-id
```

## `grove-gemini version`

Prints version information for the `grove-gemini` binary.

| Flag     | Description                              |
| -------- | ---------------------------------------- |
| `--json` | Outputs the version information in JSON format. |

**Example**

```bash
grove-gemini version
```