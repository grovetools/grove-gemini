# CLI Reference

Complete command reference for `grove-gemini`.

## grove-gemini

<div class="terminal">
Tools for Google's Gemini API

Usage:
  grove-gemini [command]

Available Commands:
  cache        Manage the local cache of Gemini contexts
  completion   Generate the autocompletion script for the specified shell
  config       Manage grove-gemini configuration
  count-tokens Count tokens for a given text using Gemini API
  help         Help about any command
  query        Query Gemini API usage, metrics, and billing data from Google Cloud
  request      Make a request to Gemini API with grove-context support
  version      Print the version information for this binary

Flags:
  -c, --config string   Path to grove.yml config file
  -h, --help            help for grove-gemini
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging

Use "grove-gemini [command] --help" for more information about a command.
</div>

### grove-gemini request

<div class="terminal">
Make a request to the Gemini API using grove-context for automatic hot/cold context management.

This command works similarly to grove-flow's oneshot executor:
- Uses .grove/rules to generate context if available
- Manages cold context caching automatically
- Includes hot context as dynamic files
- Supports custom cache TTL and directives from rules file

Examples:
  # Simple prompt
  grove-gemini request -p "Explain the main function"

  # From file
  grove-gemini request -f prompt.md

  # With specific model and output file
  grove-gemini request -m gemini-2.0-flash -f prompt.md -o response.md

  # Regenerate context before request
  grove-gemini request --regenerate -p "Review the codebase architecture"

  # Force recreation of cache
  grove-gemini request --recache -p "Review the codebase architecture"

  # Use a specific cache
  grove-gemini request --use-cache 53f364cda78e82a8 -p "Review using old context"

  # With custom working directory
  grove-gemini request -w /path/to/project -p "Analyze this project"

Usage:
  grove-gemini request [flags]

Flags:
      --cache-ttl string          Cache TTL (e.g., 1h, 30m, 24h) (default "5m")
      --context strings           Additional context files to include
  -f, --file string               Read prompt from file
  -h, --help                      help for request
      --max-output-tokens int32   Maximum tokens in response (-1 to use default) (default -1)
  -m, --model string              Gemini model to use (default "gemini-2.0-flash")
      --no-cache                  Disable context caching
  -o, --output string             Write response to file instead of stdout
  -p, --prompt string             Prompt text
      --recache                   Force recreation of the Gemini cache
      --regenerate                Regenerate context before request
      --temperature float32       Temperature for randomness (0.0-2.0, -1 to use default) (default -1)
      --top-k int32               Top-k sampling (-1 to use default) (default -1)
      --top-p float32             Top-p nucleus sampling (0.0-1.0, -1 to use default) (default -1)
      --use-cache string          Specify a cache name (short hash) to use for this request, bypassing automatic selection
  -w, --workdir string            Working directory (defaults to current)
  -y, --yes                       Skip cache creation confirmation prompt

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging
</div>

### grove-gemini cache

<div class="terminal">
Provides commands to manage the local cache of Gemini API context data. You can list, inspect, clear, and prune cached items. Use 'grove-gemini cache tui' to launch the interactive interface.

Usage:
  grove-gemini cache [command]

Available Commands:
  clear       Clear caches from Google's servers (default: remote-only)
  inspect     Show detailed information about a specific cache
  list        List all caches with both local and API status
  prune       Mark expired caches as cleared and optionally clean up
  tui         Launch the interactive cache management TUI

Flags:
  -h, --help   help for cache

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging

Use "grove-gemini cache [command] --help" for more information about a command.
</div>

#### grove-gemini cache clear

<div class="terminal">
Clears caches from Google's servers and updates local tracking.
By default, only clears the remote cache and marks the local file as cleared.
Use --with-local to also remove the local cache file.
Use --preserve-local to skip updating the local cache file.

Usage:
  grove-gemini cache clear [cache-name...] | --all [flags]

Flags:
      --all              Clear all caches in the current project
  -h, --help             help for clear
      --preserve-local   Don't update local cache files at all
      --with-local       Also remove local cache files (default: mark as cleared)

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging
</div>

#### grove-gemini cache inspect

<div class="terminal">
Show detailed information about a specific cache

Usage:
  grove-gemini cache inspect [cache-name] [flags]

Flags:
  -h, --help   help for inspect

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging
</div>

#### grove-gemini cache list

<div class="terminal">
List cached contents showing both local storage and Google API status.
By default, shows a combined view of local cache files and their status on Google's servers.
Use --local-only or --api-only to filter the view.

Usage:
  grove-gemini cache list [flags]

Flags:
      --api-only     Show only caches from Google's API servers
  -h, --help         help for list
      --local-only   Show only local cache information

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging
</div>

#### grove-gemini cache prune

<div class="terminal">
Marks expired cache records as cleared and removes them from Google's API.
By default, updates local files to mark them as expired.
Use --remove-local to also remove the local cache files.

Usage:
  grove-gemini cache prune [flags]

Flags:
  -h, --help           help for prune
      --remove-local   Remove local cache files instead of marking them

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging
</div>

#### grove-gemini cache tui

<div class="terminal">
Launch the interactive cache management TUI

Usage:
  grove-gemini cache tui [flags]

Flags:
  -h, --help   help for tui

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging
</div>

### grove-gemini query

<div class="terminal">
Provides commands to query various Google Cloud services for Gemini API metrics, token usage logs, and billing information.

Usage:
  grove-gemini query [command]

Available Commands:
  billing     Query Gemini API billing data from BigQuery
  dashboard   Interactive dashboard for GCP billing data visualization
  explore     Explore available logs for Gemini API
  local       Query local Gemini API logs
  metrics     Query Gemini API metrics from Cloud Monitoring
  requests    Query individual Gemini API requests from local logs
  tokens      Query detailed token usage from Cloud Logging
  tui         Launch an interactive TUI to visualize local query logs

