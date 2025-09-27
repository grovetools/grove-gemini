# Observability Guide Documentation

You are writing a guide to the observability features of `gemapi`, focusing on the `gemapi query` command.

## Task
For each `gemapi query` subcommand, provide a detailed section that includes:
1.  **`query local`**:
    - What it does (queries local JSONL logs).
    - What information is available (tokens, cost, latency, git context, etc.).
    - Example usage with flags like `--hours` and `--errors`.
2.  **`query metrics`**:
    - What it does (queries Google Cloud Monitoring).
    - What metrics are fetched (request count, error rate).
    - The GCP service it relies on.
    - Example usage.
3.  **`query tokens`**:
    - What it does (queries Google Cloud Logging).
    - What information it provides (detailed token usage from the API).
    - The GCP service it relies on.
    - Example usage.
4.  **`query billing`**:
    - What it does (queries a BigQuery billing export).
    - **Crucially**, explain the prerequisite: users must have a BigQuery billing export configured in their GCP account.
    - What information it provides (SKU-level cost data).
    - Example usage with the required `--dataset-id` and `--table-id` flags.

## Output Format
- An H2 (`##`) heading for each subcommand.
- Clear explanations of the purpose and data source for each command.
- `bash` examples for each.
- Highlight any prerequisites, especially for `query billing`.