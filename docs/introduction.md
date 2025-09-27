# Introduction to grove-gemini (gemapi)

`grove-gemini` is a command-line interface for Google's Gemini API, designed to streamline interactions and provide comprehensive usage insights, particularly within the Grove ecosystem. It integrates context management, caching, and observability features to create an efficient development workflow.

The tool is intended for developers who use the Gemini API and want to automate the inclusion of codebase context, manage costs through caching, and monitor their API consumption across various services.

## Key Features

-   **Smart Context Management**: Leverages `grove-context` to automatically build and manage large code contexts. By using a `.grove/rules` file, `gemapi` can intelligently include relevant parts of a codebase in API requests.

-   **Advanced Caching**: Caches large, infrequently changing "cold context" files using the Gemini Caching API. This functionality is designed to reduce both latency and token costs for subsequent requests that rely on the same foundational context.

-   **Rich Observability**: Provides a suite of `query` commands to inspect API usage from multiple data sources. Users can analyze local request logs, query performance metrics from Google Cloud Monitoring, track token consumption from Google Cloud Logging, and examine cost data from BigQuery billing exports.

-   **Token Utilities**: Includes a `count-tokens` command to estimate the token count of a prompt. This allows users to check if their input fits within model limits and to approximate costs before making an API call.