Flags:
  -h, --help   help for query

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging

Use "grove-gemini query [command] --help" for more information about a command.
</div>

#### grove-gemini query billing

<div class="terminal">
Fetches and displays Gemini API billing information from a BigQuery billing export table.

This command requires a BigQuery billing export table containing detailed usage cost data to be enabled for your billing account. 

To set up billing export:
1. Go to the Google Cloud Console Billing section
2. Select your billing account
3. Click "Billing export"
4. Enable "Detailed usage cost" export to BigQuery
5. Note the dataset and table IDs created

Usage:
  grove-gemini query billing [flags]

Flags:
  -d, --dataset-id string   BigQuery dataset ID containing billing export
      --days int            Number of days to look back (default 7)
  -h, --help                help for billing
  -p, --project-id string   GCP project ID
  -t, --table-id string     BigQuery table ID for billing export

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging
</div>

#### grove-gemini query dashboard

<div class="terminal">
Launches an interactive TUI dashboard that visualizes Gemini API billing data from BigQuery.

Features:
- Real-time cost visualization
- Daily/weekly/monthly views
- SKU breakdown table
- Interactive navigation

Usage:
  grove-gemini query dashboard [flags]

Flags:
  -d, --dataset-id string   BigQuery dataset ID containing billing export
      --days int            Number of days to display (default 30)
  -h, --help                help for dashboard
  -p, --project-id string   GCP project ID
  -t, --table-id string     BigQuery table ID for billing export

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging
</div>

#### grove-gemini query explore

<div class="terminal">
Explores Cloud Logging to find what logs are available for the Gemini API service.
This command helps discover the correct resource types, log names, and payload structures.

Usage:
  grove-gemini query explore [flags]

Flags:
  -h, --help                help for explore
  -H, --hours int           Number of hours to look back (default 1)
  -l, --limit int           Maximum number of entries to examine (default 10)
  -p, --project-id string   GCP project ID

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging
</div>

#### grove-gemini query local

<div class="terminal">
Displays locally logged Gemini API requests with token usage, costs, and performance metrics.

Usage:
  grove-gemini query local [flags]

Flags:
      --errors         Show only failed requests
  -h, --help           help for local
  -H, --hours int      Number of hours to look back (default 24)
  -l, --limit int      Maximum number of requests to display (default 100)
  -m, --model string   Filter by model name

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging
</div>

#### grove-gemini query metrics

<div class="terminal">
Fetches and displays Gemini API request counts, error rates, and latency metrics from Google Cloud Monitoring.

Usage:
  grove-gemini query metrics [flags]

Flags:
      --debug               Enable debug output
  -h, --help                help for metrics
  -H, --hours int           Number of hours to look back (default 24)
  -p, --project-id string   GCP project ID

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging
</div>

#### grove-gemini query requests

<div class="terminal">
Displays a table of individual Gemini API requests with details like timestamp, method, tokens, latency, and status.

This command reads from local logs since Google doesn't publish individual Gemini API requests to Cloud Logging.

Usage:
  grove-gemini query requests [flags]

Flags:
      --errors         Show only failed requests
  -h, --help           help for requests
  -H, --hours int      Number of hours to look back (default 1)
  -l, --limit int      Maximum number of requests to display (default 100)
  -m, --model string   Filter by model name

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging
</div>

#### grove-gemini query tokens

<div class="terminal">
Fetches and displays detailed Gemini API token usage information including prompt tokens, completion tokens, cache hits, and estimated costs from Google Cloud Logging.

Usage:
  grove-gemini query tokens [flags]

Flags:
      --debug               Enable debug output
  -h, --help                help for tokens
  -H, --hours int           Number of hours to look back (default 24)
  -p, --project-id string   GCP project ID

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging
</div>

#### grove-gemini query tui

<div class="terminal">
Launch an interactive TUI to visualize local query logs

Usage:
  grove-gemini query tui [flags]

Flags:
  -h, --help   help for tui

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging
</div>

### grove-gemini config

<div class="terminal">
Configure default settings for grove-gemini, such as the default GCP project.

Usage:
  grove-gemini config [command]

Available Commands:
  get         Get configuration values
  set         Set configuration values

Flags:
  -h, --help   help for config

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging

Use "grove-gemini config [command] --help" for more information about a command.
</div>

#### grove-gemini config get

<div class="terminal">
Get configuration values

Usage:
  grove-gemini config get [command]

Available Commands:
  billing     Get the default BigQuery billing configuration
  project     Get the default GCP project

Flags:
  -h, --help   help for get

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging

Use "grove-gemini config get [command] --help" for more information about a command.
</div>

#### grove-gemini config get billing

<div class="terminal">

</div>

#### grove-gemini config get project

<div class="terminal">

</div>

#### grove-gemini config set

<div class="terminal">
Set configuration values

Usage:
  grove-gemini config set [command]

Available Commands:
  billing     Set the default BigQuery billing dataset and table
  project     Set the default GCP project

Flags:
  -h, --help   help for set

Global Flags:
  -c, --config string   Path to grove.yml config file
      --json            Output in JSON format
  -v, --verbose         Enable verbose logging

Use "grove-gemini config set [command] --help" for more information about a command.
</div>

#### grove-gemini config set billing

<div class="terminal">

</div>

#### grove-gemini config set project

<div class="terminal">

</div>

### grove-gemini version

<div class="terminal">
Print the version information for this binary

Usage:
  grove-gemini version [flags]

Flags:
  -h, --help   help for version
      --json   Output version information in JSON format

Global Flags:
  -c, --config string   Path to grove.yml config file
  -v, --verbose         Enable verbose logging
</div>

