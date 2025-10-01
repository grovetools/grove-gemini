# Configuration

`gemapi` is configured through environment variables, a project-level `grove.yml` file, and user-specific settings.

## API Key Configuration

The Gemini API key is required for requests. `gemapi` resolves the key from the following sources in order of precedence:

1.  **Environment Variable**: The `GEMINI_API_KEY` environment variable.
    ```bash
    export GEMINI_API_KEY="your-api-key-here"
    ```

2.  **`api_key_command` in `grove.yml`**: A command in the project's `grove.yml` file that prints the API key to standard output when executed.
    ```yaml
    # grove.yml
    gemini:
      api_key_command: "gcloud secrets versions access latest --secret=gemini-api-key"
    ```

3.  **`api_key` in `grove.yml`**: The API key set directly in `grove.yml`.
    ```yaml
    # grove.yml
    gemini:
      api_key: "your-api-key-here"
    ```

If an API key is not found in any of these sources, `gemapi` will return an error.

## GCP Project Configuration

Commands such as `gemapi query metrics` require a Google Cloud Project ID. A default project can be set to avoid passing the `--project-id` flag with each command.

### Setting the Default Project

The `gemapi config set project` command saves a default project ID to a local configuration file.

```bash
gemapi config set project your-gcp-project-id
```

### Viewing the Default Project

The `gemapi config get project` command shows the current default project and the order in which it is resolved.

```bash
gemapi config get project
```

**Example Output:**
```
GCP Project Resolution Order:
1. Command flag: --project-id
2. Environment variable GCP_PROJECT_ID: (not set)
3. Saved configuration: your-gcp-project-id

Current default project: your-gcp-project-id
```

## Model Selection

The Gemini model is specified per-request using the `--model` (or `-m`) flag.

```bash
gemapi request -m gemini-1.5-pro-latest "Explain this code."
```

If the `--model` flag is not provided, the `request` command defaults to `gemini-2.0-flash`.

## Context Configuration

`gemapi` uses `grove-context` to include files from the codebase in API requests. This is enabled by creating a `.grove/rules` file in the project's root directory. No additional `gemapi` configuration is required.

If a `.grove/rules` file is present, `gemapi` will:
1.  Generate a `.grove/context` file for "hot" context.
2.  Generate a `.grove/cached-context` file for "cold" context.
3.  Attach these context files to the `gemapi request`.

This behavior can be disabled for a specific request by running the command from a directory that does not contain a `.grove` subdirectory.

## Environment Variables

`gemapi` recognizes the following environment variables:

| Variable | Description |
| --- | --- |
| `GEMINI_API_KEY` | Your Google Gemini API key. |
| `GCP_PROJECT_ID` | Your default Google Cloud Project ID, used by `query` subcommands. |
| `GROVE_DEBUG` | If set to `1` or `true`, enables logging that saves request payloads to local files. |

## Configuration Files

`gemapi` uses two primary types of configuration files:

-   **`grove.yml`**: The project-level configuration file in the repository root. It can contain `gemini`-specific settings for the API key.
-   **`~/.grove/gemini-cache/gcp-config.json`**: A user-specific file that stores the default GCP Project ID set by `gemapi config set project`. This file should not be committed to version control.