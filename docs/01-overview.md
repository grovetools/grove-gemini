# Grove-Gemini: Gemini API Command-Line Interface

The `gemapi` command, provided by the `grove-gemini` repository, is a command-line interface for Google's Gemini API that integrates with the Grove ecosystem. It is designed to streamline API interactions while providing comprehensive usage insights, cost management, and observability features for development workflows.

## Key Features

-   **Context Management**: Leverages `grove-context` to automatically build and manage large code contexts. By using a `.grove/rules` file, `gemapi` can include relevant parts of a codebase in API requests.

-   **Caching**: Caches large, infrequently changing "cold context" files using the Gemini Caching API. This functionality is designed to reduce both latency and token costs for subsequent requests that rely on the same foundational context.

-   **Observability**: Provides a suite of `query` commands to inspect API usage from multiple data sources. Users can analyze local request logs, query performance metrics from Google Cloud Monitoring, track token consumption from Google Cloud Logging, and examine cost data from BigQuery billing exports.

-   **Token Utilities**: Includes a `count-tokens` command to estimate the token count of a prompt. This allows users to check if their input fits within model limits and to approximate costs before making an API call.
