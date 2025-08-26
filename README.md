# grove-gemini (`gemapi`)

[![CI](https://github.com/mattsolo1/grove-gemini/actions/workflows/ci.yml/badge.svg)](https://github.com/mattsolo1/grove-gemini/actions/workflows/ci.yml)

A powerful command-line interface for Google's Gemini API, with advanced context management, caching, and observability features.

## What is `gemapi`?

`gemapi` is a CLI tool designed to streamline interactions with the Gemini API, particularly within the [Grove](https://github.com/mattsolo1/grove) ecosystem. It leverages `grove-context` to automatically build and manage large code contexts, intelligently caching them using Gemini's Caching API to reduce latency and cost.

Beyond making requests, `gemapi` provides a rich set of observability tools to query local request logs, Google Cloud metrics, token usage logs, and even billing data, giving you a complete picture of your API usage.

## Key Features

-   **Smart Context Management**: Automatically builds and includes context from your codebase using a `.grove/rules` file, just like `grove-flow`.
-   **Advanced Caching**: Caches large "cold context" files using the Gemini Caching API. Features include:
    -   Automatic invalidation when source files change.
    -   Support for TTLs and directives like `@freeze-cache` and `@no-expire` in your rules file.
    -   Confirmation prompts for potentially costly cache creation operations.
-   **Rich Observability**: A comprehensive `query` command to inspect usage from multiple sources:
    -   `query local`: View detailed local logs of all `gemapi` requests.
    -   `query metrics`: Fetch request counts and error rates from Google Cloud Monitoring.
    -   `query tokens`: Analyze token usage from Google Cloud Logging.
    -   `query billing`: Pull cost data directly from BigQuery billing exports.
-   **Token Utilities**: A `count-tokens` command to estimate costs and check if your prompt fits within model limits before making an API call.
-   **Configuration Management**: A simple `config` command to set defaults, such as your GCP project ID.


## Dependencies

`grove-context`

## Installation

```bash
grove install gemapi
```

## Configuration

Before first use, you must configure your environment:

1.  **Set your Gemini API Key:**

    ```bash
    export GEMINI_API_KEY="your-api-key-here"
    ```

2.  **(Optional) Set a default GCP Project:** For the `query` commands that interact with Google Cloud, you can set a default project to avoid passing the `--project-id` flag every time.

    ```bash
    gemapi config set project your-gcp-project-id
    ```

    You can always check the resolution order with `gemapi config get project`.

## Usage

### Making Requests (`gemapi request`)

The `request` command is the core of `gemapi`. It intelligently assembles context, manages caching, and sends your prompt to the Gemini API.

**Basic Request:**
```bash
gemapi request "Explain the main function in main.go"
```

**Using a Prompt File:**
```bash
gemapi request -f prompt.md
```

**Specifying a Model and Output File:**
```bash
gemapi request -m gemini-1.5-pro-latest -f prompt.md -o response.md
```

**Forcing Context Regeneration:**
If you've updated your `.grove/rules`, force a regeneration of the context files before the request.
```bash
gemapi request --regenerate "Review the codebase architecture based on the new rules."
```

**Forcing a Cache Rebuild:**
To ignore the existing cache and create a new one from the current cold context.
```bash
gemapi request --recache "Analyze the latest version of the code."
```

### Observing Usage (`gemapi query`)

The `query` command is a powerful tool for understanding your API usage and costs.

**Query Local Request Logs:**
Get a detailed, table-formatted view of recent requests made from your machine, including token counts, cost, latency, and git context.
```bash
# View requests from the last 24 hours
gemapi query local

# View only failed requests from the last 3 days
gemapi query local --hours 72 --errors
```

**Query Cloud Metrics:**
Fetch request counts and error rates from Google Cloud Monitoring.
```bash
gemapi query metrics --project-id your-project --hours 48
```

**Query Billing Data:**
*Requires a BigQuery billing export to be configured.*
```bash
gemapi query billing \
  --project-id your-project \
  --dataset-id your_billing_dataset \
  --table-id your_billing_table
```

### Managing the Cache (`gemapi cache`)

Interact with the local records of your Gemini caches.

**List Caches:**
```bash
gemapi cache list
```

**Inspect a Specific Cache:**
```bash
gemapi cache inspect <cache-name>
```

**Clear Caches:**
Removes local cache records. Does not delete the cache on Google's servers.
```bash
# Clear a specific cache
gemapi cache clear <cache-name>

# Clear all caches for the current project
gemapi cache clear --all
```

**Prune Expired Caches:**
```bash
gemapi cache prune
```

### Counting Tokens (`gemapi count-tokens`)

Estimate the token count and input cost for a given text.

**From an Argument:**
```bash
gemapi count-tokens "How many tokens is this string?"
```

**From Standard Input:**
```bash
cat my_file.txt | gemapi count-tokens
```

## Development

### Building

To build the `gemapi` binary:
```bash
make build
```

### Testing

Run unit tests:
```bash
make test
```

Run end-to-end tests:
```bash
make test-e2e
```

### Linting

To run the linter:
```bash
make lint
```
```
