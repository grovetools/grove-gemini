# Installation & Configuration

This guide covers the installation and essential configuration steps required to start using `gemapi`.

## Installation

The recommended way to install `grove-gemini` is by using the `grove` meta-CLI, which manages all tools within the Grove ecosystem.

### Prerequisites

You must have the `grove` meta-CLI installed. If you don't, please follow the **[Grove Ecosystem Installation Guide](https://github.com/mattsolo1/grove-meta/blob/main/docs/02-installation.md)** first. This guide also covers essential setup like configuring your `PATH`.

### Install Command

Once the `grove` CLI is set up, you can install `grove-gemini` with a single command:

```bash
grove install gemapi
```

### Verifying the Installation

To confirm that the tool was installed correctly, you can run its `version` command:

```bash
gemapi version
```

### Building from Source

For contributors, the recommended way to work with the source code is to clone the entire Grove ecosystem monorepo. Please refer to the **[Building from Source](https://github.com/mattsolo1/grove-meta/blob/main/docs/02-installation.md#3-building-from-source-for-contributors)** section in the main installation guide for details.

## API Key Configuration

`gemapi` requires a Google Gemini API key to make requests. The key can be configured in several ways, with the following order of precedence:

1.  An environment variable (`GEMINI_API_KEY`)
2.  The output of a shell command defined in `grove.yml` (`api_key_command`)
3.  A direct value in `grove.yml` (`api_key`)

### Recommended Method: Environment Variable

The most direct and secure method is to set the `GEMINI_API_KEY` environment variable.

```bash
export GEMINI_API_KEY="your-api-key-here"
```

To make this setting permanent, add the `export` command to your shell's profile file (e.g., `~/.bashrc`, `~/.zshrc`, or `~/.profile`).

### Alternative Methods (Advanced)

For project-specific configurations, you can define the API key within your `grove.yml` file. This is useful for teams that need to share a common setup.

**Using a command:**
This method is useful for fetching keys from a secure vault.

```yaml
# grove.yml
name: my-project
description: A project using gemapi.

gemini:
  api_key_command: "gcloud auth print-access-token" # Example command
```

**Using a direct value:**
Storing plaintext keys in configuration is generally discouraged. This method should only be used in secure environments.

```yaml
# grove.yml
name: my-project
description: A project using gemapi.

gemini:
  # Not recommended for production use
  api_key: "your-plaintext-api-key"
```

## GCP Project Configuration

Several `gemapi query` subcommands (`metrics`, `tokens`, `billing`) interact with Google Cloud Platform services to fetch observability data. These commands require a GCP Project ID to function correctly.

While you can pass the project ID with the `--project-id` flag on every call, setting a default project streamlines the process.

### Set a Default Project

Use the `gemapi config set project` command to store a default GCP Project ID in your local `gemapi` configuration.

```bash
gemapi config set project your-gcp-project-id
```

This configuration is saved locally and will be used as a fallback when no project ID is provided via a flag or environment variable.

### Verify Project Configuration

To check the currently configured project and understand the resolution order, use the `gemapi config get project` command.

```bash
gemapi config get project
```

This command displays the sources `gemapi` will check for a project ID, in order of precedence:

1.  The `--project-id` command-line flag.
2.  The `GCP_PROJECT_ID` environment variable.
3.  The value saved in the `gemapi` configuration file.