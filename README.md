<!-- DOCGEN:OVERVIEW:START -->

`grove-gemini` is a command-line tool for interacting with Google's Gemini models. It provides infrastructure for managing context caching, executing prompts, and monitoring API usage through local logs and remote GCP metrics.

## Core Mechanisms

**Context Caching Lifecycle**: The tool automates the management of Gemini's context caching feature. It distinguishes between:
*   **Cold Context**: Static files defined in `.grove/cached-context` or via `.grove/rules`. The tool hashes these files; if the content changes, a new cache is created. If the hash matches an existing active cache on Google's servers, that cache is reused.
*   **Hot Context**: Dynamic files passed at runtime which are sent with every request.

**Dual-Layer Observability**:
*   **Local Logging**: Every request made through the CLI is logged to JSONL files in `~/.local/state/grove/logs/gemini/`. This provides immediate access to request parameters, token usage, and response latency.
*   **Remote Analytics**: The `query` command suite interfaces with Google Cloud Monitoring and BigQuery. It aggregates billing data and service-level metrics to provide cost analysis and usage trends across the organization.

**Configuration Resolution**: Project settings (GCP Project ID, Billing datasets) are resolved hierarchically from command-line flags, environment variables, and the `grove.yml` configuration file.

## Features

### Request Execution
*   **`gemini request`**: Sends prompts to Gemini models. It parses `.grove/rules` to automatically assemble context, handles cache creation/lookup transparently, and supports output redirection.
*   **`gemini count-tokens`**: Calculates token counts for input text using the model's tokenizer to estimate usage against context window limits.

### Cache Management
*   **`gemini cache list`**: Displays local and remote status of context caches, including expiration times and token counts.
*   **`gemini cache prune`**: Identifies and removes expired caches from the local state tracking.
*   **`gemini cache tui`**: An interactive terminal interface for managing caches. It provides views for inspecting cache metadata, viewing usage efficiency scores, and manually deleting entries.

### Analytics & Monitoring
*   **`gemini query local`**: Displays a tabular view of recent requests initiated from the local machine, including estimated costs and response times.
*   **`gemini query billing`**: Queries BigQuery exports to show actual accrued costs by SKU (e.g., Input Tokens, Output Tokens).
*   **`gemini query dashboard`**: Launches a terminal dashboard visualizing cost trends, request volume, and SKU breakdowns over time.

## Integrations

*   **`grove-flow`**: Acts as the execution engine for Gemini-based jobs within Flow plans.
*   **`grove cx`**: Uses context rules defined by `cx` to determine which files should be included in the cached context.

<!-- DOCGEN:OVERVIEW:END -->

<!-- DOCGEN:TOC:START -->

See the [documentation](docs/) for detailed usage instructions:
- [Overview](docs/01-overview.md)
- [Configuration](docs/03-configuration.md)

<!-- DOCGEN:TOC:END -->
