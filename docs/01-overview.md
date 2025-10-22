# Grove Gemini

Grove Gemini (`gemapi`) is a command-line interface for Google's Gemini API. It is designed to manage codebase context for API requests and includes commands to query API usage.

<!-- placeholder for animated gif -->

## Key Features

*   **Context Management**: Builds prompt context by reading a `.grove/rules` file and passing matching files to the API. This process is handled by `grove-context`.

*   **Caching (experimental)**: Caches "cold context" files using the Gemini Caching API. This feature is opt-in via an `@enable-cache` directive in the rules file. A terminal interface (`gemapi cache tui`) is available for cache management.

*   **API Usage Querying**: Includes a `query` command to inspect API usage from multiple sources: `query local` for local request logs, `query metrics` for Google Cloud Monitoring, `query tokens` for Google Cloud Logging, and `query billing` for BigQuery exports.

*   **Token Utilities**: Provides a `count-tokens` command to estimate the token count and cost for a given text before making an API call.

## How It Works

When `gemapi request` is executed, it performs the following steps:
1.  It checks for a `.grove/rules` file in the current project. If found, it uses `grove-context` to generate `hot-context` and `cached-context` files based on the defined patterns.
2.  If caching is enabled via an `@enable-cache` directive, the tool attempts to find a valid cache for the static `cached-context` file. If a valid cache is not found or if file contents have changed, it creates a new cache using the Gemini Caching API.
3.  A request is sent to the Gemini API containing the user's prompt, any `hot-context` files, and a reference to the cached content (if used).
4.  The tool logs request metrics locally and prints the API response to standard output.

## Ecosystem Integration

Grove Gemini functions as a component of the Grove tool suite and executes other tools as subprocesses.

*   **Grove Meta-CLI (`grove`)**: Handles installation, updates, and version management of `gemapi` and other tools.

*   **Grove Context (`cx`)**: Before a request, `gemapi` can call `grove-context` to read `.grove/rules` and generate file-based context. This context is then provided to the Gemini API.

### Installation

Install via the Grove meta-CLI:
```bash
grove install gemini
```

Verify installation:
```bash
gemapi version
```

Requires the `grove` meta-CLI. See the [Grove Installation Guide](https://github.com/mattsolo1/grove-meta/blob/main/docs/02-installation.md) if you don't have it installed.