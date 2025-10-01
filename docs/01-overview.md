# Grove Gemini

<img src="./images/grove-gemini-readme.svg" width="60%" />

Grove Gemini (`gemapi`) is a command-line interface for Google's Gemini API that supports development workflows by managing codebase context and API interactions. The tool can use the Gemini Caching API for large contexts and includes commands to query local logs and Google Cloud for usage metrics and billing data.

<!-- placeholder for animated gif -->

## Key Features

*   **Context Management**: Builds prompt context by reading a `.grove/rules` file and passing matching files to the API. This process is handled by `grove-context`.

*   **Caching (experimental)**: Caches "cold context" files using the Gemini Caching API via an `@enable-cache` directive. Provides a terminal interface (`gemapi cache tui`) for cache management.

*   **Observability (experimental)**: Includes a `query` command to inspect API usage from multiple sources: `query local` for local logs, `query metrics` for Google Cloud Monitoring, `query tokens` for Google Cloud Logging, and `query billing` for BigQuery exports.

*   **Token Utilities**: Provides a `count-tokens` command to estimate token count and cost for a given text before making an API call.

## How It Works

When `gemapi request` is executed, it first checks for a `.grove/rules` file in the current project. If found, it uses `grove-context` to generate context files based on the defined patterns.

If caching is enabled via an `@enable-cache` directive, the tool attempts to find a valid cache for static "cold context" files. If none is found or if files have changed, it creates a new cache via the Gemini Caching API.

The final request to the Gemini API includes the user's prompt, any "hot context" files, and a reference to the cached content. The tool logs request metrics locally and prints the API response to standard output.

## Ecosystem Integration

Grove Gemini functions as a component of the Grove tool suite and executes other tools in the ecosystem as subprocesses.

*   **Grove Meta-CLI (`grove`)**: Handles installation, updates, and version management of `grove-gemini` and other tools.

*   **Grove Context (`cx`)**: Before executing a request, `gemapi` calls `grove-context` to read `.grove/rules` files and generate file-based context, which is then provided to the Gemini API.

## Installation

Install via the Grove meta-CLI:
```bash
grove install gemini
```

Verify installation:
```bash
gemapi version
```

Requires the `grove` meta-CLI. See the [Grove Installation Guide](https://github.com/mattsolo1/grove-meta/blob/main/docs/02-installation.md) if you don't have it installed.
