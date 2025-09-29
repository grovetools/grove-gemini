# Configuration

`grove-gemini` is configured through a combination of environment variables, project-level configuration files, and user-specific settings. This guide covers the primary configuration options for setting up API access and default behavior.

## API Key Configuration

The Gemini API key is required for all requests. `gemapi` resolves the key from the following sources, in order of precedence:

1.  **Environment Variable (Recommended)**: The `GEMINI_API_KEY` environment variable. This is the most direct and secure method for providing your API key.
    ```bash
    export GEMINI_API_KEY="your-api-key-here"
    ```

2.  **`api_key_command` in `grove.yml`**: A command specified in your project's `grove.yml` file that, when executed, prints the API key to standard output. This is useful for retrieving keys from a secure vault or keychain.

    **Example `grove.yml`:**
    ```yaml
    name: my-project
    # ...
    gemini:
      api_key_command: "gcloud secrets versions access latest --secret=gemini-api-key"
    ```

3.  **`api_key` in `grove.yml`**: The API key can be set directly in `grove.yml`. This method is convenient but not recommended for repositories that are shared or public.

    **Example `grove.yml`:**
    ```yaml
    name: my-project
    # ...
    gemini:
      api_key: "your-api-key-here" # Use with caution
    ```

If no API key is found in any of these sources, `gemapi` will return an error.

## GCP Project Configuration

For observability commands like `gemapi query metrics` that interact with Google Cloud services, a GCP Project ID must be specified. You can set a default project to avoid passing the `--project-id` flag with every command.

### Setting the Default Project

Use the `gemapi config set project` command to save a default project ID. This setting is stored locally on your machine and applies to all `gemapi` commands.

```bash
gemapi config set project your-gcp-project-id
```

### Viewing the Default Project

To see the current default project and the order in which it's resolved, use `gemapi config get project`.

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

The Gemini model is typically specified on a per-request basis using the `--model` (or `-m`) flag.

```bash
gemapi request -m gemini-1.5-pro-latest "Explain this code."
```

If the `--model` flag is not provided, the `request` command defaults to `gemini-2.0-flash`.

## Context Configuration

`grove-gemini` integrates with `grove-context` to automatically include relevant files from your codebase in API requests. This feature is enabled by creating a `.grove/rules` file in your project's root directory. No additional configuration within `gemapi` is required.

If a `.grove/rules` file is present, `gemapi` will:
1.  Generate a `.grove/context` file containing "hot" (frequently changing) context.
2.  Generate a `.grove/cached-context` file for "cold" (static) context.
3.  Automatically attach these context files to your `gemapi request`.

To disable this behavior for a specific request, you can run the command from a directory that does not contain a `.grove` subdirectory.

## Environment Variables

`gemapi` recognizes the following environment variables:

| Variable           | Description                                                                                              |
| ------------------ | -------------------------------------------------------------------------------------------------------- |
| `GEMINI_API_KEY`   | **(Required)** Your Google Gemini API key.                                                               |
| `GCP_PROJECT_ID`   | Your default Google Cloud Project ID, used by `query` subcommands.                                       |
| `GROVE_DEBUG`      | If set to `1` or `true`, enables debug logging, which saves detailed request payloads to local files.      |

## Configuration Files

`gemapi` uses two primary types of configuration files:

-   **`grove.yml`**: The project-level configuration file located in your repository root. It can contain `gemini`-specific settings for the API key.
-   **`~/.grove/gemini-cache/gcp-config.json`**: A user-specific file that stores the default GCP Project ID set by the `gemapi config set project` command. This file should not be committed to version control.