# Usage Patterns Documentation

Document common usage patterns and practical examples for the `gemapi` CLI.

## Task
Provide practical examples for a variety of scenarios. Reference the `README.md` for existing examples and expand on them. Include patterns such as:
1.  **Simple Q&A**: Basic text generation from a prompt string or file.
2.  **Codebase Analysis**: Using a `.grove/rules` file to ask high-level questions about a project's architecture.
3.  **Iterative Development with Caching**: Show a workflow where a user makes multiple requests about a large codebase, benefiting from the cache to reduce cost and latency on each subsequent request.
4.  **Cost and Usage Monitoring**: Demonstrate a workflow for checking local usage with `query local` and then getting a cost breakdown from `query billing`.
5.  **Pre-flight Checks**: Show how to use `gemapi count-tokens` to estimate the size of a prompt before sending it.

## Output Format
- Descriptive headings for each pattern.
- Step-by-step instructions where appropriate.
- `bash` command examples with explanations.