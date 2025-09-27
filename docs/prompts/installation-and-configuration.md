# Installation & Configuration Documentation

You are documenting how to install and configure the `gemapi` CLI tool.

## Task
Create a comprehensive guide covering:
1.  **Installation**: Explain how to install `gemapi` using `grove install gemapi`.
2.  **API Key Configuration**: Detail the primary method of setting the `GEMINI_API_KEY` environment variable. Briefly mention the other resolution methods (from `grove.yml`) as advanced options. Reference `pkg/config/api_key.go`.
3.  **GCP Project Configuration**: Explain why setting a default GCP project is important for the `query` commands. Show how to set it using `gemapi config set project <PROJECT_ID>` and how to verify it with `gemapi config get project`. Reference `cmd/config.go`.

## Output Format
Provide clean, well-formatted Markdown with:
- Clear headings for each setup step.
- Shell command examples in code blocks.
- Brief explanations for why each step is necessary.