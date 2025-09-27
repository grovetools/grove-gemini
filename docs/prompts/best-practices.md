# Best Practices Documentation

Document recommended best practices for using `gemapi` effectively, securely, and economically.

## Task
Cover important practices including:
1.  **API Key Security**: Recommend using environment variables or a secure command for `api_key_command` over storing plaintext keys in `grove.yml`.
2.  **Cost Management**:
    - Explain the cost implications of creating caches.
    - Advise users to use the caching feature for large, stable parts of their codebase.
    - Recommend using cheaper models like `gemini-2.0-flash` for tasks that don't require the most powerful model.
    - Encourage regular use of `query billing` and `query local` to monitor costs.
3.  **Effective Context Management**:
    - Give tips on writing effective `.grove/rules` to keep the context relevant and reasonably sized.
    - Explain when to use `--regenerate` (after changing rules) vs. `--recache` (after changing code).
4.  **Workflow Tips**:
    - Use `-o` to pipe output to files.
    - Integrate `gemapi` into shell scripts for automated tasks.

## Output Format
- Clear headings for each practice area (e.g., `## Cost Management`).
- Use "Do" and "Don't" lists for clarity.
- Provide short command-line examples to illustrate points.