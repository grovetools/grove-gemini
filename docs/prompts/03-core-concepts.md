# Core Concepts Documentation

You are documenting the fundamental concepts of the `gemapi` CLI tool.

## Task
Identify and explain the core concepts that a user must understand to use `gemapi` effectively. Focus on:
1.  **Integration with `grove-context`**: Explain how `gemapi` uses the `.grove/rules` file to partition a project's codebase into "hot" and "cold" context.
2.  **Hot vs. Cold Context**:
    - **Cold Context** (`.grove/cached-context`): Large, infrequently changing files that are ideal for the Gemini Caching API.
    - **Hot Context** (`.grove/context`): Smaller, frequently changing files that are sent with every request.
3.  **The Caching Layer**: Explain at a high level that `gemapi` can store the "cold context" in the Gemini Caching API to save on tokens for subsequent requests. Mention that this is an opt-in feature.
4.  **Observability Data Sources**: Briefly explain the different places `gemapi` pulls data from for its `query` commands (local logs, Cloud Monitoring, Cloud Logging, BigQuery).

## Output Format
- A section for each core concept (e.g., `## Hot vs. Cold Context`).
- Clear explanations of what each concept is and why it matters for the user.
- Simple diagrams or flow descriptions where helpful.