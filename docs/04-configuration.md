# Configuration

Configuration for `grove-gemini` is managed through environment variables, project-level files, and user-specific settings.

## API and Project Configuration

### API Key

The Gemini API key is required for requests. It is resolved from the following sources, in order of precedence:

1.  **`GEMINI_API_KEY` Environment Variable**:
    ```bash
    export GEMINI_API_KEY="your-api-key"
    ```

2.  **`api_key_command` in `grove.yml`**: A command that prints the key to stdout.
    ```yaml
    # ./{project_root}/grove.yml
    gemini:
      api_key_command: "gcloud secrets versions access latest --secret=gemini-api-key"
    ```

3.  **`api_key` in `grove.yml`**: A static key defined in the file.
    ```yaml
    # ./{project_root}/grove.yml
    gemini:
      api_key: "your-api-key"
    ```

### GCP Project ID

A Google Cloud Project ID is required for `query` subcommands. A default can be configured to avoid using the `--project-id` flag for every command.

To set the default project, use the `config set` command. This writes the ID to a user-specific configuration file.
```bash
grove-gemini config set project your-gcp-project-id
```

The project ID is resolved in the following order:
1.  The `--project-id` command-line flag.
2.  The `GCP_PROJECT_ID` environment variable.
3.  The value stored in the user configuration file.

## Model and Generation Parameters

### Model Selection

The model is specified per-request with the `--model` (or `-m`) flag. If unspecified, `grove-gemini request` defaults to `gemini-2.0-flash`.

```bash
grove-gemini request -m gemini-1.5-pro-latest -p "Explain this function."
```

### Generation Parameters

The following flags can be used with `grove-gemini request` to control the generation process:

| Flag | Description | Default |
| --- | --- | --- |
| `--temperature` | Controls sampling randomness (0.0-2.0). | Model default |
| `--top-p` | Sets the nucleus sampling threshold (0.0-1.0). | Model default |
| `--top-k` | Sets the top-k sampling value. | Model default |
| `--max-output-tokens` | Maximum number of tokens in the response. | Model default |

**Example:**
```bash
grove-gemini request --temperature 0.5 --max-output-tokens 2048 -p "Write a unit test."
```

## Context File Integration

If a `.grove/rules` file exists in the project root, `grove-gemini request` uses `grove-context` to generate context files from the codebase.

-   `.grove/context` is generated for frequently changing ("hot") context.
-   `.grove/cached-context` is generated for less frequently changing ("cold") context.

These files are automatically attached to API requests. The `--regenerate` flag can be used to force an update to the context files before a request is made. This behavior is intrinsic to `grove-context` and does not require `grove-gemini`-specific configuration.

## Environment Variables

| Variable | Description |
| --- | --- |
| `GEMINI_API_KEY` | Google Gemini API key. |
| `GCP_PROJECT_ID` | Default Google Cloud Project ID for `query` subcommands. |

## Configuration Files

-   **`grove.yml`**: A project-level file located in the repository root. It can contain a `gemini` extension to configure the API key.
-   **`~/.grove/gemini-cache/gcp-config.json`**: A user-specific file that stores the default GCP Project ID. This file is managed by `grove-gemini config` commands and should not be added to version control.