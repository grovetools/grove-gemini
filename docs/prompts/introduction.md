# Introduction Documentation

You are an expert technical writer creating an introduction for the `gemapi` CLI tool.

## Task
Write a clear, engaging introduction that:
- Explains that `gemapi` is a CLI for Google's Gemini API and is part of the Grove ecosystem.
- Highlights its three core feature pillars:
    1.  **Smart Context Management**: Integration with `grove-context` via `.grove/rules` to automatically manage and include codebase context in requests.
    2.  **Advanced Caching**: Caching large "cold context" files using the Gemini Caching API to reduce cost and latency.
    3.  **Rich Observability**: A suite of `query` commands to inspect usage, costs, and performance from both local logs and Google Cloud services.
- Briefly mentions other utilities like `count-tokens`.
- Identifies the target audience (developers using the Gemini API, especially within the Grove ecosystem).

## Output Format
Provide clean, well-formatted Markdown with:
- A clear main heading.
- A "Key Features" section with a bulleted list.